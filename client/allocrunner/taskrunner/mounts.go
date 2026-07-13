// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package taskrunner

import "github.com/hashicorp/nomad/plugins/drivers"

func ensureMountpointInserted(mounts []*drivers.MountConfig, mount *drivers.MountConfig) []*drivers.MountConfig {
	for _, existing := range mounts {
		if existing.IsEqual(mount) {
			return mounts
		}
	}
	return append(mounts, mount)
}
