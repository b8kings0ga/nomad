//go:build !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package taskrunner

import (
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/allocrunner/interfaces"
	"github.com/hashicorp/nomad/nomad/structs"
)

func appendIntegrationTokenHooks(hooks []interfaces.TaskHook, tr *TaskRunner, task *structs.Task, logger log.Logger) []interfaces.TaskHook {
	hooks = append(hooks, newConsulHook(logger, tr))
	if task.Vault != nil && tr.vaultClientFunc != nil {
		hooks = append(hooks, newVaultHook(&vaultHookConfig{
			vaultBlock: task.Vault, vaultConfigsFunc: tr.clientConfig.GetVaultConfigs,
			clientFunc: tr.vaultClientFunc, events: tr, lifecycle: tr, updater: tr,
			logger: logger, alloc: tr.Alloc(), task: tr.Task(), widmgr: tr.widmgr,
		}))
	}
	return hooks
}

func appendIntegrationServiceHooks(hooks []interfaces.TaskHook, tr *TaskRunner, task *structs.Task, alloc *structs.Allocation, namespace string, logger log.Logger) []interfaces.TaskHook {
	if task.UsesConnect() {
		tg := tr.Alloc().Job.LookupTaskGroup(tr.Alloc().TaskGroup)
		cfg := tr.clientConfig.GetConsulConfigs(tr.logger)[task.GetConsulClusterName(tg)]
		if cfg != nil && cfg.Token != "" {
			hooks = append(hooks, newSIDSHook(sidsHookConfig{alloc: tr.Alloc(), task: tr.Task(), lifecycle: tr, logger: logger, allocHookResources: tr.allocHookResources}))
		}
		if task.UsesConnectSidecar() {
			hooks = append(hooks,
				newEnvoyVersionHook(newEnvoyVersionHookConfig(alloc, tr.consulProxiesClientFunc, logger)),
				newEnvoyBootstrapHook(newEnvoyBootstrapHookConfig(alloc, cfg, namespace, tr.consulServiceClient, tr.clientConfig.Node, logger)))
		} else if task.Kind.IsConnectNative() {
			hooks = append(hooks, newConnectNativeHook(newConnectNativeHookConfig(alloc, cfg, logger)))
		}
	}
	return append(hooks, newScriptCheckHook(scriptCheckHookConfig{
		alloc: tr.Alloc(), task: tr.Task(), consul: tr.consulServiceClient,
		logger: logger, arHookResources: tr.allocHookResources,
	}))
}
