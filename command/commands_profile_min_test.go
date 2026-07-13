// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package command

import (
	"testing"

	"github.com/hashicorp/cli"
)

func TestMinimalCommandFilter(t *testing.T) {
	commands := map[string]cli.CommandFactory{
		"job":                    nil,
		"acl auth-method create": nil,
		"acl binding-rule list":  nil,
		"license":                nil,
		"login":                  nil,
		"namespace apply":        nil,
		"node pool list":         nil,
		"plugin status":          nil,
		"setup consul":           nil,
		"volume claim list":      nil,
		"volume snapshot list":   nil,
	}
	filterCommandsForBuild(commands)
	if _, ok := commands["job"]; !ok {
		t.Fatal("core job command was removed")
	}
	for name := range commands {
		if name != "job" && name != "volume claim list" {
			t.Fatalf("unsupported command %q was retained", name)
		}
	}
}
