// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package structs

import (
	"errors"
	"math"
	"time"
)

const (
	MimirBenchmarkSpecV1 = "mimir-bench-v1"
	MimirPlacementSpecV1 = "mimir-placement-v1"
	MimirHealthMaxAge    = 90 * time.Second
)

type MimirRegionalNetwork struct {
	Region                 string  `json:"region"`
	AnchorURL              string  `json:"anchor_url,omitempty"`
	Protocol               string  `json:"protocol,omitempty"`
	UploadBytesPerSecond   float64 `json:"upload_mbps"`
	DownloadBytesPerSecond float64 `json:"download_mbps"`
	RTTMilliseconds        float64 `json:"rtt_ms"`
	LossRatio              float64 `json:"packet_loss"`
	JitterMilliseconds     float64 `json:"jitter_ms"`
}

type MimirNodeCapability struct {
	BenchmarkSpec                 string                 `json:"benchmark_spec"`
	ResultID                      string                 `json:"result_id"`
	ArtifactSHA256                string                 `json:"artifact_sha256"`
	HardwareFingerprint           string                 `json:"hardware_fingerprint"`
	MeasuredAt                    time.Time              `json:"measured_at"`
	SampleCount                   int                    `json:"sample_count"`
	CPUSingleMCU                  float64                `json:"cpu_single_mcu"`
	CPUTotalMCU                   float64                `json:"cpu_total_mcu"`
	Threads                       int                    `json:"cpu_threads"`
	RAMBytes                      int64                  `json:"memory_capacity_bytes"`
	RAMMMU                        float64                `json:"memory_bandwidth_mmu"`
	RAMLatencyNS                  float64                `json:"memory_latency_ns"`
	DiskBytes                     int64                  `json:"storage_capacity_bytes"`
	DiskReadBytesPerSecond        float64                `json:"storage_seq_read_bps"`
	DiskWriteBytesPerSecond       float64                `json:"storage_seq_write_bps"`
	DiskRandomReadIOPS            float64                `json:"storage_random_read_iops"`
	DiskRandomWriteIOPS           float64                `json:"storage_random_write_iops"`
	DiskFsyncMicroseconds         float64                `json:"storage_fsync_us"`
	NetworkUploadBytesPerSecond   float64                `json:"network_upload_mbps"`
	NetworkDownloadBytesPerSecond float64                `json:"network_download_mbps"`
	Regions                       []MimirRegionalNetwork `json:"network_regions"`
}

func (c *MimirNodeCapability) Copy() *MimirNodeCapability {
	if c == nil {
		return nil
	}
	n := *c
	n.Regions = append([]MimirRegionalNetwork(nil), c.Regions...)
	return &n
}

func (c *MimirNodeCapability) Validate() error {
	if c == nil || c.BenchmarkSpec != MimirBenchmarkSpecV1 || c.ResultID == "" || c.ArtifactSHA256 == "" || c.HardwareFingerprint == "" {
		return errors.New("invalid mimir capability identity")
	}
	if c.SampleCount != 5 || c.CPUSingleMCU <= 0 || c.CPUTotalMCU <= 0 || c.Threads <= 0 || c.RAMBytes <= 0 || c.RAMMMU <= 0 || c.DiskBytes <= 0 {
		return errors.New("invalid mimir capability measurements")
	}
	return nil
}

type MimirNodeHealth struct {
	PlacementSpec             string    `json:"placement_spec"`
	SampledAt                 time.Time `json:"sampled_at"`
	CPUStealRatio             float64   `json:"cpu_steal_fraction"`
	CPUPSISomeRatio           float64   `json:"cpu_psi_some_fraction"`
	MemoryPSIFullRatio        float64   `json:"memory_psi_full_fraction"`
	IOPSIFullRatio            float64   `json:"io_psi_full_fraction"`
	MemoryAvailableBytes      int64     `json:"memory_available_bytes"`
	IOQueueDepth              float64   `json:"disk_queue_depth"`
	NetworkUtilizationRatio   float64   `json:"network_utilization_fraction"`
	NetworkLossRatio          float64   `json:"packet_loss_fraction"`
	NetworkJitterMilliseconds float64   `json:"jitter_ms"`
	NetworkRTTMilliseconds    float64   `json:"rtt_ms"`
}

func (h *MimirNodeHealth) Copy() *MimirNodeHealth {
	if h == nil {
		return nil
	}
	n := *h
	return &n
}

type MimirWorkloadRequirement struct {
	CPUExpectedMCU                float64 `json:"cpu_expected_mcu"`
	CPUSingleMinimumMCU           float64 `json:"cpu_single_min_mcu"`
	MemoryBytes                   int64   `json:"memory_bytes"`
	MemoryBandwidthMMU            float64 `json:"memory_bandwidth_mmu"`
	DiskBytes                     int64   `json:"storage_bytes"`
	DiskReadBytesPerSecond        float64 `json:"storage_read_bps"`
	DiskWriteBytesPerSecond       float64 `json:"storage_write_bps"`
	DiskReadIOPS                  float64 `json:"storage_read_iops"`
	DiskWriteIOPS                 float64 `json:"storage_write_iops"`
	DiskFsyncMaxMicroseconds      float64 `json:"storage_fsync_max_us"`
	NetworkUploadBytesPerSecond   float64 `json:"network_upload_mbps"`
	NetworkDownloadBytesPerSecond float64 `json:"network_download_mbps"`
	Region                        string  `json:"egress_region"`
	Protocol                      string  `json:"protocol"`
	RTTTargetMilliseconds         float64 `json:"latency_target_ms"`
	BenchmarkWorkload             bool    `json:"benchmark_workload"`
	ControlPlaneWorkload          bool    `json:"control_plane_workload"`
}

