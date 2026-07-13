//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import "fmt"

func (*Client) consulDiscoveryImpl() error {
	return fmt.Errorf("nomad_min: unsupported feature consul server discovery")
}
