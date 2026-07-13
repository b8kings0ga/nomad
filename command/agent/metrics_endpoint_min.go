// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package agent

// The metrics API and all exporters are absent from minimal builds.
func registerMetricsEndpoint(*HTTPServer) {}
