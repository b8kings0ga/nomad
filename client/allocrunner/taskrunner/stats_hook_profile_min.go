// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package taskrunner

import (
	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
)

func appendStatsHook(hooks []interfaces.TaskHook, _ *TaskRunner, _ hclog.Logger) []interfaces.TaskHook {
	return hooks
}
