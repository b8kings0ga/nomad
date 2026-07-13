//go:build !nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import vault "github.com/hashicorp/vault/api"

func (c *VaultConfig) ApiConfig() (*vault.Config, error) {
	conf := vault.DefaultConfig()
	tlsConf := &vault.TLSConfig{
		CACert: c.TLSCaFile, CAPath: c.TLSCaPath, ClientCert: c.TLSCertFile,
		ClientKey: c.TLSKeyFile, TLSServerName: c.TLSServerName,
	}
	if c.TLSSkipVerify != nil {
		tlsConf.Insecure = *c.TLSSkipVerify
	}
	if err := conf.ConfigureTLS(tlsConf); err != nil {
		return nil, err
	}
	conf.Address = c.Addr
	return conf, nil
}
