//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package taskrunner

import (
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
	"github.com/hashicorp/nomad/nomad/structs"
)

func appendSecretsHook(hooks []interfaces.TaskHook, _ *TaskRunner, _ *structs.Task) []interfaces.TaskHook {
	return hooks
}
