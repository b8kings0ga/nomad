// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package agent

import (
	"strings"
	"testing"
)

func TestMinimalTelemetryRejectsExporters(t *testing.T) {
	config := DefaultConfig()
	config.Telemetry.PrometheusMetrics = true
	_, err := new(Command).setupTelemetry(config)
	if err == nil || !strings.Contains(err.Error(), "nomad_min: unsupported feature metrics exporter") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMinimalTelemetryUsesBlackhole(t *testing.T) {
	config := DefaultConfig()
	if _, err := new(Command).setupTelemetry(config); err != nil {
		t.Fatal(err)
	}
}

func TestMinimalTelemetryAllowsLocalAllocationStats(t *testing.T) {
	config := DefaultConfig()
	config.Telemetry.PublishAllocationMetrics = true
	if _, err := new(Command).setupTelemetry(config); err != nil {
		t.Fatal(err)
	}
	config.Telemetry.PublishNodeMetrics = true
	if _, err := new(Command).setupTelemetry(config); err == nil {
		t.Fatal("node metrics must remain disabled")
	}
}
