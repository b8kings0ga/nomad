// Copyright IBM Corp. 2015, 2025
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package agent

import "testing"

func TestMinimalConfigDisablesImplicitIntegrations(t *testing.T) {
	config := DefaultConfig()
	if err := validateBuildProfileConfig(config); err != nil {
		t.Fatal(err)
	}
	consul := config.Consuls[0]
	if *consul.AutoAdvertise || *consul.ServerAutoJoin || *consul.ClientAutoJoin {
		t.Fatal("implicit consul integration remains enabled")
	}
	if config.Vaults[0].Enabled == nil || *config.Vaults[0].Enabled {
		t.Fatal("implicit vault integration remains enabled")
	}
}

func TestMinimalConfigRejectsIntegrationConfig(t *testing.T) {
	config := DefaultConfig()
	config.Consuls[0].Addr = "127.0.0.1:8500"
	if err := validateBuildProfileConfig(config); err == nil {
		t.Fatal("expected consul configuration rejection")
	}

	config = DefaultConfig()
	config.Vaults[0].Role = "nomad"
	if err := validateBuildProfileConfig(config); err == nil {
		t.Fatal("expected vault configuration rejection")
	}
}
