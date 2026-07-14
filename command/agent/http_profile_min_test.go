// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMinimalHTTPDoesNotRegisterEventStream(t *testing.T) {
	server := &HTTPServer{mux: http.NewServeMux()}
	registerEventEndpoint(server)

	_, pattern := server.mux.Handler(httptest.NewRequest(http.MethodGet, "/v1/event/stream", nil))
	if pattern != "" {
		t.Fatalf("event stream route registered as %q", pattern)
	}
}
