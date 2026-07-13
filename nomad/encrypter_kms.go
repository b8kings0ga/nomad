// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package nomad

import (
	"context"
	"fmt"
	"os"

	kms "github.com/hashicorp/go-kms-wrapping/v2"
	"github.com/hashicorp/go-kms-wrapping/v2/aead"
	"github.com/hashicorp/go-kms-wrapping/wrappers/awskms/v2"
	"github.com/hashicorp/go-kms-wrapping/wrappers/azurekeyvault/v2"
	"github.com/hashicorp/go-kms-wrapping/wrappers/gcpckms/v2"
	"github.com/hashicorp/go-kms-wrapping/wrappers/transit/v2"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/nomad/structs/config"
)

func fallbackVaultConfig(provider *structs.KEKProviderConfig, vaultcfg *config.VaultConfig) {
	setFallback := func(key, cfg, env, fallback string) {
		if provider.Config == nil {
			provider.Config = map[string]string{}
		}
		if _, ok := provider.Config[key]; ok {
			return
		}
		if cfg != "" {
			provider.Config[key] = cfg
		} else if envVal := os.Getenv(env); envVal != "" {
			provider.Config[key] = envVal
		} else {
			provider.Config[key] = fallback
		}
	}

	setFallback("address", vaultcfg.Addr, "VAULT_ADDR", "")
	setFallback("token", vaultcfg.Token, "VAULT_TOKEN", "")
	setFallback("tls_ca_cert", vaultcfg.TLSCaPath, "VAULT_CACERT", "")
	setFallback("tls_client_cert", vaultcfg.TLSCertFile, "VAULT_CLIENT_CERT", "")
	setFallback("tls_client_key", vaultcfg.TLSKeyFile, "VAULT_CLIENT_KEY", "")
	setFallback("tls_server_name", vaultcfg.TLSServerName, "VAULT_TLS_SERVER_NAME", "")
	skipVerify := ""
	if vaultcfg.TLSSkipVerify != nil {
		skipVerify = fmt.Sprintf("%v", *vaultcfg.TLSSkipVerify)
	}
	setFallback("tls_skip_verify", skipVerify, "VAULT_SKIP_VERIFY", "false")
}

func (e *Encrypter) newKMSWrapper(provider *structs.KEKProviderConfig, keyID string, kek []byte) (kms.Wrapper, error) {
	var wrapper kms.Wrapper
	switch provider.Provider {
	case structs.KEKProviderAWSKMS:
		wrapper = awskms.NewWrapper()
	case structs.KEKProviderAzureKeyVault:
		wrapper = azurekeyvault.NewWrapper()
	case structs.KEKProviderGCPCloudKMS:
		wrapper = gcpckms.NewWrapper()
	case structs.KEKProviderVaultTransit:
		wrapper = transit.NewWrapper()
	default:
		return newAEADWrapper(keyID, kek)
	}

	if config, ok := e.providerConfigs[provider.ID()]; ok {
		if _, err := wrapper.SetConfig(context.Background(), kms.WithConfigMap(config.Config)); err != nil {
			return nil, err
		}
	}
	return wrapper, nil
}

func newAEADWrapper(keyID string, kek []byte) (kms.Wrapper, error) {
	wrapper := aead.NewWrapper()
	wrapper.SetConfig(context.Background(),
		aead.WithAeadType(kms.AeadTypeAesGcm),
		aead.WithHashType(kms.HashTypeSha256),
		kms.WithKeyId(keyID),
	)
	if err := wrapper.SetAesGcmKeyBytes(kek); err != nil {
		return nil, err
	}
	return wrapper, nil
}
