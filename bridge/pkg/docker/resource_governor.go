// Package docker provides resource governance for container isolation.
//
// Resolves: Gap - Agent Resource Isolation
//
// Enforces CPU, memory, and I/O limits on containers to prevent
// noisy neighbor issues and resource exhaustion attacks.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
)

// ResourceLimits defines resource constraints for containers
type ResourceLimits struct {
	// CPU limits
	CPUShares    int64   // CPU shares (relative weight)
	CPUQuota     int64   // CPU quota in microseconds per period
	CPUPeriod    int64   // CPU period in microseconds
	CPUPercent   float64 // Maximum CPU percent (0-100)
	CpusetCpus   string  // CPUs to use (e.g., "0-3" or "0,2")
	CpusetMems   string  // Memory nodes to use

	// Memory limits
	Memory            int64  // Memory limit in bytes
	MemorySwap        int64  // Total memory + swap limit
	MemoryReservation int64  // Memory soft limit
	MemorySwappiness  *int64 // Swappiness (0-100, nil for default)

	// I/O limits
	BlkioWeight       uint16  // Block IO weight (10-1000)
	BlkioReadBps      uint64  // Read bytes per second
	BlkioWriteBps     uint64  // Write bytes per second
	BlkioReadIOps     uint64  // Read I/O operations per second
	BlkioWriteIOps    uint64  // Write I/O operations per second

	// PIDs limit
	PidsLimit int64 // Maximum number of PIDs

	// Process limits
	MaxProcesses int64 // Maximum number of processes
	MaxOpenFiles uint64 // Maximum open file descriptors
}

// ResourceProfile represents predefined resource profiles
type ResourceProfile string

const (
	ProfileMinimal    ResourceProfile = "minimal"    // Minimal resources (CPU: 5%, Memory: 128MB)
	ProfileLight      ResourceProfile = "light"      // Light usage (CPU: 10%, Memory: 256MB)
	ProfileStandard   ResourceProfile = "standard"   // Standard usage (CPU: 25%, Memory: 512MB)
	ProfileHeavy      ResourceProfile = "heavy"      // Heavy usage (CPU: 50%, Memory: 1GB)
	ProfileUnlimited  ResourceProfile = "unlimited"  // No limits (not recommended)
)

// DefaultLimits returns the default resource limits for a profile
func DefaultLimits(profile ResourceProfile) ResourceLimits {
	switch profile {
	case ProfileMinimal:
		return ResourceLimits{
			CPUShares:        512,  // ~5% of CPU
			CPUPercent:       5,
			Memory:           128 * 1024 * 1024, // 128MB
			MemorySwap:       256 * 1024 * 1024, // 256MB total
			PidsLimit:        32,
			BlkioWeight:      100,
		}
	case ProfileLight:
		return ResourceLimits{
			CPUShares:        1024, // ~10% of CPU
			CPUPercent:       10,
			Memory:           256 * 1024 * 1024, // 256MB
			MemorySwap:       512 * 1024 * 1024, // 512MB total
			PidsLimit:        64,
			BlkioWeight:      200,
		}
	case ProfileStandard:
		return ResourceLimits{
			CPUShares:        2560, // ~25% of CPU
			CPUPercent:       25,
			Memory:           512 * 1024 * 1024, // 512MB
			MemorySwap:       1024 * 1024 * 1024, // 1GB total
			PidsLimit:        128,
			BlkioWeight:      500,
		}
	case ProfileHeavy:
		return ResourceLimits{
			CPUShares:        5120, // ~50% of CPU
			CPUPercent:       50,
			Memory:           1024 * 1024 * 1024, // 1GB
			MemorySwap:       2048 * 1024 * 1024, // 2GB total
			PidsLimit:        256,
			BlkioWeight:      800,
		}
	case ProfileUnlimited:
		return ResourceLimits{} // No limits
	default:
		return DefaultLimits(ProfileStandard)
	}
}

