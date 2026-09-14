// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package agent

import (
	"fmt"
	"time"

	metrics "github.com/hashicorp/go-metrics/compat"
)

func (c *Command) setupTelemetry(config *Config) (*metrics.InmemSink, error) {
	tel := config.Telemetry
	if tel != nil && (tel.StatsiteAddr != "" || tel.StatsdAddr != "" || tel.PrometheusMetrics ||
		tel.DataDogAddr != "" || len(tel.DataDogTags) != 0 || tel.CirconusAPIToken != "" ||
		tel.CirconusCheckSubmissionURL != "" || tel.PublishNodeMetrics ||
		tel.IncludeAllocMetadataInMetrics || len(tel.AllowedMetadataKeysInMetrics) != 0) {
		return nil, fmt.Errorf("nomad_min: unsupported feature metrics exporter")
	}
	// PublishAllocationMetrics permits the opt-in mimc task sampler and local
	// allocation stats API. It does not enable an exporter in the minimal build.
	// Agent and HTTP interfaces retain the concrete InmemSink type for full
	// builds. Keep a tiny detached instance for that API, while all process
	// metrics go to a blackhole sink.
	inm := metrics.NewInmemSink(time.Hour, time.Hour)
	conf := metrics.DefaultConfig("nomad")
	conf.EnableHostname = false
	conf.EnableHostnameLabel = false
	if _, err := metrics.NewGlobal(conf, &metrics.BlackholeSink{}); err != nil {
		return nil, err
	}
	return inm, nil
}
