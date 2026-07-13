//go:build linux && nomad_min

// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package allocrunner

import (
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/plugins/drivers"
)

func (c *cniNetworkConfigurator) setupTransparentProxyArgs(
	_ *structs.Allocation,
	_ *drivers.NetworkIsolationSpec,
	_ *portMappings,
) (*transparentProxyArgs, error) {
	return nil, nil
}
