//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package nomad

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/command/agent/consul"
	"github.com/hashicorp/nomad/nomad/structs"
)

type ConsulConfigsAPI interface {
	SetIngressCE(context.Context, string, string, string, string, *structs.ConsulIngressConfigEntry) error
	SetTerminatingCE(context.Context, string, string, string, string, *structs.ConsulTerminatingConfigEntry) error
	Stop()
}

type consulConfigsAPI struct{}

func NewConsulConfigsAPI(consul.ConfigAPIFunc, hclog.Logger) *consulConfigsAPI {
	return &consulConfigsAPI{}
}
func (*consulConfigsAPI) SetIngressCE(context.Context, string, string, string, string, *structs.ConsulIngressConfigEntry) error {
	return fmt.Errorf("nomad_min: unsupported feature consul config entries")
}
func (*consulConfigsAPI) SetTerminatingCE(context.Context, string, string, string, string, *structs.ConsulTerminatingConfigEntry) error {
	return fmt.Errorf("nomad_min: unsupported feature consul config entries")
}
func (*consulConfigsAPI) Stop() {}
