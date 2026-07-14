// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package nomad

func registerEventStreamingEndpoint(server *Server) {
	// Event is a streaming-only endpoint so we don't want to register it as a
	// normal RPC.
	eventEndpoint := NewEventEndpoint(server)
	eventEndpoint.register()
}
