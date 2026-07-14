// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package main

import (
	"strings"
	"testing"

	"github.com/hashicorp/cli"
)

func TestMinimalVersionCommand(t *testing.T) {
	ui := cli.NewMockUi()
	if code := runMinimal([]string{"version"}, ui); code != 0 {
		t.Fatalf("version exited with %d: %s", code, ui.ErrorWriter.String())
	}
	if output := ui.OutputWriter.String(); !strings.Contains(output, "BuildProfile nomad_min") {
		t.Fatalf("unexpected version output: %q", output)
	}
}

func TestMinimalRejectsOperatorCommands(t *testing.T) {
	ui := cli.NewMockUi()
	if code := runMinimal([]string{"job", "status"}, ui); code == 0 {
		t.Fatal("unsupported operator command succeeded")
	}
	if output := ui.ErrorWriter.String(); !strings.Contains(output, "nomad_min: unsupported command job") {
		t.Fatalf("unexpected error output: %q", output)
	}
}

func TestMinimalRequiresCommand(t *testing.T) {
	ui := cli.NewMockUi()
	if code := runMinimal(nil, ui); code == 0 {
		t.Fatal("empty command succeeded")
	}
	if output := ui.ErrorWriter.String(); !strings.Contains(output, "nomad_min: expected agent or version") {
		t.Fatalf("unexpected error output: %q", output)
	}
}
