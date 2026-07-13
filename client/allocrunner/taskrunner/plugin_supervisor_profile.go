// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package taskrunner

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
)

func appendCSIPluginHook(hooks []interfaces.TaskHook, tr *TaskRunner, logger hclog.Logger) []interfaces.TaskHook {
	if tr.Task().CSIPluginConfig == nil {
		return hooks
	}
	return append(hooks, newCSIPluginSupervisorHook(&csiPluginSupervisorHookConfig{
		clientStateDirPath: tr.clientConfig.StateDir,
		events:             tr,
		runner:             tr,
		lifecycle:          tr,
		capabilities:       tr.driverCapabilities,
		logger:             logger,
	}))
}
