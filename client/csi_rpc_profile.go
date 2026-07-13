// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package client

import "net/rpc"

func setupCSIClientEndpoint(server *rpc.Server, c *Client) {
	_ = server.Register(&CSI{c})
}
