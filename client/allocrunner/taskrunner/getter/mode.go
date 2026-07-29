// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package getter

import (
	"io/fs"

	getterlib "github.com/hashicorp/go-getter"
)

type artifactMode = getterlib.ClientMode

const (
	artifactModeAny  = getterlib.ClientModeAny
	artifactModeFile = getterlib.ClientModeFile
	artifactModeDir  = getterlib.ClientModeDir
	umask            = fs.ModeSetuid | fs.ModeSetgid
)
