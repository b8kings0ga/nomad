//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import "time"

const minimalStatsInterval = 10 * time.Second

func runtimeStatsInterval(configured time.Duration) time.Duration {
	if configured < minimalStatsInterval {
		return minimalStatsInterval
	}
	return configured
}

func (c *Client) collectRuntimeStats(bool) {
	if err := c.hostStatsCollector.Collect(); err != nil {
		c.logger.Warn("error fetching host resource usage stats", "error", err)
	}
}
