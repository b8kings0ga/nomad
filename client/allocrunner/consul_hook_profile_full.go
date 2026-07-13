//go:build !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package allocrunner

import (
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
	clientconfig "github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/client/consul"
)

func appendConsulAllocHook(hooks []interfaces.RunnerHook, ar *allocRunner, config *clientconfig.Config, logger log.Logger) []interfaces.RunnerHook {
	return append(hooks, newConsulHook(consulHookConfig{
		alloc: ar.alloc, allocdir: ar.allocDir, widmgr: ar.widmgr,
		consulConfigs:           ar.clientConfig.GetConsulConfigs(logger),
		consulClientConstructor: consul.NewConsulClientFactory(config),
		hookResources:           ar.hookResources, logger: logger, db: ar.stateDB,
	}))
}
