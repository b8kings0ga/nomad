// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package agent

// Minimal builds never make update-check or security-bulletin network calls.
func (c *Command) startUpdateCheck(*Config) {}
