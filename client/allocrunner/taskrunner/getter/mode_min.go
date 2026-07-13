// Copyright IBM Corp. 2015, 2025
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package getter

type artifactMode string

const (
	artifactModeAny  artifactMode = "any"
	artifactModeFile artifactMode = "file"
	artifactModeDir  artifactMode = "dir"
)
