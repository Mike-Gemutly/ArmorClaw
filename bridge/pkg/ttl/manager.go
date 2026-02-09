// Package ttl provides container time-to-live management for ArmorClaw
package ttl

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ContainerState tracks a container's activity and TTL status
type ContainerState struct {
	ContainerID string
	SessionID    string
	LastActive  time.Time
	CreatedAt    time.Time
	Labels      map[string]string
}

// DockerClient defines the interface for Docker operations
type DockerClient interface {
	RemoveContainer(containerID string) error
	StopContainer(containerID string) error
}

// Manager manages container TTL and auto-cleanup
type Manager struct {
	idleTimeout   time.Duration
	checkInterval time.Duration
	containers    map[string]*ContainerState
	mutex         sync.RWMutex
	dockerClient  DockerClient
	ctx           context.Context
	cancel        context.CancelFunc
	logger        *log.Logger
}

// NewManager creates a new TTL manager
func NewManager(idleTimeout time.Duration, dockerClient DockerClient) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		idleTimeout:   idleTimeout,
		checkInterval: 1 * time.Minute,
		containers:    make(map[string]*ContainerState),
		dockerClient:  dockerClient,
		ctx:           ctx,
		cancel:        cancel,
		logger:        log.Default(),
	}
}

// SetLogger sets a custom logger for the TTL manager
func (m *Manager) SetLogger(logger *log.Logger) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.logger = logger
}

// Register registers a container for TTL monitoring
func (m *Manager) Register(containerID, sessionID string, labels map[string]string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	m.containers[containerID] = &ContainerState{
		ContainerID: containerID,
		SessionID:    sessionID,
		LastActive:  now,
		CreatedAt:    now,
		Labels:      labels,
	}

	m.logger.Printf("[TTL] Registered container %s (session: %s, idle timeout: %v)",
		containerID, sessionID, m.idleTimeout)
}

// Unregister removes a container from TTL monitoring
func (m *Manager) Unregister(containerID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if state, exists := m.containers[containerID]; exists {
		delete(m.containers, containerID)
		m.logger.Printf("[TTL] Unregistered container %s (session: %s)",
			containerID, state.SessionID)
	}
}

// Heartbeat updates the last active time for a container
func (m *Manager) Heartbeat(containerID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	state, exists := m.containers[containerID]
	if !exists {
		return fmt.Errorf("container not registered: %s", containerID)
	}

	previousActive := state.LastActive
	state.LastActive = time.Now()

	// Log if container was inactive for a while
	idleTime := time.Since(previousActive)
	if idleTime > 5*time.Minute {
		m.logger.Printf("[TTL] Container %s heartbeat after %v idle",
			containerID, idleTime.Round(time.Second))
	}

	return nil
}

// GetState returns the current state of a container
func (m *Manager) GetState(containerID string) (*ContainerState, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	state, exists := m.containers[containerID]
	if !exists {
		return nil, fmt.Errorf("container not registered: %s", containerID)
	}

	// Return a copy to prevent external modification
	copy := *state
	return &copy, nil
}

// GetIdleTime returns how long a container has been idle
func (m *Manager) GetIdleTime(containerID string) (time.Duration, error) {
	state, err := m.GetState(containerID)
	if err != nil {
		return 0, err
	}

	return time.Since(state.LastActive), nil
}

// Start begins the background cleanup loop
func (m *Manager) Start() {
	m.logger.Printf("[TTL] Starting TTL manager (idle timeout: %v, check interval: %v)",
		m.idleTimeout, m.checkInterval)

	go m.cleanupLoop()
}

// Stop gracefully shuts down the TTL manager
func (m *Manager) Stop() {
	m.logger.Printf("[TTL] Stopping TTL manager")
	m.cancel()
}

// cleanupLoop runs the continuous cleanup loop
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Printf("[TTL] Cleanup loop stopped")
			return
		case <-ticker.C:
			m.cleanupIdleContainers()
		}
	}
}

