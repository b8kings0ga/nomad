//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package structs

import (
	"sync"

	"github.com/hashicorp/nomad/client/pluginmanager/csimanager"
	"github.com/hashicorp/nomad/helper"
	"github.com/hashicorp/nomad/nomad/structs"
)

type AllocHookResources struct {
	csiMounts      map[string]*csimanager.MountInfo
	consulTokens   map[string]map[string]*minimalACLToken
	consulCheckIDs [][]string
	networkStatus  *structs.AllocNetworkStatus
	mu             sync.RWMutex
}

type minimalACLToken struct {
	SecretID string
}

func NewAllocHookResources() *AllocHookResources {
	return &AllocHookResources{
		csiMounts:      map[string]*csimanager.MountInfo{},
		consulTokens:   map[string]map[string]*minimalACLToken{},
		consulCheckIDs: [][]string{},
	}
}

func (a *AllocHookResources) GetConsulTokens() map[string]map[string]*minimalACLToken {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.consulTokens
}

func (a *AllocHookResources) GetCSIMounts() map[string]*csimanager.MountInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return helper.DeepCopyMap(a.csiMounts)
}

func (a *AllocHookResources) SetCSIMounts(m map[string]*csimanager.MountInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.csiMounts = m
}

func (a *AllocHookResources) GetConsulCheckIDs() [][]string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([][]string, len(a.consulCheckIDs))
	for i, ids := range a.consulCheckIDs {
		result[i] = append([]string(nil), ids...)
	}
	return result
}

func (a *AllocHookResources) SetConsulCheckIDs(ids [][]string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.consulCheckIDs = ids
}

func (a *AllocHookResources) GetAllocNetworkStatus() *structs.AllocNetworkStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.networkStatus.Copy()
}

func (a *AllocHookResources) SetAllocNetworkStatus(status *structs.AllocNetworkStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.networkStatus = status
}
