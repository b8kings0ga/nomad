// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDriverCapabilitiesAreInstanceScoped(t *testing.T) {
	first := &Driver{config: &DriverConfig{DisableLogCollection: true}}
	second := &Driver{config: &DriverConfig{DisableLogCollection: false}}
	a, err := first.Capabilities()
	require.NoError(t, err)
	b, err := second.Capabilities()
	require.NoError(t, err)
	require.True(t, a.DisableLogCollection)
	require.False(t, b.DisableLogCollection)
	require.NotSame(t, a, b)
}
