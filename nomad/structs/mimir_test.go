// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package structs

import (
	"testing"
	"time"

	"github.com/shoenig/test/must"
)

func testMimirCapability(cpu float64, ram int64) *MimirNodeCapability {
	return &MimirNodeCapability{BenchmarkSpec: MimirBenchmarkSpecV1, ResultID: "r1", ArtifactSHA256: "sha256:x", HardwareFingerprint: "hw", SampleCount: 5, CPUSingleMCU: cpu, CPUTotalMCU: cpu, Threads: 1, RAMBytes: ram, RAMMMU: 10, DiskBytes: 1 << 40, DiskRandomReadIOPS: 10000, DiskRandomWriteIOPS: 10000, NetworkUploadBytesPerSecond: 1000, NetworkDownloadBytesPerSecond: 1000}
}

func TestMimirPlacementCost_AbsoluteAndExtremeFixtures(t *testing.T) {
	now := time.Unix(1000, 0)
	health := &MimirNodeHealth{PlacementSpec: MimirPlacementSpecV1, SampledAt: now, NetworkRTTMilliseconds: 1}
	req := &MimirWorkloadRequirement{CPUExpectedMCU: .5, MemoryBytes: 256 << 20}
	small := testMimirCapability(1, 512<<20)
	large := testMimirCapability(128, 512<<30)
	smallCost, ok := MimirPlacementCost(small, health, req, MimirResourceUsage{}, now)
	must.True(t, ok)
	largeCost, ok := MimirPlacementCost(large, health, req, MimirResourceUsage{}, now)
	must.True(t, ok)
	must.Less(t, largeCost, smallCost)
	// Evaluating an existing candidate again is independent of adding a faster node.
	again, ok := MimirPlacementCost(small, health, req, MimirResourceUsage{}, now)
	must.True(t, ok)
	must.Eq(t, smallCost, again)
}

func TestMimirPlacementCost_HardFilters(t *testing.T) {
	now := time.Unix(1000, 0)
	cap := testMimirCapability(2, 1<<30)
	_, ok := MimirPlacementCost(cap, &MimirNodeHealth{PlacementSpec: MimirPlacementSpecV1, SampledAt: now.Add(-91 * time.Second)}, &MimirWorkloadRequirement{CPUExpectedMCU: 1}, MimirResourceUsage{}, now)
	must.False(t, ok)
	_, ok = MimirPlacementCost(nil, &MimirNodeHealth{PlacementSpec: MimirPlacementSpecV1, SampledAt: now}, &MimirWorkloadRequirement{CPUExpectedMCU: 1}, MimirResourceUsage{}, now)
	must.False(t, ok)
	_, ok = MimirPlacementCost(cap, &MimirNodeHealth{PlacementSpec: MimirPlacementSpecV1, SampledAt: now}, &MimirWorkloadRequirement{CPUExpectedMCU: 3}, MimirResourceUsage{}, now)
	must.False(t, ok)
	_, ok = MimirPlacementCost(nil, nil, &MimirWorkloadRequirement{BenchmarkWorkload: true}, MimirResourceUsage{}, now)
	must.True(t, ok)
}
