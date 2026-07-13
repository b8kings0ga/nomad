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
		"job":             nil,
		"license":         nil,
		"namespace apply": nil,
		"node pool list":  nil,
		"setup consul":    nil,
	}
	filterCommandsForBuild(commands)
	if _, ok := commands["job"]; !ok {
		t.Fatal("core job command was removed")
	}
	for name := range commands {
		if name != "job" {
			t.Fatalf("unsupported command %q was retained", name)
		}
	}
}
