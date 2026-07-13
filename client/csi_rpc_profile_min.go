// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package client

import "net/rpc"

func setupCSIClientEndpoint(*rpc.Server, *Client) {}
