//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package consul

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/nomad/structs/config"
)

type SupportedProxiesAPI interface {
	Proxies() (map[string][]string, error)
}
type SupportedProxiesAPIFunc func(string) SupportedProxiesAPI
type Client interface{}
type ConsulClientFunc func(*config.ConsulConfig, hclog.Logger) (Client, error)
type NodeGetter interface{}

func NewConsulClientFactory(NodeGetter) ConsulClientFunc {
	return func(*config.ConsulConfig, hclog.Logger) (Client, error) {
		return nil, fmt.Errorf("nomad_min: unsupported feature consul")
	}
}
