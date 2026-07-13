//go:build !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package taskrunner

import (
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
	"github.com/hashicorp/nomad/nomad/structs"
)

func appendSecretsHook(hooks []interfaces.TaskHook, tr *TaskRunner, task *structs.Task) []interfaces.TaskHook {
	if len(task.Secrets) == 0 {
		return hooks
	}
	return append(hooks, newSecretsHook(&secretsHookConfig{
		logger:         tr.logger,
		lifecycle:      tr,
		events:         tr,
		clientConfig:   tr.clientConfig,
		envBuilder:     tr.envBuilder,
		nomadNamespace: tr.alloc.Job.Namespace,
		jobId:          tr.alloc.Job.ID,
	}, task.Secrets))
}
