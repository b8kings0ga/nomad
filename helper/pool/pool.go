// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package pool

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"
	"sync"
	"sync/atomic"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	msgpackrpc "github.com/hashicorp/net-rpc-msgpackrpc/v2"
	"github.com/hashicorp/nomad/helper"
	"github.com/hashicorp/nomad/helper/tlsutil"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/yamux"
)

// ErrRPCNotSent identifies failures before the RPC method or arguments were sent.
// Callers may retry these failures without replaying an ambiguous mutation.
var ErrRPCNotSent = errors.New("RPC request was not sent")

// ErrPoolShutdown identifies a permanently closed pool. Retrying against another
// address cannot recover it; the owner must create a new pool instead.
var ErrPoolShutdown = errors.New("rpc error: shutdown")

// defaultDialTimeout is the fallback timeout used when a ConnPool is
// constructed without an explicit dial timeout.
const defaultDialTimeout = 10 * time.Second

// NewClientCodec returns a new rpc.ClientCodec to be used to make RPC calls.
func NewClientCodec(conn io.ReadWriteCloser) rpc.ClientCodec {
	return msgpackrpc.NewCodecFromHandle(true, true, conn, structs.MsgpackHandle)
}

// NewServerCodec returns a new rpc.ServerCodec to be used to handle RPCs.
func NewServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return msgpackrpc.NewCodecFromHandle(true, true, conn, structs.MsgpackHandle)
}

// NewServerCodecContext cancels blocking reads when the peer closes its RPC
// stream. The context is attached after decoding and never crosses the wire.
func NewServerCodecContext(ctx context.Context, conn io.ReadWriteCloser) rpc.ServerCodec {
	ctx, cancel := context.WithCancel(ctx)
	return &queryContextCodec{ServerCodec: NewServerCodec(conn), ctx: ctx, cancel: cancel}
}

type queryContextCodec struct {
	rpc.ServerCodec
	ctx    context.Context
	cancel context.CancelFunc
}

func (c *queryContextCodec) ReadRequestHeader(request *rpc.Request) error {
	err := c.ServerCodec.ReadRequestHeader(request)
	if err != nil {
		c.cancel()
	}
	return err
}

func (c *queryContextCodec) ReadRequestBody(args any) error {
	if err := c.ServerCodec.ReadRequestBody(args); err != nil {
		c.cancel()
		return err
	}
	if query, ok := args.(interface{ SetQueryContext(context.Context) }); ok {
		query.SetQueryContext(c.ctx)
	}
	return nil
}

func (c *queryContextCodec) Close() error {
	c.cancel()
	return c.ServerCodec.Close()
}

// streamClient is used to wrap a stream with an RPC client
type StreamClient struct {
	stream net.Conn
	codec  rpc.ClientCodec
}

func (sc *StreamClient) Close() {
	sc.stream.Close()
	sc.codec.Close()
}

// Conn is a pooled connection to a Nomad server
type Conn struct {
	refCount    int32
	shouldClose int32

	addr     net.Addr
	session  *yamux.Session
	lastUsed atomic.Pointer[time.Time]

	pool          *ConnPool
	tlsGeneration uint64

	clients    *list.List
	clientLock sync.Mutex
}

// markForUse does all the bookkeeping required to ready a connection for use,
// and ensure that active connections don't get reaped.
func (c *Conn) markForUse() {
	now := time.Now()
	c.lastUsed.Store(&now)
	atomic.AddInt32(&c.refCount, 1)
}

// releaseUse is the complement of `markForUse`, to free up the reference count
func (c *Conn) releaseUse() {
	refCount := atomic.AddInt32(&c.refCount, -1)
	if refCount == 0 && atomic.LoadInt32(&c.shouldClose) == 1 {
		c.Close()
	}
}

func (c *Conn) Close() error {
	return c.session.Close()
}

