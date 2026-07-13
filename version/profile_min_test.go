// Copyright IBM Corp. 2015, 2025
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package version

import (
	"strings"
	"testing"
)

func TestMinimalBuildProfile(t *testing.T) {
	v := GetVersion()
	if v.BuildProfile != "nomad_min" {
		t.Fatalf("unexpected build profile %q", v.BuildProfile)
	}
	if !strings.Contains(v.FullVersionNumber(false), "BuildProfile nomad_min") {
		t.Fatalf("version output does not include minimal profile: %q", v.FullVersionNumber(false))
	}
}
