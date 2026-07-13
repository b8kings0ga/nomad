// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package agent

func registerNodePoolEndpoints(s *HTTPServer) {
	s.mux.HandleFunc("/v1/node/pools", s.wrap(s.NodePoolsRequest))
	s.mux.HandleFunc("/v1/node/pool/", s.wrap(s.NodePoolSpecificRequest))
}

func registerCSIEndpoints(s *HTTPServer) {
	s.mux.HandleFunc("/v1/volumes", s.wrap(s.CSIVolumesRequest))
	s.mux.HandleFunc("/v1/volumes/external", s.wrap(s.CSIExternalVolumesRequest))
	s.mux.HandleFunc("/v1/volumes/snapshot", s.wrap(s.CSISnapshotsRequest))
	s.mux.HandleFunc("/v1/volume/csi/", s.wrap(s.CSIVolumeSpecificRequest))
	s.mux.HandleFunc("/v1/plugins", s.wrap(s.CSIPluginsRequest))
}

func registerEnterpriseEndpoints(s *HTTPServer) {
	s.mux.HandleFunc("/v1/operator/license", s.wrap(s.LicenseRequest))
	s.mux.HandleFunc("/v1/operator/utilization", s.wrap(s.OperatorUtilizationRequest))
	s.mux.HandleFunc("/v1/namespaces", s.wrap(s.NamespacesRequest))
	s.mux.HandleFunc("/v1/namespace", s.wrap(s.NamespaceCreateRequest))
	s.mux.HandleFunc("/v1/namespace/", s.wrap(s.NamespaceSpecificRequest))
}
