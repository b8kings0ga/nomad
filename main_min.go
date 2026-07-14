// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package main

import (
	"fmt"
	"os"

	// The agent invokes these modes as subprocesses of its own executable.
	// Their init functions inspect os.Args and exit before the minimal command
	// dispatcher runs.
	_ "github.com/hashicorp/nomad/client/allocrunner/taskrunner/getter"
	_ "github.com/hashicorp/nomad/client/allocrunner/taskrunner/template/renderer"
	_ "github.com/hashicorp/nomad/client/logmon"
	_ "github.com/hashicorp/nomad/drivers/docker/docklog"
	_ "github.com/hashicorp/nomad/drivers/shared/executor"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/nomad/command/agent"
	"github.com/hashicorp/nomad/version"
)

func main() {
	os.Exit(Run(os.Args[1:]))
}

// Run dispatches the runtime commands retained by the minimal agent package.
func Run(args []string) int {
	return runMinimal(args, &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	})
}

func runMinimal(args []string, ui cli.Ui) int {
	if len(args) == 0 {
		ui.Error("nomad_min: expected agent or version")
		return 1
	}

	switch args[0] {
	case "agent":
		return (&agent.Command{
			Version:    version.GetVersion(),
			Ui:         ui,
			ShutdownCh: make(chan struct{}),
		}).Run(args[1:])
	case "version", "-version", "--version":
		ui.Output(version.GetVersion().FullVersionNumber(true))
		return 0
	default:
		ui.Error(fmt.Sprintf("nomad_min: unsupported command %s", args[0]))
		return 1
	}
}
