// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package nomad

func applyBuildProfileDefaults(config *Config) {
	// Mimir clusters are deliberately small. A single scheduler worker avoids
	// one long-lived worker and its queues per host CPU while retaining the
	// operator override for larger installations.
	config.NumSchedulers = 1

	// The minimal agent has no event-stream API. Do not allocate the state
	// event broker or its retention buffer.
	config.EnableEventBroker = false
	config.EventBufferSize = 0
}
