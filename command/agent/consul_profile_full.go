//go:build !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package agent

import (
	consulapi "github.com/hashicorp/consul/api"
	clientconsul "github.com/hashicorp/nomad/client/consul"
	"github.com/hashicorp/nomad/command/agent/consul"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/nomad/structs/config"
)

func (a *Agent) setupConsulsProfile(cfgs []*config.ConsulConfig) error {
	isClient := a.config.Client != nil && a.config.Client.Enabled
	a.consulServices = consul.NewServiceClientWrapper()
	proxies := map[string]*consul.ConnectProxies{}
	entries := map[string]consul.ConfigAPI{}
	for _, cfg := range cfgs {
		cluster := cfg.Name
		apiCfg, err := cfg.ApiConfig()
		if err != nil {
			return err
		}
		client, err := consulapi.NewClient(apiCfg)
		if err != nil {
			return err
		}
		entries[cluster] = client.ConfigEntries()
		if cluster == structs.ConsulDefaultCluster {
			a.consulACLs, a.consulCatalog = client.ACL(), client.Catalog()
		}
		agentClient := client.Agent()
		ns := consul.NewNamespacesClient(client.Namespaces(), agentClient)
		a.consulServices.AddClient(cluster, consul.NewServiceClient(agentClient, ns, a.logger, isClient))
		proxies[cluster] = consul.NewConnectProxiesClient(agentClient)
	}
	a.consulProxiesFunc = func(cluster string) clientconsul.SupportedProxiesAPI { return proxies[cluster] }
	a.consulConfigEntriesFunc = func(cluster string) consul.ConfigAPI { return entries[cluster] }
	a.consulServices.Run()
	return nil
}
