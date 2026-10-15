// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package agent

import "net/http"

// These paths overlap /v1/node/. Explicitly reject removed APIs so they do not
// fall through to node-ID parsing and turn an unsupported API into a 500.
func registerUnavailableNodePoolEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/v1/node/pools", http.NotFound)
	mux.HandleFunc("/v1/node/pool/", http.NotFound)
}