// ResourceUsage represents current resource usage of a container
type ResourceUsage struct {
	ContainerID    string    `json:"container_id"`
	CPUPercent     float64   `json:"cpu_percent"`
	MemoryUsage    int64     `json:"memory_usage"`
	MemoryLimit    int64     `json:"memory_limit"`
	MemoryPercent  float64   `json:"memory_percent"`
	NetworkRxBytes uint64    `json:"network_rx_bytes"`
	NetworkTxBytes uint64    `json:"network_tx_bytes"`
	BlockRead      uint64    `json:"block_read"`
	BlockWrite     uint64    `json:"block_write"`
	PIDs           int64     `json:"pids"`
	Timestamp      time.Time `json:"timestamp"`
}

// ResourceViolation represents a resource limit violation
type ResourceViolation struct {
	ContainerID string    `json:"container_id"`
	Type        string    `json:"type"` // cpu, memory, io, pids
	Current     float64   `json:"current"`
	Limit       float64   `json:"limit"`
	PercentOver float64   `json:"percent_over"`
	Timestamp   time.Time `json:"timestamp"`
	Action      string    `json:"action"` // warn, throttle, terminate
}

// ResourceGovernor manages resource limits and monitoring
type ResourceGovernor struct {
	client       *Client
	logger       *slog.Logger
	limits       ResourceLimits
	thresholds   AlertThresholds
	mu           sync.RWMutex
	usageCache   map[string]ResourceUsage
	violations   []ResourceViolation
	stopMonitor  chan struct{}
	monitoring   bool
}

// AlertThresholds defines when to alert on resource usage
type AlertThresholds struct {
	CPUWarn      float64 // CPU percent to warn (e.g., 80)
	CPUCritical  float64 // CPU percent for critical alert
	MemWarn      float64 // Memory percent to warn
	MemCritical  float64 // Memory percent for critical alert
	IOWarn       float64 // I/O percent to warn
	IOCritical   float64 // I/O percent for critical alert
}

// DefaultAlertThresholds returns default alert thresholds
func DefaultAlertThresholds() AlertThresholds {
	return AlertThresholds{
		CPUWarn:     80,
		CPUCritical: 95,
		MemWarn:     80,
		MemCritical: 95,
		IOWarn:      80,
		IOCritical:  95,
	}
}

// containerStats represents Docker container stats (local definition for API response)
type containerStats struct {
	Read        time.Time `json:"read"`
	PreRead     time.Time `json:"preread"`
	PidsStats   struct {
		Current int64 `json:"current"`
	} `json:"pids_stats"`
	CPUStats struct {
		CPUUsage struct {
			TotalUsage  uint64   `json:"total_usage"`
			PercpuUsage []uint64 `json:"percpu_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage  uint64   `json:"total_usage"`
			PercpuUsage []uint64 `json:"percpu_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64             `json:"usage"`
		Stats map[string]float64 `json:"stats"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
	BlkioStats struct {
		IOServiceBytesRecursive []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_service_bytes_recursive"`
	} `json:"blkio_stats"`
}

// NewResourceGovernor creates a new resource governor
func NewResourceGovernor(client *Client, profile ResourceProfile, logger *slog.Logger) *ResourceGovernor {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})).With("component", "resource_governor")
	}

	return &ResourceGovernor{
		client:      client,
		logger:      logger,
		limits:      DefaultLimits(profile),
		thresholds:  DefaultAlertThresholds(),
		usageCache:  make(map[string]ResourceUsage),
		violations:  make([]ResourceViolation, 0),
		stopMonitor: make(chan struct{}),
	}
}

// SetLimits sets custom resource limits
func (g *ResourceGovernor) SetLimits(limits ResourceLimits) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.limits = limits
}

// GetLimits returns current resource limits
func (g *ResourceGovernor) GetLimits() ResourceLimits {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.limits
}

// SetThresholds sets alert thresholds
func (g *ResourceGovernor) SetThresholds(thresholds AlertThresholds) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.thresholds = thresholds
}

