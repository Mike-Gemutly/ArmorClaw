// Package health provides container health monitoring for ArmorClaw
// This ensures containers are running correctly and can be recovered if they fail
package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/docker"
	"github.com/armorclaw/bridge/pkg/logger"
	"log/slog"
)

// Monitor tracks container health and takes recovery actions
type Monitor struct {
	dockerClient *docker.Client
	checkInterval time.Duration
	maxFailures  int
	containers   map[string]*ContainerHealth
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	securityLog  *logger.SecurityLogger
	onFailure    FailureHandler
}

// ContainerHealth holds health status for a container
type ContainerHealth struct {
	ID           string
	Name         string
	State        string
	FailureCount int
	LastCheck    time.Time
	LastHealthy  time.Time
	mu           sync.RWMutex
}

// Copy returns a copy of the ContainerHealth without the mutex
func (h *ContainerHealth) Copy() *ContainerHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return &ContainerHealth{
		ID:           h.ID,
		Name:         h.Name,
		State:        h.State,
		FailureCount: h.FailureCount,
		LastCheck:    h.LastCheck,
		LastHealthy:  h.LastHealthy,
	}
}

// FailureHandler is called when a container fails
type FailureHandler func(containerID, containerName, reason string)

// MonitorConfig holds configuration for health monitoring
type MonitorConfig struct {
	CheckInterval   time.Duration // How often to check container health
	MaxFailures     int           // Max consecutive failures before action
	MaxStaleness    time.Duration // Max time since last health check
	RestartOnFailure bool         // Automatically restart failed containers
}

// DefaultMonitorConfig returns default monitoring configuration
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		CheckInterval:    30 * time.Second,
		MaxFailures:      3,
		MaxStaleness:     5 * time.Minute,
		RestartOnFailure: false, // Manual intervention by default
	}
}

// NewMonitor creates a new container health monitor
func NewMonitor(dockerClient *docker.Client, config MonitorConfig) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())

	if config.CheckInterval == 0 {
		config.CheckInterval = DefaultMonitorConfig().CheckInterval
	}
	if config.MaxFailures == 0 {
		config.MaxFailures = DefaultMonitorConfig().MaxFailures
	}

	return &Monitor{
		dockerClient: dockerClient,
		checkInterval: config.CheckInterval,
		maxFailures:  config.MaxFailures,
		containers:   make(map[string]*ContainerHealth),
		ctx:          ctx,
		cancel:       cancel,
		securityLog:  logger.NewSecurityLogger(logger.Global().WithComponent("health_monitor")),
	}
}

// SetFailureHandler sets a custom handler for container failures
func (m *Monitor) SetFailureHandler(handler FailureHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onFailure = handler
}

// Start begins monitoring containers
func (m *Monitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.wg.Add(1)
	go m.monitorLoop()

	m.securityLog.LogSecurityEvent("health_monitor_started",
		slog.Duration("check_interval", m.checkInterval),
		slog.Int("max_failures", m.maxFailures))

	return nil
}

// Stop stops monitoring containers
func (m *Monitor) Stop() {
	m.cancel()
	m.wg.Wait()
	m.securityLog.LogSecurityEvent("health_monitor_stopped")
}

// Register adds a container to be monitored
func (m *Monitor) Register(containerID, containerName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	health := &ContainerHealth{
		ID:          containerID,
		Name:        containerName,
		State:       "unknown",
		LastCheck:   time.Now(),
		LastHealthy: time.Now(),
	}

	m.containers[containerID] = health

	m.securityLog.LogSecurityEvent("container_registered",
		slog.String("container_id", containerID),
		slog.String("container_name", containerName))
}

// Unregister removes a container from monitoring
func (m *Monitor) Unregister(containerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.containers, containerID)

	m.securityLog.LogSecurityEvent("container_unregistered",
		slog.String("container_id", containerID))
}

