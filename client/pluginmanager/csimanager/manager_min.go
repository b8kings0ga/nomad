// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package csimanager

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/dynamicplugins"
	"github.com/hashicorp/nomad/client/pluginmanager"
	"github.com/hashicorp/nomad/nomad/structs"
)

type MountInfo struct {
	Source   string
	IsDevice bool
}

func (m *MountInfo) Copy() *MountInfo {
	if m == nil {
		return nil
	}
	result := *m
	return &result
}

type UsageOptions struct {
	ReadOnly       bool
	AttachmentMode structs.VolumeAttachmentMode
	AccessMode     structs.VolumeAccessMode
	MountOptions   *structs.CSIMountOptions
}

func (u *UsageOptions) ToFS() string {
	var result strings.Builder
	if u.ReadOnly {
		result.WriteString("ro-")
	} else {
		result.WriteString("rw-")
	}
	result.WriteString(string(u.AttachmentMode))
	result.WriteByte('-')
	result.WriteString(string(u.AccessMode))
	return result.String()
}

type VolumeManager interface{}

type Manager interface {
	PluginManager() pluginmanager.PluginManager
	WaitForPlugin(context.Context, string, string) error
	ManagerForPlugin(context.Context, string) (VolumeManager, error)
	Shutdown()
}

type UpdateNodeCSIInfoFunc func(string, *structs.CSIInfo)
type TriggerNodeEvent func(*structs.NodeEvent)

type Config struct {
	Logger                hclog.Logger
	DynamicRegistry       dynamicplugins.Registry
	UpdateNodeCSIInfoFunc UpdateNodeCSIInfoFunc
	PluginResyncPeriod    time.Duration
	TriggerNodeEvent      TriggerNodeEvent
}

type disabledManager struct{}

func New(*Config) Manager { return disabledManager{} }

func (disabledManager) PluginManager() pluginmanager.PluginManager { return disabledManager{} }
func (disabledManager) Run()                                       {}
func (disabledManager) Shutdown()                                  {}
func (disabledManager) PluginType() string                         { return "csi-disabled" }

func (disabledManager) WaitForPlugin(context.Context, string, string) error {
	return fmt.Errorf("nomad_min: unsupported feature csi")
}

func (disabledManager) ManagerForPlugin(context.Context, string) (VolumeManager, error) {
	return nil, fmt.Errorf("nomad_min: unsupported feature csi")
}
