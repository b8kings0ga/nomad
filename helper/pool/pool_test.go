// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package pool

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/rpc"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/ci"
	"github.com/hashicorp/nomad/helper/testlog"
	"github.com/hashicorp/yamux"
	"github.com/shoenig/test/must"
)

func newTestPool(t *testing.T) *ConnPool {
	l := testlog.HCLogger(t)
	p := NewPool(l, 1*time.Minute, 10, nil, yamux.DefaultConfig(), 10*time.Second)
	return p
}

func Test_NewPool(t *testing.T) {

	// Generate a custom yamux configuration, so we can ensure this gets stored
	// as expected.
	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.AcceptBacklog = math.MaxInt

	testPool := NewPool(hclog.NewNullLogger(), 10*time.Second, 10, nil, yamuxConfig, 15*time.Second)
	must.NotNil(t, testPool)
	must.NotNil(t, testPool.yamuxCfg)
	must.Eq(t, yamuxConfig.AcceptBacklog, testPool.yamuxCfg.AcceptBacklog)
	must.Eq(t, 15*time.Second, testPool.dialTimeout)
}

func Test_NewPool_DefaultDialTimeout(t *testing.T) {
	// A zero (or negative) dial timeout should fall back to the default
	// rather than dialing with no timeout at all.
	testPool := NewPool(hclog.NewNullLogger(), 10*time.Second, 10, nil, yamux.DefaultConfig(), 0)
	must.NotNil(t, testPool)
	must.Eq(t, defaultDialTimeout, testPool.dialTimeout)
}

func TestConnPool_ConnListener(t *testing.T) {
	ports := ci.PortAllocator.Grab(1)
	addrStr := fmt.Sprintf("127.0.0.1:%d", ports[0])
	addr, err := net.ResolveTCPAddr("tcp", addrStr)
	must.NoError(t, err)

	exitCh := make(chan struct{})
	defer close(exitCh)
	go func() {
		ln, listenErr := net.Listen("tcp", addrStr)
		must.NoError(t, listenErr)
		defer func() { _ = ln.Close() }()
		conn, _ := ln.Accept()
		defer func() { _ = conn.Close() }()
		<-exitCh
	}()

	time.Sleep(100 * time.Millisecond)

	// Create a test pool
	pool := newTestPool(t)

	// Setup a listener
	c := make(chan *Conn, 1)
	pool.SetConnListener(c)

	// Make an RPC
	_, err = pool.acquire("test", addr)
	must.NoError(t, err)

	// Assert we get a connection.
	select {
	case <-c:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout")
	}

	// Test that the channel is closed when the pool shuts down.
	err = pool.Shutdown()
	must.NoError(t, err)

	_, ok := <-c
	must.False(t, ok)
}

func TestRPCConnectionFailureMarksRequestNotSent(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr()
	listener.Close()
	p := newTestPool(t)
	defer p.Shutdown()
	err = p.RPC("global", addr, "ServiceRegistration.Upsert", nil, nil)
	if !errors.Is(err, ErrRPCNotSent) {
		t.Fatalf("missing unsent proof: %v", err)
	}
}

func TestConnPoolShutdownRejectsConnections(t *testing.T) {
	for _, inFlight := range []bool{false, true} {
		t.Run(fmt.Sprintf("in_flight=%t", inFlight), func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			must.NoError(t, err)
			t.Cleanup(func() { listener.Close() })
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				io.Copy(io.Discard, conn)
			}()

			p := newTestPool(t)
			t.Cleanup(func() { p.Shutdown() })
			started := make(chan struct{})
			release := make(chan struct{})
			var releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(release) }) }
			defer unblock()
			p.tlsWrap = func(_ string, conn net.Conn) (net.Conn, error) {
				close(started)
				<-release
				return conn, nil
			}
			done := make(chan error, 1)
			acquire := func() {
				conn, err := p.acquire("global", listener.Addr())
				if conn != nil {
					conn.releaseUse()
				}
				done <- err
			}
			if inFlight {
				go acquire()
				select {
				case <-started:
				case <-time.After(2 * time.Second):
					t.Fatal("connection setup did not start")
				}
			}
			must.NoError(t, p.Shutdown())
			unblock()
			if !inFlight {
				go acquire()
			}
			select {
			case err := <-done:
				must.Error(t, err)
			case <-time.After(2 * time.Second):
				t.Fatal("connection setup did not stop after shutdown")
			}
			p.Lock()
			remaining := len(p.pool)
			p.Unlock()
			must.Eq(t, 0, remaining)
		})
	}
}