// getClient is used to get a cached or new client
func (c *Conn) getRPCClient() (*StreamClient, error) {
	// Check for cached client
	c.clientLock.Lock()
	front := c.clients.Front()
	if front != nil {
		c.clients.Remove(front)
	}
	c.clientLock.Unlock()
	if front != nil {
		return front.Value.(*StreamClient), nil
	}

	// Open a new session
	stream, err := c.session.Open()
	if err != nil {
		return nil, err
	}

	if _, err := stream.Write([]byte{byte(RpcNomad)}); err != nil {
		stream.Close()
		return nil, err
	}

	// Create a client codec
	codec := NewClientCodec(stream)

	// Return a new stream client
	sc := &StreamClient{
		stream: stream,
		codec:  codec,
	}
	return sc, nil
}

// returnClient is used when done with a stream
// to allow re-use by a future RPC
func (c *Conn) returnClient(client *StreamClient) {
	didSave := false
	c.clientLock.Lock()
	if c.clients.Len() < c.pool.maxStreams && atomic.LoadInt32(&c.shouldClose) == 0 {
		c.clients.PushFront(client)
		didSave = true

		// If this is a Yamux stream, shrink the internal buffers so that
		// we can GC the idle memory
		if ys, ok := client.stream.(*yamux.Stream); ok {
			ys.Shrink()
		}
	}
	c.clientLock.Unlock()
	if !didSave {
		client.Close()
	}
}

func (c *Conn) IsClosed() bool {
	return c.session.IsClosed()
}

func (c *Conn) AcceptStream() (net.Conn, error) {
	s, err := c.session.AcceptStream()
	if err != nil {
		return nil, err
	}

	c.markForUse()
	return &incomingStream{
		Stream: s,
		parent: c,
	}, nil
}

// incomingStream wraps yamux.Stream but frees the underlying yamux.Session
// when closed
type incomingStream struct {
	*yamux.Stream

	parent    *Conn
	closeOnce sync.Once
	closeErr  error
}

func (s *incomingStream) Close() error {
	s.closeOnce.Do(func() {
		s.closeErr = s.Stream.Close()
		// Release the parent exactly once, even when closing the stream fails.
		s.parent.releaseUse()
	})
	return s.closeErr
}

// ConnPool is used to maintain a connection pool to other
// Nomad servers. This is used to reduce the latency of
// RPC requests between servers. It is only used to pool
// connections in the rpcNomad mode. Raft connections
// are pooled separately.
type ConnPool struct {
	sync.Mutex

	// logger is the logger to be used
	logger *log.Logger

	// The maximum time to keep a connection open
	maxTime time.Duration

	// The maximum number of open streams to keep
	maxStreams int

	// dialTimeout is the timeout used when establishing new outbound
	// connections. A zero value falls back to defaultDialTimeout.
	dialTimeout time.Duration

	// Pool maps an address to a open connection
	pool map[string]*Conn

	// limiter is used to throttle the number of connect attempts
	// to a given address. The first thread will attempt a connection
	// and put a channel in here, which all other threads will wait
	// on to close.
	limiter map[string]chan struct{}

	// TLS wrapper
	tlsWrap       tlsutil.RegionWrapper
	tlsGeneration uint64

	// yamuxCfg is used for setup rpc client
	yamuxCfg *yamux.Config

	// Used to indicate the pool is shutdown
	shutdown   bool
	shutdownCh chan struct{}

	// connListener is used to notify a potential listener of a new connection
	// being made.
	connListener chan<- *Conn
}

