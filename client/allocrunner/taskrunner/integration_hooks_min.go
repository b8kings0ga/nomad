//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package taskrunner

import (
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
	"github.com/hashicorp/nomad/nomad/structs"
)

func appendIntegrationTokenHooks(hooks []interfaces.TaskHook, _ *TaskRunner, _ *structs.Task, _ log.Logger) []interfaces.TaskHook {
	return hooks
}
func appendIntegrationServiceHooks(hooks []interfaces.TaskHook, _ *TaskRunner, _ *structs.Task, _ *structs.Allocation, _ string, _ log.Logger) []interfaces.TaskHook {
	return hooks
}