// ApplyToHostConfig applies resource limits to a container host config
func (g *ResourceGovernor) ApplyToHostConfig(hostConfig *container.HostConfig) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if hostConfig == nil {
		return fmt.Errorf("host config is nil")
	}

	// Apply CPU limits
	if g.limits.CPUShares > 0 {
		hostConfig.CPUShares = g.limits.CPUShares
	}
	if g.limits.CPUQuota > 0 {
		hostConfig.CPUQuota = g.limits.CPUQuota
	}
	if g.limits.CPUPeriod > 0 {
		hostConfig.CPUPeriod = g.limits.CPUPeriod
	}
	if g.limits.CpusetCpus != "" {
		hostConfig.CpusetCpus = g.limits.CpusetCpus
	}
	if g.limits.CpusetMems != "" {
		hostConfig.CpusetMems = g.limits.CpusetMems
	}

	// Apply memory limits
	if g.limits.Memory > 0 {
		hostConfig.Memory = g.limits.Memory
	}
	if g.limits.MemorySwap > 0 {
		hostConfig.MemorySwap = g.limits.MemorySwap
	}
	if g.limits.MemoryReservation > 0 {
		hostConfig.MemoryReservation = g.limits.MemoryReservation
	}
	if g.limits.MemorySwappiness != nil {
		hostConfig.MemorySwappiness = g.limits.MemorySwappiness
	}

	// Apply I/O limits
	if g.limits.BlkioWeight > 0 {
		hostConfig.BlkioWeight = g.limits.BlkioWeight
	}
	// Note: Blkio BPS and IOPS limits require specific device paths
	// These are applied at container creation with device mappings

	// Apply PIDs limit
	if g.limits.PidsLimit > 0 {
		hostConfig.PidsLimit = &g.limits.PidsLimit
	}

	g.logger.Info("resource_limits_applied",
		"cpu_shares", g.limits.CPUShares,
		"memory_mb", g.limits.Memory/1024/1024,
		"pids_limit", g.limits.PidsLimit,
	)

	return nil
}

// GetContainerUsage retrieves resource usage for a container
func (g *ResourceGovernor) GetContainerUsage(ctx context.Context, containerID string) (*ResourceUsage, error) {
	stats, err := g.client.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer stats.Body.Close()

	var statsJSON containerStats
	if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	usage := &ResourceUsage{
		ContainerID: containerID,
		Timestamp:   time.Now(),
		PIDs:        statsJSON.PidsStats.Current,
	}

	// Calculate CPU percent
	cpuDelta := float64(statsJSON.CPUStats.CPUUsage.TotalUsage - statsJSON.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(statsJSON.CPUStats.SystemUsage - statsJSON.PreCPUStats.SystemUsage)
	if systemDelta > 0 && cpuDelta > 0 {
		usage.CPUPercent = (cpuDelta / systemDelta) * float64(len(statsJSON.CPUStats.CPUUsage.PercpuUsage)) * 100
	}

	// Memory usage
	usage.MemoryUsage = int64(statsJSON.MemoryStats.Usage)
	if limit, ok := statsJSON.MemoryStats.Stats["limit"]; ok {
		usage.MemoryLimit = int64(limit)
	}
	if usage.MemoryLimit > 0 {
		usage.MemoryPercent = float64(usage.MemoryUsage) / float64(usage.MemoryLimit) * 100
	}

	// Network I/O
	for _, nw := range statsJSON.Networks {
		usage.NetworkRxBytes += nw.RxBytes
		usage.NetworkTxBytes += nw.TxBytes
	}

	// Block I/O
	for _, blk := range statsJSON.BlkioStats.IOServiceBytesRecursive {
		if blk.Op == "Read" {
			usage.BlockRead += blk.Value
		} else if blk.Op == "Write" {
			usage.BlockWrite += blk.Value
		}
	}

	// Cache usage
	g.mu.Lock()
	g.usageCache[containerID] = *usage
	g.mu.Unlock()

	return usage, nil
}

// CheckViolations checks if container is violating resource limits
func (g *ResourceGovernor) CheckViolations(usage *ResourceUsage) []ResourceViolation {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var violations []ResourceViolation
	now := time.Now()

	// Check CPU violations
	if g.limits.CPUPercent > 0 && usage.CPUPercent > g.limits.CPUPercent {
		percentOver := (usage.CPUPercent / g.limits.CPUPercent - 1) * 100
		action := "warn"
		if usage.CPUPercent > g.thresholds.CPUCritical {
			action = "throttle"
		}

		violations = append(violations, ResourceViolation{
			ContainerID: usage.ContainerID,
			Type:        "cpu",
			Current:     usage.CPUPercent,
			Limit:       g.limits.CPUPercent,
			PercentOver: percentOver,
			Timestamp:   now,
			Action:      action,
		})
	}

	// Check memory violations
	if g.limits.Memory > 0 && usage.MemoryUsage > int64(float64(g.limits.Memory)*g.thresholds.MemWarn/100) {
		percentOver := float64(usage.MemoryUsage) / float64(g.limits.Memory) * 100
		action := "warn"
		if percentOver > g.thresholds.MemCritical {
			action = "throttle"
		}

		violations = append(violations, ResourceViolation{
			ContainerID: usage.ContainerID,
			Type:        "memory",
			Current:     float64(usage.MemoryUsage),
			Limit:       float64(g.limits.Memory),
			PercentOver: percentOver - 100,
			Timestamp:   now,
			Action:      action,
		})
	}

	// Check PIDs limit
	if g.limits.PidsLimit > 0 && usage.PIDs > g.limits.PidsLimit {
		percentOver := float64(usage.PIDs) / float64(g.limits.PidsLimit) * 100
		violations = append(violations, ResourceViolation{
			ContainerID: usage.ContainerID,
			Type:        "pids",
			Current:     float64(usage.PIDs),
			Limit:       float64(g.limits.PidsLimit),
			PercentOver: percentOver - 100,
			Timestamp:   now,
			Action:      "terminate", // PIDs limit breach is serious
		})
	}

	// Store violations
	if len(violations) > 0 {
		g.mu.Lock()
		g.violations = append(g.violations, violations...)
		g.mu.Unlock()

		for _, v := range violations {
			g.logger.Warn("resource_violation",
				"container_id", v.ContainerID,
				"type", v.Type,
				"current", v.Current,
				"limit", v.Limit,
				"action", v.Action,
			)
		}
	}

	return violations
}

// StartMonitoring starts periodic resource monitoring
func (g *ResourceGovernor) StartMonitoring(interval time.Duration, containerIDs []string) {
	g.mu.Lock()
	if g.monitoring {
		g.mu.Unlock()
		return
	}
	g.monitoring = true
	g.mu.Unlock()

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for _, id := range containerIDs {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					usage, err := g.GetContainerUsage(ctx, id)
					cancel()

					if err != nil {
						g.logger.Warn("monitoring_failed",
							"container_id", id,
							"error", err,
						)
						continue
					}

					g.CheckViolations(usage)
				}
			case <-g.stopMonitor:
				g.logger.Info("monitoring_stopped")
				return
			}
		}
	}()

	g.logger.Info("monitoring_started",
		"interval", interval,
		"containers", len(containerIDs),
	)
}

