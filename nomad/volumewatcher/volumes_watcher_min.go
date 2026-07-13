// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package volumewatcher

import "github.com/hashicorp/nomad/nomad/state"

type Watcher struct{}

func NewDisabledWatcher() *Watcher { return &Watcher{} }

func (*Watcher) SetEnabled(bool, *state.StateStore, string) {}
