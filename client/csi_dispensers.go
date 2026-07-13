// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package client

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/dynamicplugins"
	"github.com/hashicorp/nomad/plugins/csi"
)

func dynamicPluginDispensers(logger hclog.Logger) map[string]dynamicplugins.PluginDispenser {
	return map[string]dynamicplugins.PluginDispenser{
		dynamicplugins.PluginTypeCSIController: func(info *dynamicplugins.PluginInfo) (interface{}, error) {
			return csi.NewClient(info.ConnectionInfo.SocketPath, logger.Named("csi_client").With("plugin.name", info.Name, "plugin.type", "controller")), nil
		},
		dynamicplugins.PluginTypeCSINode: func(info *dynamicplugins.PluginInfo) (interface{}, error) {
			return csi.NewClient(info.ConnectionInfo.SocketPath, logger.Named("csi_client").With("plugin.name", info.Name, "plugin.type", "client")), nil
		},
	}
}
