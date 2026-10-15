// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package pool

import (
	"container/list"
	"context"
	"errors"
	"io"
	"net"
	"net/rpc"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/yamux"
)

func TestRPCContextCancelsSaturatedStreamOpen(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		io.Copy(io.Discard, conn) // No yamux ACK: deliberately saturate stream opens.
	}()
	cfg := yamux.DefaultConfig()
	cfg.AcceptBacklog = 1
	p := NewPool(hclog.NewNullLogger(), 0, 1, nil, cfg, time.Second)
	defer p.Shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	err = p.RPCContext(ctx, "global", listener.Addr(), "Test.Ping", struct{}{}, new(string))
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("first RPC: %v", err)
	}
	p.Lock()
	conn := p.pool[listener.Addr().String()]
	p.Unlock()
	if conn == nil {
		t.Fatal("cancellation discarded shared session")
	}
	const callers = 32
	done := make(chan error, callers)
	for range callers {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer cancel()
			done <- p.RPCContext(ctx, "global", listener.Addr(), "Test.Ping", struct{}{}, new(string))
		}()
	}
	deadline := time.NewTimer(500 * time.Millisecond)
	defer deadline.Stop()
	for range callers {
		select {
		case err := <-done:
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("saturated RPC: %v", err)
			}
		case <-deadline.C:
			t.Fatal("canceled RPCs remained blocked opening streams")
		}
	}
	if conn.IsClosed() {
		t.Fatal("canceled stream opens reset shared session")
	}
	if refs := atomic.LoadInt32(&conn.refCount); refs > 1 {
		t.Fatalf("abandoned opens are not bounded to one worker per connection: %d", refs)
	}
	p.Shutdown()
	until := time.Now().Add(time.Second)
	for atomic.LoadInt32(&conn.refCount) != 0 && time.Now().Before(until) {
		time.Sleep(time.Millisecond)
	}
	if refs := atomic.LoadInt32(&conn.refCount); refs != 0 {
		t.Fatalf("shutdown leaked open worker: %d", refs)
	}
}

func TestRPCContextStreamOpenRecoversAfterLateACK(t *testing.T) {
	local, remote := net.Pipe()
	cfg := yamux.DefaultConfig()
	cfg.AcceptBacklog = 1
	client, err := yamux.Client(local, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := yamux.Server(remote, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	// Fill the SYN backlog before allowing the server to accept any streams.
	filler, err := client.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer filler.Close()
	p := newTestPool(t)
	defer p.Shutdown()
	addr := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 4647}
	conn := &Conn{addr: addr, session: client, pool: p, clients: list.New()}
	now := time.Now()
	conn.lastUsed.Store(&now)
	p.pool[addr.String()] = conn
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	err = p.RPCContext(ctx, "global", addr, "Test.Ping", struct{}{}, new(string))
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("saturated RPC: %v", err)
	}
	if client.IsClosed() {
		t.Fatal("canceling an open reset the session")
	}

	rpcServer := rpc.NewServer()
	if err := rpcServer.RegisterName("Test", new(poolRPCService)); err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			stream, err := server.Accept()
			if err != nil {
				return
			}
			go func() {
				defer stream.Close()
				var mode [1]byte
				if _, err := io.ReadFull(stream, mode[:]); err != nil {
					return
				}
				rpcServer.ServeCodec(NewServerCodec(stream))
			}()
		}
	}()
	filler.Close()
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var reply string
	if err := p.RPCContext(ctx, "global", addr, "Test.Ping", struct{}{}, &reply); err != nil {
		t.Fatalf("session did not recover after late ACK: %v", err)
	}
	if reply != "pong" {
		t.Fatalf("unexpected reply: %q", reply)
	}
	if client.IsClosed() {
		t.Fatal("recovery used a reset session")
	}
}

func TestRPCContextConcurrentStreamOpen(t *testing.T) {
	addr, accepted := startPoolRPCServer(t, new(poolRPCService))
	p := newTestPool(t)
	defer p.Shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	const callers = 64
	done := make(chan error, callers)
	for range callers {
		go func() {
			var reply string
			err := p.RPCContext(ctx, "global", addr, "Test.Ping", struct{}{}, &reply)
			if err == nil && reply != "pong" {
				err = errors.New("unexpected Ping response")
			}
			done <- err
		}()
	}
	for range callers {
		if err := <-done; err != nil {
			t.Errorf("concurrent RPC failed: %v", err)
		}
	}
	if connections := accepted.Load(); connections != 1 {
		t.Fatalf("concurrent opens did not share one connection: %d", connections)
	}
}
