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

func registerExternalAuthEndpoints(s *HTTPServer) {
	s.mux.HandleFunc("/v1/acl/auth-methods", s.wrap(s.ACLAuthMethodListRequest))
	s.mux.HandleFunc("/v1/acl/auth-method", s.wrap(s.ACLAuthMethodRequest))
	s.mux.HandleFunc("/v1/acl/auth-method/", s.wrap(s.ACLAuthMethodSpecificRequest))
	s.mux.HandleFunc("/v1/acl/binding-rules", s.wrap(s.ACLBindingRuleListRequest))
	s.mux.HandleFunc("/v1/acl/binding-rule", s.wrap(s.ACLBindingRuleRequest))
	s.mux.HandleFunc("/v1/acl/binding-rule/", s.wrap(s.ACLBindingRuleSpecificRequest))
	s.mux.HandleFunc("/v1/acl/oidc/auth-url", s.wrap(s.ACLOIDCAuthURLRequest))
	s.mux.HandleFunc("/v1/acl/oidc/complete-auth", s.wrap(s.ACLOIDCCompleteAuthRequest))
	s.mux.HandleFunc("/v1/acl/login", s.wrap(s.ACLLoginRequest))
}
