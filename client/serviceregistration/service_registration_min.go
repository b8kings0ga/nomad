//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package serviceregistration

import (
	"context"
	"maps"

	"github.com/hashicorp/nomad/nomad/structs"
)

type Handler interface {
	RegisterWorkload(*WorkloadServices) error
	RemoveWorkload(*WorkloadServices)
	UpdateWorkload(*WorkloadServices, *WorkloadServices) error
	AllocRegistrations(string) (*AllocRegistration, error)
	UpdateTTL(string, string, string, string) error
}
type HandlerFunc func(string) Handler
type WorkloadRestarter interface {
	Restart(context.Context, *structs.TaskEvent, bool) error
}

type AllocRegistration struct {
	Tasks map[string]*ServiceRegistrations
}

func (a *AllocRegistration) Copy() *AllocRegistration {
	if a == nil {
		return nil
	}
	c := &AllocRegistration{Tasks: make(map[string]*ServiceRegistrations, len(a.Tasks))}
	for k, v := range a.Tasks {
		c.Tasks[k] = v.copy()
	}
	return c
}
func (a *AllocRegistration) NumServices() int {
	if a == nil {
		return 0
	}
	n := 0
	for _, t := range a.Tasks {
		n += len(t.Services)
	}
	return n
}
func (a *AllocRegistration) NumChecks() int {
	if a == nil {
		return 0
	}
	n := 0
	for _, t := range a.Tasks {
		for _, s := range t.Services {
			n += len(s.Checks)
		}
	}
	return n
}

type ServiceRegistrations struct {
	Services map[string]*ServiceRegistration
}

func (t *ServiceRegistrations) copy() *ServiceRegistrations {
	c := &ServiceRegistrations{Services: make(map[string]*ServiceRegistration, len(t.Services))}
	for k, v := range t.Services {
		c.Services[k] = v.copy()
	}
	return c
}

type minimalService struct{ ID, Service, Status string }
type minimalCheck struct{ CheckID, Name, Status, Output string }
type ServiceRegistration struct {
	ServiceID      string
	CheckIDs       map[string]struct{}
	CheckOnUpdate  map[string]string
	Service        *minimalService
	Checks         []*minimalCheck
	SidecarService *minimalService
	SidecarChecks  []*minimalCheck
}

func (s *ServiceRegistration) copy() *ServiceRegistration {
	return &ServiceRegistration{ServiceID: s.ServiceID, CheckIDs: maps.Clone(s.CheckIDs), CheckOnUpdate: maps.Clone(s.CheckOnUpdate)}
}
