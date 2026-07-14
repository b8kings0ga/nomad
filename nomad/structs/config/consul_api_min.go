//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"time"

	"github.com/hashicorp/nomad/nomad/structs"
)

func DefaultConsulConfig() *ConsulConfig {
	return &ConsulConfig{
		Name: "default", ServerServiceName: "nomad", ServerHTTPCheckName: "Nomad Server HTTP Check",
		ServerSerfCheckName: "Nomad Server Serf Check", ServerRPCCheckName: "Nomad Server RPC Check",
		ClientServiceName: "nomad-client", ClientHTTPCheckName: "Nomad Client HTTP Check",
		AutoAdvertise: new(false), ChecksUseAdvertise: new(false),
		ServerAutoJoin: new(false), ClientAutoJoin: new(false), Timeout: 5 * time.Second,
		ServiceIdentityAuthMethod: structs.ConsulWorkloadsDefaultAuthMethodName,
		TaskIdentityAuthMethod:    structs.ConsulWorkloadsDefaultAuthMethodName,
		Addr:                      "127.0.0.1:8500", EnableSSL: new(false), VerifySSL: new(true),
	}
}
