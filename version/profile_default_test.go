// Copyright IBM Corp. 2015, 2025
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package version

import "testing"

func TestDefaultBuildHasNoProfile(t *testing.T) {
	if got := GetVersion().BuildProfile; got != "" {
		t.Fatalf("unexpected default build profile %q", got)
	}
}
