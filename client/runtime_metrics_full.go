//go:build !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import "time"

func runtimeStatsInterval(configured time.Duration) time.Duration { return configured }

func (c *Client) collectRuntimeStats(publish bool) {
	if err := c.hostStatsCollector.Collect(); err != nil {
		c.logger.Warn("error fetching host resource usage stats", "error", err)
	} else if publish {
		c.emitHostStats()
	}
	c.emitClientMetrics()
}
