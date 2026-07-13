// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package agent

import (
	"fmt"
	"log"
)

type minimalDiscover struct{}

func (minimalDiscover) Addrs(string, *log.Logger) ([]string, error) {
	return nil, fmt.Errorf("nomad_min: unsupported feature cloud retry-join discovery")
}

func (minimalDiscover) Help() string    { return "cloud retry-join discovery is not built" }
func (minimalDiscover) Names() []string { return nil }

func newAutoDiscover() AutoDiscoverInterface {
	return autoDiscover{goDiscover: minimalDiscover{}, netAddrs: &netAddrs{}}
}
