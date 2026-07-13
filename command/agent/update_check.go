// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package agent

import (
	"fmt"
	"path/filepath"
	"time"

	checkpoint "github.com/hashicorp/go-checkpoint"
	"github.com/hashicorp/nomad/helper"
)

func (c *Command) startUpdateCheck(config *Config) {
	if config.DisableUpdateCheck == nil || *config.DisableUpdateCheck {
		return
	}

	version := config.Version.Version
	if config.Version.VersionPrerelease != "" {
		version += fmt.Sprintf("-%s", config.Version.VersionPrerelease)
	}
	params := &checkpoint.CheckParams{Product: "nomad", Version: version}
	if !config.DisableAnonymousSignature {
		params.SignatureFile = filepath.Join(config.DataDir, "checkpoint-signature")
	}

	checkpoint.CheckInterval(params, 24*time.Hour, c.checkpointResults)
	go func() {
		time.Sleep(helper.RandomStagger(30 * time.Second))
		c.checkpointResults(checkpoint.Check(params))
	}()
}

func (c *Command) checkpointResults(results *checkpoint.CheckResponse, err error) {
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to check for updates: %v", err))
		return
	}
	if results.Outdated {
		c.Ui.Error(fmt.Sprintf("Newer Nomad version available: %s (currently running: %s)", results.CurrentVersion, c.Version.VersionNumber()))
	}
	for _, alert := range results.Alerts {
		if alert.Level == "info" {
			c.Ui.Info(fmt.Sprintf("Bulletin [%s]: %s (%s)", alert.Level, alert.Message, alert.URL))
		} else {
			c.Ui.Error(fmt.Sprintf("Bulletin [%s]: %s (%s)", alert.Level, alert.Message, alert.URL))
		}
	}
}
