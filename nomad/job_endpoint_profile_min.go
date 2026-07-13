// Copyright IBM Corp. 2015, 2025
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package nomad

import (
	"fmt"
	"strings"

	"github.com/hashicorp/nomad/nomad/structs"
)

type profileJobHook struct{}

func (profileJobHook) Name() string { return "build-profile" }

func (profileJobHook) Mutate(job *structs.Job) (*structs.Job, []error, error) {
	if job.Namespace != "" && job.Namespace != structs.DefaultNamespace {
		return nil, nil, unsupportedJobFeature("namespace")
	}
	if job.NodePool != "" && job.NodePool != structs.NodePoolDefault {
		return nil, nil, unsupportedJobFeature("node pool")
	}

	for _, group := range job.TaskGroups {
		if group.Consul != nil {
			return nil, nil, unsupportedJobFeature("consul")
		}
		if err := validateMinimalServices(group.Services); err != nil {
			return nil, nil, err
		}
		for _, volume := range group.Volumes {
			if volume != nil && volume.Type == structs.VolumeTypeCSI {
				return nil, nil, unsupportedJobFeature("csi")
			}
		}
		for _, task := range group.Tasks {
			if task.Vault != nil {
				return nil, nil, unsupportedJobFeature("vault")
			}
			if task.Consul != nil || strings.Contains(string(task.Kind), "connect") {
				return nil, nil, unsupportedJobFeature("consul")
			}
			if task.CSIPluginConfig != nil {
				return nil, nil, unsupportedJobFeature("csi")
			}
			if task.Driver != "docker" && task.Driver != "raw_exec" && task.Driver != "exec2" {
				return nil, nil, unsupportedJobFeature("driver " + task.Driver)
			}
			if err := validateMinimalServices(task.Services); err != nil {
				return nil, nil, err
			}
		}
	}

	return job, nil, nil
}

func validateMinimalServices(services []*structs.Service) error {
	for _, service := range services {
		if service == nil {
			continue
		}
		if service.Provider != structs.ServiceProviderNomad {
			return unsupportedJobFeature("consul service provider")
		}
		if service.Connect != nil {
			return unsupportedJobFeature("consul connect")
		}
	}
	return nil
}

func unsupportedJobFeature(name string) error {
	return fmt.Errorf("nomad_min: unsupported feature %s", name)
}
