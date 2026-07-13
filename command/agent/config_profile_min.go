// Copyright IBM Corp. 2015, 2025
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package agent

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/nomad/helper/pointer"
	structconfig "github.com/hashicorp/nomad/nomad/structs/config"
)

func validateBuildProfileConfig(config *Config) error {
	if len(config.Consuls) != 1 || config.Consuls[0] == nil {
		return fmt.Errorf("nomad_min: unsupported feature consul configuration")
	}
	consul := config.Consuls[0]
	wantConsul := structconfig.DefaultConsulConfig()
	// The upstream implicit defaults are true, while Mimir writes the same
	// settings explicitly as false. Treat both forms as the minimal baseline;
	// all other Consul connection and identity configuration remains rejected.
	consulCopy := consul.Copy()
	consulCopy.AutoAdvertise = pointer.Of(false)
	consulCopy.ServerAutoJoin = pointer.Of(false)
	consulCopy.ClientAutoJoin = pointer.Of(false)
	wantConsul.AutoAdvertise = pointer.Of(false)
	wantConsul.ServerAutoJoin = pointer.Of(false)
	wantConsul.ClientAutoJoin = pointer.Of(false)
	if !reflect.DeepEqual(consulCopy, wantConsul) {
		return fmt.Errorf("nomad_min: unsupported feature consul configuration")
	}

	if len(config.Vaults) != 1 || config.Vaults[0] == nil {
		return fmt.Errorf("nomad_min: unsupported feature vault configuration")
	}
	vaultCopy := config.Vaults[0].Copy()
	vaultCopy.Enabled = pointer.Of(false)
	wantVault := structconfig.DefaultVaultConfig()
	wantVault.Enabled = pointer.Of(false)
	if !reflect.DeepEqual(vaultCopy, wantVault) {
		return fmt.Errorf("nomad_min: unsupported feature vault configuration")
	}

	// Nomad's upstream defaults opt into Consul discovery and registration.
	// The minimal profile changes those implicit defaults before any clients or
	// background loops are constructed.
	consul.AutoAdvertise = pointer.Of(false)
	consul.ServerAutoJoin = pointer.Of(false)
	consul.ClientAutoJoin = pointer.Of(false)
	config.Vaults[0].Enabled = pointer.Of(false)
	return nil
}
