//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package nomad

import raftboltdb "github.com/hashicorp/raft-boltdb/v2"

func (*Server) startRuntimeMetrics(*EvalBroker) {}

func (*Server) startRaftStoreMetrics(*raftboltdb.BoltStore) {}
