//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package agent

import (
	clientconsul "github.com/hashicorp/nomad/client/consul"
	"github.com/hashicorp/nomad/command/agent/consul"
	"github.com/hashicorp/nomad/nomad/structs/config"
)

func (a *Agent) setupConsulsProfile([]*config.ConsulConfig) error {
	a.consulServices = consul.NewServiceClientWrapper()
	a.consulProxiesFunc = func(string) clientconsul.SupportedProxiesAPI { return nil }
	a.consulConfigEntriesFunc = func(string) consul.ConfigAPI { return nil }
	return nil
}
