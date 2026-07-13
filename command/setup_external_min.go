//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

type SetupConsulCommand struct{ Meta }

func (*SetupConsulCommand) Help() string     { return "Consul setup is unavailable in nomad_min" }
func (*SetupConsulCommand) Synopsis() string { return "Unavailable in nomad_min" }
func (c *SetupConsulCommand) Run([]string) int {
	c.Ui.Error("nomad_min: unsupported feature consul")
	return 1
}

type SetupVaultCommand struct{ Meta }

func (*SetupVaultCommand) Help() string     { return "Vault setup is unavailable in nomad_min" }
func (*SetupVaultCommand) Synopsis() string { return "Unavailable in nomad_min" }
func (c *SetupVaultCommand) Run([]string) int {
	c.Ui.Error("nomad_min: unsupported feature vault")
	return 1
}
