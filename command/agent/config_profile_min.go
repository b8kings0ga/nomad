// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package agent

import (
	"fmt"
	"reflect"

	structconfig "github.com/hashicorp/nomad/nomad/structs/config"
)

func applyBuildProfileDefaults(config *Config) {
	config.Server.EnableEventBroker = new(false)
	config.Server.EventBufferSize = new(0)
}

func validateBuildProfileConfig(config *Config) error {
	if config.Server.EnableEventBroker != nil && *config.Server.EnableEventBroker {
		return fmt.Errorf("nomad_min: unsupported feature event stream")
	}
	if config.Server.EventBufferSize != nil && *config.Server.EventBufferSize != 0 {
		return fmt.Errorf("nomad_min: unsupported feature event stream buffer")
	}

	if len(config.Consuls) != 1 || config.Consuls[0] == nil {
		return fmt.Errorf("nomad_min: unsupported feature consul configuration")
	}
	consul := config.Consuls[0]
	wantConsul := structconfig.DefaultConsulConfig()
	// The upstream implicit defaults are true, while Mimir writes the same
	// settings explicitly as false. Treat both forms as the minimal baseline;
	// all other Consul connection and identity configuration remains rejected.
	consulCopy := consul.Copy()
	consulCopy.AutoAdvertise = new(false)
	consulCopy.ServerAutoJoin = new(false)
	consulCopy.ClientAutoJoin = new(false)
	wantConsul.AutoAdvertise = new(false)
	wantConsul.ServerAutoJoin = new(false)
	wantConsul.ClientAutoJoin = new(false)
	if !reflect.DeepEqual(consulCopy, wantConsul) {
		return fmt.Errorf("nomad_min: unsupported feature consul configuration")
	}

	if len(config.Vaults) != 1 || config.Vaults[0] == nil {
		return fmt.Errorf("nomad_min: unsupported feature vault configuration")
	}
	vaultCopy := config.Vaults[0].Copy()
	vaultCopy.Enabled = new(false)
	wantVault := structconfig.DefaultVaultConfig()
	wantVault.Enabled = new(false)
	if !reflect.DeepEqual(vaultCopy, wantVault) {
		return fmt.Errorf("nomad_min: unsupported feature vault configuration")
	}

	// Nomad's upstream defaults opt into Consul discovery and registration.
	// The minimal profile changes those implicit defaults before any clients or
	// background loops are constructed.
	consul.AutoAdvertise = new(false)
	consul.ServerAutoJoin = new(false)
	consul.ClientAutoJoin = new(false)
	config.Vaults[0].Enabled = new(false)
	config.Telemetry.DisableAllocationHookMetrics = new(true)
	return nil
}
