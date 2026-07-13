//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package vaultclient

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/nomad/structs/config"
)

type VaultClientFunc func(string) (VaultClient, error)

type JWTLoginRequest struct{ JWT, Role, Namespace string }

type VaultClient interface {
	Start()
	Stop()
	DeriveTokenWithJWT(context.Context, JWTLoginRequest) (string, bool, int, error)
	RenewToken(string, int) (<-chan error, error)
	StopRenewToken(string) error
}

type vaultClient struct{}

func NewVaultClient(*config.VaultConfig, hclog.Logger) (*vaultClient, error) {
	return &vaultClient{}, nil
}
func (*vaultClient) Start() {}
func (*vaultClient) Stop()  {}
func (*vaultClient) DeriveTokenWithJWT(context.Context, JWTLoginRequest) (string, bool, int, error) {
	return "", false, 0, fmt.Errorf("nomad_min: unsupported feature vault")
}
func (*vaultClient) RenewToken(string, int) (<-chan error, error) {
	return nil, fmt.Errorf("nomad_min: unsupported feature vault")
}
func (*vaultClient) StopRenewToken(string) error { return nil }
