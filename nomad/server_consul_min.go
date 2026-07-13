//go:build nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package nomad

import "fmt"

func (s *Server) setupBootstrapHandler() error {
	return fmt.Errorf("nomad_min: unsupported feature consul server discovery")
}

func (s *Server) setupConsulSyncer() error {
	conf := s.config.GetDefaultConsul()
	if conf.ServerAutoJoin != nil && *conf.ServerAutoJoin {
		return s.setupBootstrapHandler()
	}
	return nil
}
