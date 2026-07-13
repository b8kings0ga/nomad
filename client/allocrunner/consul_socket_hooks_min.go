//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package allocrunner

import (
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
	clientconfig "github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/nomad/structs"
)

func appendConsulSocketHooks(hooks []interfaces.RunnerHook, _ *allocRunner, _ *clientconfig.Config, _ log.Logger, _ *structs.Allocation) []interfaces.RunnerHook {
	return hooks
}