// UpdateHealth updates the health status for a container
func (m *Monitor) UpdateHealth(containerID, state string, isHealthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	health, exists := m.containers[containerID]
	if !exists {
		return
	}

	health.mu.Lock()
	defer health.mu.Unlock()

	health.State = state
	health.LastCheck = time.Now()

	if isHealthy {
		health.FailureCount = 0
		health.LastHealthy = time.Now()
	}

	m.securityLog.LogSecurityEvent("container_health_updated",
		slog.String("container_id", containerID),
		slog.String("state", state),
		slog.Bool("healthy", isHealthy),
		slog.Int("failure_count", health.FailureCount))
}

// GetHealth returns the health status for a container
func (m *Monitor) GetHealth(containerID string) (*ContainerHealth, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health, exists := m.containers[containerID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	return health.Copy(), true
}

// ListHealth returns health status for all monitored containers
func (m *Monitor) ListHealth() []*ContainerHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	healthList := make([]*ContainerHealth, 0, len(m.containers))
	for _, health := range m.containers {
		healthList = append(healthList, health.Copy())
	}

	return healthList
}

// monitorLoop is the main monitoring loop
func (m *Monitor) monitorLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkAllContainers()
		}
	}
}

// checkAllContainers checks health of all registered containers
func (m *Monitor) checkAllContainers() {
	m.mu.RLock()
	containerIDs := make([]string, 0, len(m.containers))
	for id := range m.containers {
		containerIDs = append(containerIDs, id)
	}
	m.mu.RUnlock()

	for _, containerID := range containerIDs {
		m.checkContainer(containerID)
	}
}

// checkContainer checks the health of a single container
func (m *Monitor) checkContainer(containerID string) {
	m.mu.RLock()
	health, exists := m.containers[containerID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	// Check if container is still running
	isRunning, err := m.dockerClient.IsRunning(containerID)
	health.mu.Lock()
	defer health.mu.Unlock()

	health.LastCheck = time.Now()

	if err != nil {
		// Error checking container status
		health.FailureCount++
		health.State = "error"

		m.securityLog.LogSecurityEvent("container_health_check_error",
			slog.String("container_id", containerID),
			slog.String("error", err.Error()),
			slog.Int("failure_count", health.FailureCount))

		if health.FailureCount >= m.maxFailures {
			m.handleFailure(containerID, health.Name, "health_check_error")
		}
		return
	}

	if !isRunning {
		// Container is not running
		health.FailureCount++
		health.State = "stopped"

		m.securityLog.LogSecurityEvent("container_not_running",
			slog.String("container_id", containerID),
			slog.Int("failure_count", health.FailureCount))

		if health.FailureCount >= m.maxFailures {
			m.handleFailure(containerID, health.Name, "container_stopped")
		}
		return
	}

	// Container is running
	health.State = "running"
	health.FailureCount = 0
	health.LastHealthy = time.Now()
}

// handleFailure handles a container failure
func (m *Monitor) handleFailure(containerID, containerName, reason string) {
	reasonMsg := fmt.Sprintf("%s: %s", reason, containerName)

	m.securityLog.LogSecurityEvent("container_failure_detected",
		slog.String("container_id", containerID),
		slog.String("container_name", containerName),
		slog.String("reason", reason),
		slog.Int("failure_count", m.maxFailures))

	// Call custom failure handler if set
	if m.onFailure != nil {
		m.onFailure(containerID, containerName, reasonMsg)
	}
}

// GetStats returns monitoring statistics
func (m *Monitor) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"monitored_containers": len(m.containers),
		"check_interval":       m.checkInterval.String(),
		"max_failures":         m.maxFailures,
	}

	healthyCount := 0
	unhealthyCount := 0
	unknownCount := 0

	for _, health := range m.containers {
		health.mu.RLock()
		switch health.State {
		case "running":
			healthyCount++
		case "stopped", "error":
			unhealthyCount++
		default:
			unknownCount++
		}
		health.mu.RUnlock()
	}

	stats["healthy"] = healthyCount
	stats["unhealthy"] = unhealthyCount
	stats["unknown"] = unknownCount

	return stats
}
