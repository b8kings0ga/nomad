// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package taskrunner

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
)

func appendCSIPluginHook(hooks []interfaces.TaskHook, _ *TaskRunner, _ hclog.Logger) []interfaces.TaskHook {
	return hooks
}
