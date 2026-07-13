// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package allocrunner

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
	"github.com/hashicorp/nomad/nomad/structs"
)

func appendCSIAllocHook(hooks []interfaces.RunnerHook, _ *structs.Allocation, _ hclog.Logger, _ *allocRunner) []interfaces.RunnerHook {
	return hooks
}