func TestStreamingRPCReleasesConnection(t *testing.T) {
	for _, closed := range []bool{false, true} {
		t.Run(fmt.Sprintf("closed_session=%t", closed), func(t *testing.T) {
			local, remote := net.Pipe()
			client, err := yamux.Client(local, nil)
			must.NoError(t, err)
			server, err := yamux.Server(remote, nil)
			must.NoError(t, err)
			t.Cleanup(func() { client.Close(); server.Close() })
			p := newTestPool(t)
			t.Cleanup(func() { p.Shutdown() })
			addr := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
			conn := &Conn{addr: addr, session: client, pool: p, clients: list.New()}
			now := time.Now()
			conn.lastUsed.Store(&now)
			p.pool[addr.String()] = conn
			if closed {
				client.Close()
			}
			stream, err := p.StreamingRPC("global", addr)
			if closed {
				must.Error(t, err)
				must.Eq(t, int32(0), atomic.LoadInt32(&conn.refCount))
				return
			}
			must.NoError(t, err)
			peer, err := server.Accept()
			must.NoError(t, err)
			defer peer.Close()
			var mode [1]byte
			_, err = io.ReadFull(peer, mode[:])
			must.NoError(t, err)
			must.Eq(t, byte(RpcStreaming), mode[0])
			must.Eq(t, int32(1), atomic.LoadInt32(&conn.refCount))
			var wg sync.WaitGroup
			for range 8 {
				wg.Add(1)
				go func() { defer wg.Done(); stream.Close() }()
			}
			wg.Wait()
			must.Eq(t, int32(0), atomic.LoadInt32(&conn.refCount))
			must.False(t, client.IsClosed())
			next, err := p.StreamingRPC("global", addr)
			must.NoError(t, err)
			must.NoError(t, next.Close())
			must.Eq(t, int32(0), atomic.LoadInt32(&conn.refCount))
			p.clearConn(conn)
			must.True(t, client.IsClosed())
		})
	}
}

type poolRPCService struct {
	hangStarted chan struct{}
	hangRelease chan struct{}
	active      atomic.Int32
}

func (s *poolRPCService) Hang(_ struct{}, _ *struct{}) error {
	s.active.Add(1)
	defer s.active.Add(-1)
	select {
	case s.hangStarted <- struct{}{}:
	default:
	}
	<-s.hangRelease
	return nil
}

func (s *poolRPCService) Ping(_ struct{}, reply *string) error {
	*reply = "pong"
	return nil
}

func startPoolRPCServer(t *testing.T, service *poolRPCService) (net.Addr, *atomic.Int32) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	must.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	rpcServer := rpc.NewServer()
	must.NoError(t, rpcServer.RegisterName("Test", service))
	accepted := new(atomic.Int32)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			accepted.Add(1)
			go func() {
				var mode [1]byte
				if _, err := io.ReadFull(conn, mode[:]); err != nil || RPCType(mode[0]) != RpcMultiplexV2 {
					_ = conn.Close()
					return
				}
				session, err := yamux.Server(conn, yamux.DefaultConfig())
				if err != nil {
					_ = conn.Close()
					return
				}
				defer session.Close()
				for {
					stream, err := session.Accept()
					if err != nil {
						return
					}
					go func() {
						var rpcMode [1]byte
						if _, err := io.ReadFull(stream, rpcMode[:]); err != nil || RPCType(rpcMode[0]) != RpcNomad {
							_ = stream.Close()
							return
						}
						rpcServer.ServeCodec(NewServerCodec(stream))
					}()
				}
			}()
		}
	}()

	return listener.Addr(), accepted
}

func TestConnPool_RPCContextDeadlinePreservesSharedSession(t *testing.T) {
	service := &poolRPCService{
		hangStarted: make(chan struct{}, 1),
		hangRelease: make(chan struct{}),
	}
	addr, accepted := startPoolRPCServer(t, service)
	p := newTestPool(t)
	t.Cleanup(func() { must.NoError(t, p.Shutdown()) })

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := p.RPCContext(ctx, "global", addr, "Test.Hang", struct{}{}, &struct{}{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("deadline took too long: %v", elapsed)
	}
	select {
	case <-service.hangStarted:
	default:
		t.Fatal("hanging RPC did not reach server")
	}

	var reply string
	if err := p.RPCContext(context.Background(), "global", addr, "Test.Ping", struct{}{}, &reply); err != nil {
		t.Fatalf("RPC on shared session failed after cancellation: %v", err)
	}
	must.Eq(t, "pong", reply)
	must.Eq(t, int32(1), accepted.Load(), must.Sprintf("cancellation replaced the shared TCP session"))

	close(service.hangRelease)
	deadline := time.Now().Add(2 * time.Second)
	for service.active.Load() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	must.Eq(t, int32(0), service.active.Load())
}
