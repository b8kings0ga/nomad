// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package command

import (
	"strings"

	"github.com/hashicorp/cli"
)

var minimalUnsupportedCommandPrefixes = []string{
	"license",
	"namespace",
	"node pool",
	"operator utilization",
	"quota",
	"recommendation",
	"sentinel",
	"setup consul",
	"setup vault",
}

func filterCommandsForBuild(commands map[string]cli.CommandFactory) {
	for name := range commands {
		for _, prefix := range minimalUnsupportedCommandPrefixes {
			if name == prefix || strings.HasPrefix(name, prefix+" ") {
				delete(commands, name)
				break
			}
		}
	}
}
