// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import (
	"fmt"
	"net/rpc"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/nomad"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/testutil"
)

type heartbeatCountingRPC struct {
	server    *nomad.Server
	calls     atomic.Int32
	attempts  atomic.Int32
	failFirst bool
}

func (p *heartbeatCountingRPC) RPC(method string, args, reply any) error {
	if method == "Node.UpdateStatus" && p.attempts.Add(1) == 1 && p.failFirst {
		return fmt.Errorf("temporary connection failure")
	}
	err := p.server.RPC(method, args, reply)
	if err == nil && method == "Node.UpdateStatus" {
		reply.(*structs.NodeUpdateResponse).HeartbeatTTL = time.Hour
		p.calls.Add(1)
	}
	return err
}

func TestClient_FailedHeartbeatWakesOnRPCSuccess(t *testing.T) {
	s, _, cleanup := testServer(t, nil)
	defer cleanup()
	testutil.WaitForLeader(t, s.RPC)
	proxy := &heartbeatCountingRPC{server: s, failFirst: true}
	c, cleanupClient := TestClient(t, func(cfg *config.Config) {
		cfg.DevMode = true
		cfg.RPCHandler = proxy
	})
	defer cleanupClient()
	testutil.WaitForResult(func() (bool, error) {
		return proxy.attempts.Load() > 0, nil
	}, func(err error) { t.Fatal(err) })
	// The normal development-mode retry delay is one second. A successful RPC
	// must still wake a failed heartbeat before that delay expires.
	deadline := time.Now().Add(500 * time.Millisecond)
	for proxy.calls.Load() == 0 && time.Now().Before(deadline) {
		c.fireRpcRetryWatcher()
		time.Sleep(5 * time.Millisecond)
	}
	if proxy.calls.Load() == 0 {
		t.Fatal("failed heartbeat did not wake on RPC recovery")
	}
}

func TestClient_HealthyHeartbeatIgnoresRPCSuccess(t *testing.T) {
	s, _, cleanup := testServer(t, nil)
	defer cleanup()
	testutil.WaitForLeader(t, s.RPC)
	proxy := &heartbeatCountingRPC{server: s}
	c, cleanupClient := TestClient(t, func(cfg *config.Config) {
		cfg.DevMode = true
		cfg.RPCHandler = proxy
	})
	defer cleanupClient()
	testutil.WaitForResult(func() (bool, error) {
		c.heartbeatLock.Lock()
		defer c.heartbeatLock.Unlock()
		return c.haveHeartbeated, nil
	}, func(err error) { t.Fatal(err) })
	before := proxy.calls.Load()
	for i := 0; i < 20; i++ {
		c.fireRpcRetryWatcher()
		time.Sleep(5 * time.Millisecond)
	}
	if after := proxy.calls.Load(); after != before {
		t.Fatalf("healthy heartbeat woke on RPC success: before=%d after=%d", before, after)
	}
}

func TestClient_HeartbeatNoLeaderBackoff(t *testing.T) {
	c := &Client{
		config: config.DefaultConfig(), haveHeartbeated: true, heartbeatTTL: time.Second,
		heartbeatStop: &heartbeatStop{lastOk: time.Now().Add(-2 * time.Second), lock: &sync.RWMutex{}},
	}
	for _, err := range []error{
		structs.ErrNoLeader,
		rpc.ServerError(structs.ErrNoLeader.Error()),
		fmt.Errorf("failed to update status: %w", rpc.ServerError(structs.ErrNoLeader.Error())),
		fmt.Errorf("failed to update status: %v", structs.ErrNoLeader),
	} {
		if got := c.getHeartbeatRetryIntv(err); got < registerRetryIntv || got > 2*registerRetryIntv {
			t.Errorf("no-leader error %q bypassed registration backoff: %s", err, got)
		}
	}
}