// NewPool is used to make a new connection pool
// Maintain at most one connection per host, for up to maxTime.
// Set maxTime to 0 to disable reaping. maxStreams is used to control
// the number of idle streams allowed.
// dialTimeout is the timeout used when establishing new outbound
// connections. A zero value falls back to defaultDialTimeout.
// If TLS settings are provided outgoing connections use TLS.
func NewPool(
	logger hclog.Logger, maxTime time.Duration, maxStreams int, tlsWrap tlsutil.RegionWrapper,
	yamuxCfg *yamux.Config, dialTimeout time.Duration,
) *ConnPool {
	if dialTimeout <= 0 {
		dialTimeout = defaultDialTimeout
	}
	pool := &ConnPool{
		logger:      logger.StandardLogger(&hclog.StandardLoggerOptions{InferLevels: true}),
		maxTime:     maxTime,
		maxStreams:  maxStreams,
		dialTimeout: dialTimeout,
		pool:        make(map[string]*Conn),
		limiter:     make(map[string]chan struct{}),
		tlsWrap:     tlsWrap,
		shutdownCh:  make(chan struct{}),
	}

	// The passed yamux config is the shared server object, so we do not want to
	// modify it directly. Instead, clone it and set up the logger to avoid data
	// races.
	//
	// Performing this work here avoids doing this per new connection within
	// getNewConn.
	poolMUXCfg := yamuxCfg.Clone()
	poolMUXCfg.LogOutput = nil
	poolMUXCfg.Logger = pool.logger

	pool.yamuxCfg = poolMUXCfg

	if maxTime > 0 {
		go pool.reap()
	}
	return pool
}

// Shutdown is used to close the connection pool
func (p *ConnPool) Shutdown() error {
	p.Lock()
	defer p.Unlock()

	for _, conn := range p.pool {
		conn.Close()
	}
	p.pool = make(map[string]*Conn)

	if p.shutdown {
		return nil
	}

	if p.connListener != nil {
		close(p.connListener)
		p.connListener = nil
	}

	p.shutdown = true
	close(p.shutdownCh)
	return nil
}

// ReloadTLS reloads TLS configuration on the fly
func (p *ConnPool) ReloadTLS(tlsWrap tlsutil.RegionWrapper) {
	p.Lock()
	defer p.Unlock()

	oldPool := p.pool
	for _, conn := range oldPool {
		conn.Close()
	}
	p.pool = make(map[string]*Conn)
	p.tlsWrap = tlsWrap
	p.tlsGeneration++
}

// SetConnListener is used to listen to new connections being made. The
// channel will be closed when the conn pool is closed or a new listener is set.
func (p *ConnPool) SetConnListener(l chan<- *Conn) {
	p.Lock()
	defer p.Unlock()

	// Close the old listener
	if p.connListener != nil {
		close(p.connListener)
	}

	// Store the new listener
	p.connListener = l
}

// Acquire is used to get a connection that is
// pooled or to return a new connection
func (p *ConnPool) acquire(region string, addr net.Addr) (*Conn, error) {
	return p.acquireContext(context.Background(), region, addr)
}

// acquireContext is acquire with cancellation for both dialing and callers
// waiting behind another dial attempt.
func (p *ConnPool) acquireContext(ctx context.Context, region string, addr net.Addr) (*Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Check to see if there's a pooled connection available. This is up
	// here since it should the vastly more common case than the rest
	// of the code here.
	p.Lock()
	if p.shutdown {
		p.Unlock()
		return nil, ErrPoolShutdown
	}
	c := p.pool[addr.String()]
	if c != nil {
		c.markForUse()
		p.Unlock()
		return c, nil
	}

	// If not (while we are still locked), set up the throttling structure
	// for this address, which will make everyone else wait until our
	// attempt is done.
	var wait chan struct{}
	var ok bool
	if wait, ok = p.limiter[addr.String()]; !ok {
		wait = make(chan struct{})
		p.limiter[addr.String()] = wait
	}
	isLeadThread := !ok
	p.Unlock()

	// If we are the lead thread, make the new connection and then wake
	// everybody else up to see if we got it.
	if isLeadThread {
		c, err := p.getNewConnContext(ctx, region, addr)
		p.Lock()
		delete(p.limiter, addr.String())
		close(wait)
		if err != nil {
			p.Unlock()
			return nil, err
		}
		// Shutdown may have completed while the connection was being set up.
		// Do not repopulate a pool whose reaper and owner have already stopped.
		if p.shutdown {
			p.Unlock()
			c.Close()
			return nil, ErrPoolShutdown
		}
		if c.tlsGeneration != p.tlsGeneration {
			p.Unlock()
			c.Close()
			return nil, fmt.Errorf("rpc error: TLS configuration changed during connection setup")
		}

		p.pool[addr.String()] = c

		// If there is a connection listener, notify them of the new connection.
		if p.connListener != nil {
			select {
			case p.connListener <- c:
			default:
			}
		}

		p.Unlock()
		return c, nil
	}

	// Otherwise, wait for the lead thread to attempt the connection
	// and use what's in the pool at that point.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.shutdownCh:
		return nil, ErrPoolShutdown
	case <-wait:
	}

	// See if the lead thread was able to get us a connection.
	p.Lock()
	if c := p.pool[addr.String()]; c != nil {
		c.markForUse()
		p.Unlock()
		return c, nil
	}

	p.Unlock()
	return nil, fmt.Errorf("rpc error: lead thread didn't get connection")
}

