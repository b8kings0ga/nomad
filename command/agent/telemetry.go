// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package agent

import (
	metrics "github.com/hashicorp/go-metrics/compat"
	"github.com/hashicorp/go-metrics/compat/circonus"
	"github.com/hashicorp/go-metrics/compat/datadog"
	"github.com/hashicorp/go-metrics/compat/prometheus"
)

func (c *Command) setupTelemetry(config *Config) (*metrics.InmemSink, error) {
	telConfig := config.Telemetry
	if telConfig == nil {
		telConfig = &Telemetry{}
	}
	inm := metrics.NewInmemSink(telConfig.inMemoryCollectionInterval, telConfig.inMemoryRetentionPeriod)
	metrics.DefaultInmemSignal(inm)
	metricsConf := metrics.DefaultConfig("nomad")
	metricsConf.EnableHostname = !telConfig.DisableHostname
	metricsConf.EnableHostnameLabel = !telConfig.DisableHostname
	if telConfig.UseNodeName {
		metricsConf.HostName = config.NodeName
		metricsConf.EnableHostname = true
	}
	allowed, blocked, err := telConfig.PrefixFilters()
	if err != nil {
		return inm, err
	}
	metricsConf.AllowedPrefixes = allowed
	metricsConf.BlockedPrefixes = blocked
	if telConfig.FilterDefault != nil {
		metricsConf.FilterDefault = *telConfig.FilterDefault
	}

	var fanout metrics.FanoutSink
	if telConfig.StatsiteAddr != "" {
		sink, err := metrics.NewStatsiteSink(telConfig.StatsiteAddr)
		if err != nil {
			return inm, err
		}
		fanout = append(fanout, sink)
	}
	if telConfig.StatsdAddr != "" {
		sink, err := metrics.NewStatsdSink(telConfig.StatsdAddr)
		if err != nil {
			return inm, err
		}
		fanout = append(fanout, sink)
	}
	if telConfig.PrometheusMetrics {
		sink, err := prometheus.NewPrometheusSink()
		if err != nil {
			return inm, err
		}
		fanout = append(fanout, sink)
	}
	if telConfig.DataDogAddr != "" {
		sink, err := datadog.NewDogStatsdSink(telConfig.DataDogAddr, config.NodeName)
		if err != nil {
			return inm, err
		}
		sink.SetTags(telConfig.DataDogTags)
		fanout = append(fanout, sink)
	}
	if telConfig.CirconusAPIToken != "" || telConfig.CirconusCheckSubmissionURL != "" {
		cfg := &circonus.Config{}
		cfg.Interval = telConfig.CirconusSubmissionInterval
		cfg.CheckManager.API.TokenKey = telConfig.CirconusAPIToken
		cfg.CheckManager.API.TokenApp = telConfig.CirconusAPIApp
		cfg.CheckManager.API.URL = telConfig.CirconusAPIURL
		cfg.CheckManager.Check.SubmissionURL = telConfig.CirconusCheckSubmissionURL
		cfg.CheckManager.Check.ID = telConfig.CirconusCheckID
		cfg.CheckManager.Check.ForceMetricActivation = telConfig.CirconusCheckForceMetricActivation
		cfg.CheckManager.Check.InstanceID = telConfig.CirconusCheckInstanceID
		cfg.CheckManager.Check.SearchTag = telConfig.CirconusCheckSearchTag
		cfg.CheckManager.Check.Tags = telConfig.CirconusCheckTags
		cfg.CheckManager.Check.DisplayName = telConfig.CirconusCheckDisplayName
		cfg.CheckManager.Broker.ID = telConfig.CirconusBrokerID
		cfg.CheckManager.Broker.SelectTag = telConfig.CirconusBrokerSelectTag
		if cfg.CheckManager.Check.DisplayName == "" {
			cfg.CheckManager.Check.DisplayName = "Nomad"
		}
		if cfg.CheckManager.API.TokenApp == "" {
			cfg.CheckManager.API.TokenApp = "nomad"
		}
		if cfg.CheckManager.Check.SearchTag == "" {
			cfg.CheckManager.Check.SearchTag = "service:nomad"
		}
		sink, err := circonus.NewCirconusSink(cfg)
		if err != nil {
			return inm, err
		}
		sink.Start()
		fanout = append(fanout, sink)
	}
	if len(fanout) > 0 {
		fanout = append(fanout, inm)
		metrics.NewGlobal(metricsConf, fanout)
	} else {
		metricsConf.EnableHostname = false
		metrics.NewGlobal(metricsConf, inm)
	}
	return inm, nil
}
