// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package nomad

import (
	"strings"
	"testing"

	"github.com/hashicorp/nomad/nomad/structs"
)

func TestProfileJobHookMinimal(t *testing.T) {
	tests := map[string]*structs.Job{
		"namespace":       {Namespace: "other"},
		"node pool":       {NodePool: "other"},
		"vault":           {TaskGroups: []*structs.TaskGroup{{Tasks: []*structs.Task{{Driver: "docker", Vault: &structs.Vault{}}}}}},
		"consul provider": {TaskGroups: []*structs.TaskGroup{{Services: []*structs.Service{{Provider: structs.ServiceProviderConsul}}}}},
		"csi":             {TaskGroups: []*structs.TaskGroup{{Volumes: map[string]*structs.VolumeRequest{"data": {Type: structs.VolumeTypeCSI}}}}},
		"driver":          {TaskGroups: []*structs.TaskGroup{{Tasks: []*structs.Task{{Driver: "java"}}}}},
	}

	for name, job := range tests {
		t.Run(name, func(t *testing.T) {
			_, _, err := (profileJobHook{}).Mutate(job)
			if err == nil || !strings.HasPrefix(err.Error(), "nomad_min: unsupported feature ") {
				t.Fatalf("expected nomad_min unsupported error, got %v", err)
			}
		})
	}

	valid := &structs.Job{
		Namespace: structs.DefaultNamespace,
		NodePool:  structs.NodePoolDefault,
		TaskGroups: []*structs.TaskGroup{{
			Services: []*structs.Service{{Provider: structs.ServiceProviderNomad}},
			Tasks:    []*structs.Task{{Driver: "docker"}, {Driver: "raw_exec"}, {Driver: "mimc"}},
		}},
	}
	if _, _, err := (profileJobHook{}).Mutate(valid); err != nil {
		t.Fatalf("valid minimal job rejected: %v", err)
	}
}