type MimirHealthFactors struct{ CPU, RAM, Disk, Network float64 }

func (h *MimirNodeHealth) Factors(now time.Time) (MimirHealthFactors, bool) {
	if h == nil || h.PlacementSpec != MimirPlacementSpecV1 || h.SampledAt.IsZero() || now.Sub(h.SampledAt) > MimirHealthMaxAge || h.SampledAt.Sub(now) > 5*time.Second {
		return MimirHealthFactors{}, false
	}
	clamp := func(v float64) float64 { return math.Max(.25, math.Min(1, v)) }
	jitter := 0.0
	if h.NetworkRTTMilliseconds > 0 {
		jitter = .5 * h.NetworkJitterMilliseconds / h.NetworkRTTMilliseconds
	}
	return MimirHealthFactors{
		CPU: clamp(1 - h.CPUStealRatio - h.CPUPSISomeRatio), RAM: clamp(1 - h.MemoryPSIFullRatio), Disk: clamp(1 - h.IOPSIFullRatio),
		Network: clamp(1 - h.NetworkLossRatio - math.Max(0, h.NetworkUtilizationRatio-.8) - jitter),
	}, true
}

type MimirResourceUsage struct {
	CPUMCU, RAMBytes, DiskBytes, MemoryMMU, DiskReadIOPS, DiskWriteIOPS, NetworkUpload, NetworkDownload float64
}

// MimirPlacementCost applies absolute capability constraints and returns a
// lower-is-better cost that is independent of the other candidates.
func MimirPlacementCost(c *MimirNodeCapability, h *MimirNodeHealth, q *MimirWorkloadRequirement, used MimirResourceUsage, now time.Time) (float64, bool) {
	if q == nil {
		return 0, false
	}
	if q.BenchmarkWorkload || q.ControlPlaneWorkload {
		return 0, true
	}
	if c == nil || c.Validate() != nil {
		return 0, false
	}
	f, fresh := h.Factors(now)
	if !fresh {
		return 0, false
	}
	type dim struct{ cap, used, demand, health float64 }
	dims := []dim{
		{c.CPUTotalMCU * .80 * f.CPU, used.CPUMCU, q.CPUExpectedMCU, f.CPU},
		{float64(c.RAMBytes) * .85 * f.RAM, used.RAMBytes, float64(q.MemoryBytes), f.RAM},
		{float64(c.DiskBytes) * .85 * f.Disk, used.DiskBytes, float64(q.DiskBytes), f.Disk},
		{c.RAMMMU * .85 * f.RAM, used.MemoryMMU, q.MemoryBandwidthMMU, f.RAM},
		{c.DiskRandomReadIOPS * .85 * f.Disk, used.DiskReadIOPS, q.DiskReadIOPS, f.Disk},
		{c.DiskRandomWriteIOPS * .85 * f.Disk, used.DiskWriteIOPS, q.DiskWriteIOPS, f.Disk},
		{c.NetworkUploadBytesPerSecond * .80 * f.Network, used.NetworkUpload, q.NetworkUploadBytesPerSecond, f.Network},
		{c.NetworkDownloadBytesPerSecond * .80 * f.Network, used.NetworkDownload, q.NetworkDownloadBytesPerSecond, f.Network},
	}
	if q.CPUSingleMinimumMCU > c.CPUSingleMCU*.80*f.CPU || q.DiskFsyncMaxMicroseconds > 0 && c.DiskFsyncMicroseconds > q.DiskFsyncMaxMicroseconds {
		return 0, false
	}
	if q.DiskReadBytesPerSecond > c.DiskReadBytesPerSecond*.85*f.Disk || q.DiskWriteBytesPerSecond > c.DiskWriteBytesPerSecond*.85*f.Disk {
		return 0, false
	}
	remaining, healthPenalty := []float64{}, []float64{}
	for _, d := range dims {
		if d.demand <= 0 {
			continue
		}
		if d.cap <= 0 || d.used+d.demand > d.cap {
			return 0, false
		}
		remaining = append(remaining, (d.cap-d.used-d.demand)/d.cap)
		healthPenalty = append(healthPenalty, 1-d.health)
	}
	observedRTT := h.NetworkRTTMilliseconds
	if q.Region != "" {
		found := false
		for _, r := range c.Regions {
			if r.Region == q.Region && (q.Protocol == "" || r.Protocol == q.Protocol) {
				observedRTT = r.RTTMilliseconds
				found = true
				break
			}
		}
		if !found {
			return 0, false
		}
	}
	if len(remaining) == 0 {
		remaining, healthPenalty = []float64{0}, []float64{0}
	}
	sumR, sumH, minR, maxR := 0.0, 0.0, remaining[0], remaining[0]
	for i, r := range remaining {
		sumR += r
		sumH += healthPenalty[i]
		minR = math.Min(minR, r)
		maxR = math.Max(maxR, r)
	}
	latency := 0.0
	if q.RTTTargetMilliseconds > 0 {
		latency = math.Max(0, observedRTT/q.RTTTargetMilliseconds-1)
	}
	return sumR/float64(len(remaining)) + .25*(maxR-minR) + latency + .5*sumH/float64(len(healthPenalty)), true
}
