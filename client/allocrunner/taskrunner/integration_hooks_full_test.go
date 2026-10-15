//go:build !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package taskrunner

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/allocdir"
	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/client/vaultclient"
	"github.com/hashicorp/nomad/nomad/structs"
)

func TestIntegrationVaultHookTaskContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	task := &structs.Task{Name: "task", Vault: &structs.Vault{}}
	tr := &TaskRunner{
		killCtx: ctx, task: task, alloc: &structs.Allocation{},
		clientConfig: config.DefaultConfig(), taskDir: &allocdir.TaskDir{},
		vaultClientFunc: func(string) (vaultclient.VaultClient, error) {
			t.Fatal("hook construction must not contact Vault")
			return nil, nil
		},
	}
	hooks := appendIntegrationTokenHooks(nil, tr, task, hclog.NewNullLogger())
	for _, hook := range hooks {
		if vault, ok := hook.(*vaultHook); ok {
			defer vault.taskCancel()
			if vault.taskCtx.Err() != nil {
				t.Fatal("Vault hook context started canceled")
			}
			cancel()
			select {
			case <-vault.taskCtx.Done():
			case <-time.After(time.Second):
				t.Fatal("Vault hook did not inherit task cancellation")
			}
			return
		}
	}
	t.Fatal("Vault hook was not registered")
}