// getNewConn is used to return a new connection
func (p *ConnPool) getNewConn(region string, addr net.Addr) (*Conn, error) {
	return p.getNewConnContext(context.Background(), region, addr)
}

// getNewConnContext establishes a new pooled session while honoring the
// request context throughout connection setup.
func (p *ConnPool) getNewConnContext(ctx context.Context, region string, addr net.Addr) (*Conn, error) {
	// Capture one configuration for the entire setup. ReloadTLS may replace it
	// concurrently; acquireContext rejects sessions from an older generation.
	p.Lock()
	tlsWrap, tlsGeneration := p.tlsWrap, p.tlsGeneration
	p.Unlock()

	// Try to dial the conn
	dialer := net.Dialer{Timeout: p.dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return nil, err
	}
	// Keep the callback bound to the underlying socket even if conn is later
	// replaced by a TLS wrapper.
	rawConn := conn
	stopCancel := context.AfterFunc(ctx, func() { _ = rawConn.Close() })
	defer stopCancel()

	// Cast to TCPConn
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetNoDelay(true)
	}

	// Check if TLS is enabled
	if tlsWrap != nil {
		// Switch the connection into TLS mode
		if _, err := conn.Write([]byte{byte(RpcTLS)}); err != nil {
			conn.Close()
			return nil, err
		}

		// Wrap the connection in a TLS client
		tlsConn, err := tlsWrap(region, conn)
		if err != nil {
			conn.Close()
			return nil, err
		}
		conn = tlsConn
	}

	// Write the multiplex byte to set the mode
	if _, err := conn.Write([]byte{byte(RpcMultiplexV2)}); err != nil {
		conn.Close()
		return nil, err
	}

	// Create a multiplexed session
	session, err := yamux.Client(conn, p.yamuxCfg)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Wrap the connection
	c := &Conn{
		refCount:      1,
		addr:          addr,
		session:       session,
		clients:       list.New(),
		lastUsed:      atomic.Pointer[time.Time]{},
		pool:          p,
		tlsGeneration: tlsGeneration,
	}

	now := time.Now()
	c.lastUsed.Store(&now)
	return c, nil
}

// clearConn is used to clear any cached connection, potentially in response to
// an error
func (p *ConnPool) clearConn(conn *Conn) {
	// Ensure returned streams are closed
	atomic.StoreInt32(&conn.shouldClose, 1)

	// Clear from the cache
	p.Lock()
	if c, ok := p.pool[conn.addr.String()]; ok && c == conn {
		delete(p.pool, conn.addr.String())
	}
	p.Unlock()

	// Close down immediately if idle
	if refCount := atomic.LoadInt32(&conn.refCount); refCount == 0 {
		conn.Close()
	}
}

// getClient is used to get a usable client for an address
func (p *ConnPool) getRPCClient(region string, addr net.Addr) (*Conn, *StreamClient, error) {
	return p.getRPCClientContext(context.Background(), region, addr)
}

