//go:build !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package allocrunner

import (
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
	clientconfig "github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/nomad/structs"
)

func appendConsulSocketHooks(hooks []interfaces.RunnerHook, ar *allocRunner, config *clientconfig.Config, logger log.Logger, alloc *structs.Allocation) []interfaces.RunnerHook {
	return append(hooks,
		newConsulGRPCSocketHook(logger, alloc, ar.allocDir, config.GetConsulConfigs(ar.logger), config.Node.Attributes),
		newConsulHTTPSocketHook(logger, alloc, ar.allocDir, config.GetConsulConfigs(ar.logger)))
}
