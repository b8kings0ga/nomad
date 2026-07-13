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
	if len(config.Consuls) != 1 || !reflect.DeepEqual(config.Consuls[0], structconfig.DefaultConsulConfig()) {
		return fmt.Errorf("nomad_min: unsupported feature consul configuration")
	}
	if len(config.Vaults) != 1 || !reflect.DeepEqual(config.Vaults[0], structconfig.DefaultVaultConfig()) {
		return fmt.Errorf("nomad_min: unsupported feature vault configuration")
	}

	// Nomad's upstream defaults opt into Consul discovery and registration.
	// The minimal profile changes those implicit defaults before any clients or
	// background loops are constructed.
	consul := config.Consuls[0]
	consul.AutoAdvertise = pointer.Of(false)
	consul.ServerAutoJoin = pointer.Of(false)
	consul.ClientAutoJoin = pointer.Of(false)
	config.Vaults[0].Enabled = pointer.Of(false)
	return nil
}
