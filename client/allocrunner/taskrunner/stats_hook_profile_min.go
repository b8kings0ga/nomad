// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package taskrunner

import (
	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
)

func appendStatsHook(hooks []interfaces.TaskHook, runner *TaskRunner, logger hclog.Logger) []interfaces.TaskHook {
	// Keep the minimal profile opt-in and limited to mimc. Reuse the existing
	// collection interval, lifecycle and publication policy rather than adding
	// a separate sampler or enabling every driver's statistics.
	if runner.task.Driver != "mimc" || !runner.clientConfig.PublishAllocationMetrics {
		return hooks
	}
	return append(hooks, newStatsHook(runner, runner.clientConfig.StatsCollectionInterval, true, logger))
}