// cleanupIdleContainers removes containers that have been idle too long
func (m *Manager) cleanupIdleContainers() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	var cleanedCount int

	for containerID, state := range m.containers {
		idleTime := now.Sub(state.LastActive)

		if idleTime > m.idleTimeout {
			m.logger.Printf("[TTL] Container %s idle for %v (timeout: %v), removing...",
				containerID, idleTime.Round(time.Second), m.idleTimeout)

			// Stop container first (graceful shutdown)
			if err := m.dockerClient.StopContainer(containerID); err != nil {
				m.logger.Printf("[TTL] Warning: failed to stop container %s: %v",
					containerID, err)
			}

			// Remove container
			if err := m.dockerClient.RemoveContainer(containerID); err != nil {
				m.logger.Printf("[TTL] Error: failed to remove container %s: %v",
					containerID, err)
				continue
			}

			// Unregister from tracking
			delete(m.containers, containerID)
			cleanedCount++

			m.logger.Printf("[TTL] Removed idle container: %s", containerID)
		}
	}

	if cleanedCount > 0 {
		m.logger.Printf("[TTL] Cleanup complete: removed %d idle container(s)", cleanedCount)
	}
}

// ForceRemove immediately removes a container regardless of idle time
func (m *Manager) ForceRemove(containerID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	state, exists := m.containers[containerID]
	if !exists {
		return fmt.Errorf("container not registered: %s", containerID)
	}

	m.logger.Printf("[TTL] Force removing container %s (session: %s)",
		containerID, state.SessionID)

	// Stop container first
	if err := m.dockerClient.StopContainer(containerID); err != nil {
		m.logger.Printf("[TTL] Warning: failed to stop container %s: %v",
			containerID, err)
	}

	// Remove container
	if err := m.dockerClient.RemoveContainer(containerID); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Unregister from tracking
	delete(m.containers, containerID)

	m.logger.Printf("[TTL] Force removed container: %s", containerID)
	return nil
}

// GetContainerCount returns the number of tracked containers
func (m *Manager) GetContainerCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.containers)
}

// GetActiveContainers returns a list of all tracked container IDs
func (m *Manager) GetActiveContainers() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	containers := make([]string, 0, len(m.containers))
	for containerID := range m.containers {
		containers = append(containers, containerID)
	}
	return containers
}

// GetIdleContainers returns containers that have been idle longer than threshold
func (m *Manager) GetIdleContainers(threshold time.Duration) []*ContainerState {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	now := time.Now()
	idleContainers := make([]*ContainerState, 0)

	for _, state := range m.containers {
		idleTime := now.Sub(state.LastActive)
		if idleTime > threshold {
			copy := *state
			idleContainers = append(idleContainers, &copy)
		}
	}

	return idleContainers
}

// ExtendIdleTime extends the idle timeout for testing purposes
func (m *Manager) ExtendIdleTime(containerID string, additionalTime time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	state, exists := m.containers[containerID]
	if !exists {
		return fmt.Errorf("container not registered: %s", containerID)
	}

	state.LastActive = state.LastActive.Add(additionalTime)
	m.logger.Printf("[TTL] Extended idle time for %s by %v",
		containerID, additionalTime.Round(time.Second))

	return nil
}

// SetIdleTimeout updates the idle timeout for the manager
func (m *Manager) SetIdleTimeout(timeout time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.idleTimeout = timeout
	m.logger.Printf("[TTL] Idle timeout updated to %v", timeout)
}

// GetIdleTimeout returns the current idle timeout
func (m *Manager) GetIdleTimeout() time.Duration {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.idleTimeout
}

// GetStats returns statistics about managed containers
func (m *Manager) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	now := time.Now()
	activeCount := 0
	idleCount := 0
	totalAge := time.Duration(0)

	for _, state := range m.containers {
		idleTime := now.Sub(state.LastActive)
		totalAge += now.Sub(state.CreatedAt)

		if idleTime > m.idleTimeout {
			idleCount++
		} else {
			activeCount++
		}
	}

	avgAge := time.Duration(0)
	if len(m.containers) > 0 {
		avgAge = totalAge / time.Duration(len(m.containers))
	}

	return map[string]interface{}{
		"total_containers":   len(m.containers),
		"active_containers":  activeCount,
		"idle_containers":    idleCount,
		"idle_timeout":      m.idleTimeout.String(),
		"average_age":       avgAge.String(),
	}
}
