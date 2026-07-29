// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package getter

import (
	"context"

	getterlib "github.com/hashicorp/go-getter"
)

func (p *parameters) client(ctx context.Context) *getterlib.Client {
	httpGetter := &getterlib.HttpGetter{
		Netrc: true, Header: p.Headers, XTerraformGetDisabled: true,
		DoNotCheckHeadFirst: true, ReadTimeout: p.HTTPReadTimeout, MaxBytes: p.HTTPMaxBytes,
	}
	decompressors := getterlib.LimitedDecompressors(p.DecompressionLimitFileCount, p.DecompressionLimitSize)
	getters := map[string]getterlib.Getter{
		"git":  &getterlib.GitGetter{Timeout: p.GitTimeout},
		"hg":   &getterlib.HgGetter{Timeout: p.HgTimeout},
		"gcs":  &getterlib.GCSGetter{Timeout: p.GCSTimeout},
		"s3":   &getterlib.S3Getter{Timeout: p.S3Timeout},
		"http": httpGetter, "https": httpGetter,
	}
	return &getterlib.Client{
		Ctx: ctx, Src: p.Source, Dst: p.Destination, Mode: getterMode(p.Mode), Insecure: p.Insecure,
		Umask: umask, DisableSymlinks: true,
		Decompressors: decompressors, Getters: getters,
	}
}

func getterMode(mode artifactMode) getterlib.ClientMode {
	switch mode {
	case artifactModeFile:
		return getterlib.ClientModeFile
	case artifactModeDir:
		return getterlib.ClientModeDir
	default:
		return getterlib.ClientModeAny
	}
}
