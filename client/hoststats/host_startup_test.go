// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package hoststats

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/lib/numalib"
)

func TestHostStatsCollectorConcurrentInitialStats(t *testing.T) {
	var collections atomic.Int32
	collector := NewHostStatsCollector(hclog.NewNullLogger(), &numalib.Topology{}, t.TempDir(), func() []*DeviceGroupStats {
		collections.Add(1)
		return nil
	})
	const readers = 32
	ready := make(chan struct{})
	results := make(chan *HostStats, readers)
	var wait sync.WaitGroup
	for range readers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-ready
			results <- collector.Stats()
		}()
	}
	close(ready)
	wait.Wait()
	close(results)
	if got := collections.Load(); got != 1 {
		t.Fatalf("initial collections = %d; want 1", got)
	}
	var first *HostStats
	for stats := range results {
		if stats == nil {
			t.Fatal("initial stats are nil")
		}
		if first == nil {
			first = stats
		}
		if stats != first {
			t.Fatal("concurrent first readers did not share the initial snapshot")
		}
	}
	if got := collector.Stats(); got != first {
		t.Fatal("cached read changed snapshot")
	}
}
