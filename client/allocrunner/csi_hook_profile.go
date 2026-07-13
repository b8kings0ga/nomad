// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package allocrunner

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
	"github.com/hashicorp/nomad/nomad/structs"
)

func appendCSIAllocHook(hooks []interfaces.RunnerHook, alloc *structs.Allocation, logger hclog.Logger, ar *allocRunner) []interfaces.RunnerHook {
	return append(hooks, newCSIHook(alloc, logger, ar.csiManager, ar.rpcClient, ar, ar.hookResources, ar.clientConfig.Node.SecretID))
}
