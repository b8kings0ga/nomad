//go:build !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"errors"

	ctconfig "github.com/hashicorp/consul-template/config"
)

// ToConsulTemplate converts a client WaitConfig instance to a consul-template WaitConfig.
func (wc *WaitConfig) ToConsulTemplate() (*ctconfig.WaitConfig, error) {
	if wc.IsEmpty() {
		return nil, errors.New("wait config is empty")
	}
	if err := wc.Validate(); err != nil {
		return nil, err
	}
	enabled := wc.Min == nil || *wc.Min != 0 || wc.Max == nil || *wc.Max != 0
	result := &ctconfig.WaitConfig{Enabled: new(enabled), Min: wc.Min, Max: wc.Max}
	return result, nil
}

// ToConsulTemplate converts a client RetryConfig instance to a consul-template RetryConfig.
func (rc *RetryConfig) ToConsulTemplate() (*ctconfig.RetryConfig, error) {
	if err := rc.Validate(); err != nil {
		return nil, err
	}
	result := &ctconfig.RetryConfig{Enabled: new(true), Attempts: rc.Attempts, Backoff: rc.Backoff}
	if rc.MaxBackoff != nil {
		result.MaxBackoff = rc.MaxBackoff
	}
	return result, nil
}