func (p *ConnPool) getRPCClientContext(ctx context.Context, region string, addr net.Addr) (*Conn, *StreamClient, error) {
	retries := 0
START:
	// Try to get a conn first
	conn, err := p.acquireContext(ctx, region, addr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get conn: %w", err)
	}

	// Get a client
	client, err := conn.getRPCClient()
	if err != nil {
		p.clearConn(conn)
		conn.releaseUse()

		// Try to redial, possible that the TCP session closed due to timeout
		if retries == 0 {
			retries++
			goto START
		}
		return nil, nil, fmt.Errorf("failed to start stream: %w", err)
	}
	return conn, client, nil
}

// StreamingRPC is used to make an streaming RPC call.  Callers must
// close the connection when done.
func (p *ConnPool) StreamingRPC(region string, addr net.Addr) (net.Conn, error) {
	conn, err := p.acquire(region, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get conn: %v", err)
	}

	s, err := conn.session.OpenStream()
	if err != nil {
		conn.releaseUse()
		return nil, fmt.Errorf("failed to open a streaming connection: %v", err)
	}

	stream := &incomingStream{Stream: s, parent: conn}
	if _, err := s.Write([]byte{byte(RpcStreaming)}); err != nil {
		stream.Close()
		return nil, err
	}

	return stream, nil
}

// RPC is used to make an RPC call to a remote host
func (p *ConnPool) RPC(region string, addr net.Addr, method string, args interface{}, reply interface{}) error {
	return p.RPCContext(context.Background(), region, addr, method, args, reply)
}

// RPCContext makes an RPC call bounded by ctx. Cancellation closes only the
// request's yamux stream; the shared session remains available to other RPCs.
func (p *ConnPool) RPCContext(ctx context.Context, region string, addr net.Addr, method string, args interface{}, reply interface{}) error {
	// Get a usable client
	conn, sc, err := p.getRPCClientContext(ctx, region, addr)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("rpc error: %w: %w", ErrRPCNotSent, err)
	}
	defer conn.releaseUse()

	// Make the RPC call
	done := make(chan error, 1)
	go func() {
		done <- msgpackrpc.CallWithCodec(sc.codec, method, args, reply)
	}()

	select {
	case err = <-done:
	case <-ctx.Done():
		// yamux Close is a graceful half-close and deliberately leaves reads
		// open until the peer responds. Expire this stream's I/O first so the
		// synchronous codec call returns without resetting the shared session.
		_ = sc.stream.SetDeadline(time.Now())
		sc.Close()
		<-done
		return ctx.Err()
	}
	if err != nil {
		sc.Close()

		// If we read EOF, the session is toast. Clear it and open a
		// new session next time
		// See https://github.com/hashicorp/consul/blob/v1.6.3/agent/pool/pool.go#L471-L477
		if helper.IsErrEOF(err) {
			p.clearConn(conn)
		}

		// If the error is an RPC Coded error
		// return the coded error without wrapping
		if structs.IsErrRPCCoded(err) {
			return err
		}

		// TODO wrap with RPCCoded error instead
		return fmt.Errorf("rpc error: %w", err)
	}

	// Done with the connection
	conn.returnClient(sc)
	return nil
}

// Reap is used to close conns open over maxTime
func (p *ConnPool) reap() {
	for {
		// Sleep for a while
		select {
		case <-p.shutdownCh:
			return
		case <-time.After(time.Second):
		}

		// Reap all old conns
		p.Lock()
		var removed []string
		now := time.Now()
		for host, conn := range p.pool {
			// Skip recently used connections
			if now.Sub(*conn.lastUsed.Load()) < p.maxTime {
				continue
			}

			// Skip connections with active streams
			if atomic.LoadInt32(&conn.refCount) > 0 {
				continue
			}

			// Close the conn
			conn.Close()

			// Remove from pool
			removed = append(removed, host)
		}
		for _, host := range removed {
			delete(p.pool, host)
		}
		p.Unlock()
	}
}
