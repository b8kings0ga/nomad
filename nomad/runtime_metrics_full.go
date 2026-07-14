//go:build !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package nomad

import (
	"time"

	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

func (s *Server) startRuntimeMetrics(evalBroker *EvalBroker) {
	go evalBroker.EmitStats(time.Second, s.shutdownCh)
	go s.planQueue.EmitStats(time.Second, s.shutdownCh)
	go s.planner.badNodeTracker.EmitStats(time.Second, s.shutdownCh)
	go s.blockedEvals.EmitStats(time.Second, s.shutdownCh)
	go s.heartbeatStats()
	go s.EmitRaftStats(10*time.Second, s.shutdownCh)
}

func (s *Server) startRaftStoreMetrics(store *raftboltdb.BoltStore) {
	go store.RunMetrics(s.shutdownCtx, 0)
}
