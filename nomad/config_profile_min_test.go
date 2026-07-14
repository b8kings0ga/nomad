// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package nomad

import "testing"

func TestMinimalBuildProfileDefaults(t *testing.T) {
	config := DefaultConfig()
	if config.NumSchedulers != 1 {
		t.Fatalf("expected one scheduler worker, got %d", config.NumSchedulers)
	}
	if config.EnableEventBroker {
		t.Fatal("event broker remains enabled")
	}
	if config.EventBufferSize != 0 {
		t.Fatalf("event buffer remains allocated: %d", config.EventBufferSize)
	}
}
