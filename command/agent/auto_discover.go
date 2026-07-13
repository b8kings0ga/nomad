// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package agent

import discover "github.com/hashicorp/go-discover"

func newAutoDiscover() AutoDiscoverInterface {
	return autoDiscover{goDiscover: &discover.Discover{}, netAddrs: &netAddrs{}}
}
