// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package nomad

import "net/rpc"

func registerCSIRPCEndpoints(server *rpc.Server, s *Server, ctx *RPCContext) {
	_ = server.Register(NewClientCSIEndpoint(s, ctx))
	_ = server.Register(NewCSIVolumeEndpoint(s, ctx))
	_ = server.Register(NewCSIPluginEndpoint(s, ctx))
}
