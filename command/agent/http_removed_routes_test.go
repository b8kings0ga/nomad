// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTP_UnavailableNodePoolRoutes(t *testing.T) {
	mux := http.NewServeMux()
	registerUnavailableNodePoolEndpoints(mux)
	mux.HandleFunc("/v1/node/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	for _, path := range []string{"/v1/node/pools", "/v1/node/pool/default", "/v1/node/pool/default/jobs"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s fell through removed API: %d", path, w.Code)
		}
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/node/node-a", nil))
	if w.Code != http.StatusAccepted {
		t.Fatalf("node endpoint was shadowed: %d", w.Code)
	}
}
