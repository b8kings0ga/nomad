// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package nomad

import "github.com/hashicorp/nomad/nomad/structs"

func (s *Server) replySetIndex(table string, reply *structs.QueryMeta) error {
	index, err := s.fsm.State().Index(table)
	if err != nil {
		return err
	}
	reply.Index = index
	s.setQueryMeta(reply)
	return nil
}
