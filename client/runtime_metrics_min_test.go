//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import (
	"testing"
	"time"
)

func TestMinimalRuntimeStatsInterval(t *testing.T) {
	if got := runtimeStatsInterval(time.Second); got != 10*time.Second {
		t.Fatalf("short interval was not clamped: %v", got)
	}
	if got := runtimeStatsInterval(30 * time.Second); got != 30*time.Second {
		t.Fatalf("long interval was changed: %v", got)
	}
}