// StopMonitoring stops resource monitoring
func (g *ResourceGovernor) StopMonitoring() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.monitoring {
		close(g.stopMonitor)
		g.monitoring = false
		g.stopMonitor = make(chan struct{})
	}
}

// GetViolations returns recorded violations
func (g *ResourceGovernor) GetViolations(limit int) []ResourceViolation {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if limit <= 0 || limit > len(g.violations) {
		limit = len(g.violations)
	}

	start := len(g.violations) - limit
	if start < 0 {
		start = 0
	}

	result := make([]ResourceViolation, limit)
	copy(result, g.violations[start:])
	return result
}

// ClearViolations clears violation history
func (g *ResourceGovernor) ClearViolations() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.violations = make([]ResourceViolation, 0)
}

// GetCachedUsage returns cached usage data
func (g *ResourceGovernor) GetCachedUsage(containerID string) (ResourceUsage, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	usage, ok := g.usageCache[containerID]
	return usage, ok
}

// RecommendProfile recommends a resource profile based on system resources
func RecommendProfile() ResourceProfile {
	cpuCount := runtime.NumCPU()

	// Get total system memory
	var totalMemory uint64
	// Note: This is a simplified check; real implementation would use syscalls
	// For now, we'll use CPU count as a proxy
	totalMemory = uint64(cpuCount) * 1024 * 1024 * 1024 // Assume 1GB per core

	// Recommend based on system capacity
	if cpuCount <= 2 || totalMemory < 4*1024*1024*1024 {
		return ProfileLight
	} else if cpuCount <= 4 || totalMemory < 8*1024*1024*1024 {
		return ProfileStandard
	}
	return ProfileHeavy
}
