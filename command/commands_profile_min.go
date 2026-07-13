// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package command

import (
	"strings"

	"github.com/hashicorp/cli"
)

var minimalUnsupportedCommandPrefixes = []string{
	"acl auth-method",
	"acl binding-rule",
	"license",
	"login",
	"namespace",
	"node pool",
	"operator utilization",
	"plugin",
	"quota",
	"recommendation",
	"sentinel",
	"setup consul",
	"setup vault",
	"volume create",
	"volume delete",
	"volume deregister",
	"volume detach",
	"volume init",
	"volume register",
	"volume snapshot",
	"volume status",
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
