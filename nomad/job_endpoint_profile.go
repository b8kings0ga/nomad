// Copyright IBM Corp. 2015, 2025
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package nomad

import "github.com/hashicorp/nomad/nomad/structs"

type profileJobHook struct{}

func (profileJobHook) Name() string { return "build-profile" }

func (profileJobHook) Mutate(job *structs.Job) (*structs.Job, []error, error) {
	return job, nil, nil
}
