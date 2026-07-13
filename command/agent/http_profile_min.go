// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build nomad_min

package agent

func registerNodePoolEndpoints(*HTTPServer)   {}
func registerCSIEndpoints(*HTTPServer)        {}
func registerEnterpriseEndpoints(*HTTPServer) {}
