// Copyright IBM Corp. 2015, 2025
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package agent

func validateBuildProfileConfig(*Config) error { return nil }
