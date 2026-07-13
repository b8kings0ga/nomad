// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package client

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/dynamicplugins"
)

func dynamicPluginDispensers(hclog.Logger) map[string]dynamicplugins.PluginDispenser {
	return nil
}
