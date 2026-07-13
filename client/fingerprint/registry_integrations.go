// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package fingerprint

var envFingerprinters = map[string]Factory{
	"env_aws":          NewEnvAWSFingerprint,
	"env_gce":          NewEnvGCEFingerprint,
	"env_azure":        NewEnvAzureFingerprint,
	"env_digitalocean": NewEnvDigitalOceanFingerprint,
}

func init() {
	hostFingerprinters["consul"] = NewConsulFingerprint
	hostFingerprinters["vault"] = NewVaultFingerprint
}
