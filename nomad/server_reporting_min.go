// Copyright IBM Corp. 2015, 2025
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package nomad

// buildReportingManager deliberately has no runtime implementation in the
// minimal profile. The field remains so enterprise extensions keep a stable
// Server layout in full builds.
type buildReportingManager struct{}
