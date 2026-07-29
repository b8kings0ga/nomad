// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !nomad_min

package nomad

import "github.com/hashicorp/nomad/nomad/reporting"

type buildReportingManager = *reporting.Manager
