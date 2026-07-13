//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package consul

import (
	"fmt"

	"github.com/hashicorp/nomad/client/serviceregistration"
	"github.com/hashicorp/nomad/nomad/structs"
)

const (
	ServiceTagHTTP = "http"
	ServiceTagRPC  = "rpc"
	ServiceTagSerf = "serf"
)

type CatalogAPI interface{}
type ConfigAPI interface{}
type ConfigAPIFunc func(string) ConfigAPI
type ACLsAPI interface{}
type ConnectProxies struct{}
type ServiceClientWrapper struct{}

func NewServiceClientWrapper() *ServiceClientWrapper                    { return &ServiceClientWrapper{} }
func (s *ServiceClientWrapper) Add(string, serviceregistration.Handler) {}
func (s *ServiceClientWrapper) Get(string) serviceregistration.Handler  { return s }
func (s *ServiceClientWrapper) RegisterWorkload(*serviceregistration.WorkloadServices) error {
	return fmt.Errorf("nomad_min: unsupported feature consul service registration")
}
func (s *ServiceClientWrapper) RemoveWorkload(*serviceregistration.WorkloadServices) {}
func (s *ServiceClientWrapper) UpdateWorkload(*serviceregistration.WorkloadServices, *serviceregistration.WorkloadServices) error {
	return fmt.Errorf("nomad_min: unsupported feature consul service registration")
}
func (s *ServiceClientWrapper) AllocRegistrations(string) (*serviceregistration.AllocRegistration, error) {
	return nil, nil
}
func (s *ServiceClientWrapper) UpdateTTL(string, string, string, string) error {
	return fmt.Errorf("nomad_min: unsupported feature consul service registration")
}
func (*ServiceClientWrapper) RegisterAgent(string, []*structs.Service) error {
	return fmt.Errorf("nomad_min: unsupported feature consul agent registration")
}
func (*ServiceClientWrapper) Shutdown() error { return nil }

func MakeCheckID(serviceID string, check *structs.ServiceCheck) string {
	return fmt.Sprintf("_nomad-check-%s", check.Hash(serviceID))
}
