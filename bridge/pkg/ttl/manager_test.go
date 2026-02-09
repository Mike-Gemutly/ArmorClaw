// Package ttl provides tests for the TTL manager
package ttl

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// MockDockerClient is a mock implementation of DockerClient for testing
type MockDockerClient struct {
	removedContainers []string
	stoppedContainers  []string
	stopError         error
	removeError        error
	mu                sync.Mutex
}

func (m *MockDockerClient) RemoveContainer(containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.removeError != nil {
		return m.removeError
	}

	m.removedContainers = append(m.removedContainers, containerID)
	return nil
}

func (m *MockDockerClient) StopContainer(containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopError != nil {
		return m.stopError
	}

	m.stoppedContainers = append(m.stoppedContainers, containerID)
	return nil
}

func (m *MockDockerClient) WasRemoved(containerID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range m.removedContainers {
		if id == containerID {
			return true
		}
	}
	return false
}

func (m *MockDockerClient) WasStopped(containerID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range m.stoppedContainers {
		if id == containerID {
			return true
		}
	}
	return false
}

func (m *MockDockerClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removedContainers = nil
	m.stoppedContainers = nil
	m.stopError = nil
	m.removeError = nil
}

func TestNewManager(t *testing.T) {
	mock := &MockDockerClient{}
	idleTimeout := 10 * time.Minute

	manager := NewManager(idleTimeout, mock)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.GetIdleTimeout() != idleTimeout {
		t.Errorf("Expected idle timeout %v, got %v", idleTimeout, manager.GetIdleTimeout())
	}
}

func TestRegister(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(5*time.Minute, mock)

	containerID := "test-container-1"
	sessionID := "session-123"

	manager.Register(containerID, sessionID, map[string]string{"app": "test"})

	state, err := manager.GetState(containerID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}

	if state.ContainerID != containerID {
		t.Errorf("Expected container ID %s, got %s", containerID, state.ContainerID)
	}

	if state.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, state.SessionID)
	}

	if manager.GetContainerCount() != 1 {
		t.Errorf("Expected 1 container, got %d", manager.GetContainerCount())
	}
}

func TestUnregister(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(5*time.Minute, mock)

	containerID := "test-container-1"
	sessionID := "session-123"

	manager.Register(containerID, sessionID, nil)
	manager.Unregister(containerID)

	if manager.GetContainerCount() != 0 {
		t.Errorf("Expected 0 containers after unregister, got %d", manager.GetContainerCount())
	}

	_, err := manager.GetState(containerID)
	if err == nil {
		t.Error("Expected error getting unregistered container state")
	}
}

func TestHeartbeat(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(5*time.Minute, mock)

	containerID := "test-container-1"
	sessionID := "session-123"

	manager.Register(containerID, sessionID, nil)

	// Get initial state
	state1, _ := manager.GetState(containerID)
	initialActive := state1.LastActive

	// Wait a bit and heartbeat
	time.Sleep(10 * time.Millisecond)
	err := manager.Heartbeat(containerID)
	if err != nil {
		t.Errorf("Heartbeat failed: %v", err)
	}

	// Check that LastActive was updated
	state2, _ := manager.GetState(containerID)
	if !state2.LastActive.After(initialActive) {
		t.Error("LastActive should have been updated by heartbeat")
	}
}

func TestHeartbeatNonExistent(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(5*time.Minute, mock)

	err := manager.Heartbeat("non-existent")
	if err == nil {
		t.Error("Expected error for heartbeat on non-existent container")
	}
}

func TestGetIdleTime(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(5*time.Minute, mock)

	containerID := "test-container-1"
	manager.Register(containerID, "session-123", nil)

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	idleTime, err := manager.GetIdleTime(containerID)
	if err != nil {
		t.Errorf("GetIdleTime failed: %v", err)
	}

	if idleTime < 10*time.Millisecond {
		t.Errorf("Expected idle time >= 10ms, got %v", idleTime)
	}
}

func TestForceRemove(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(5*time.Minute, mock)

	containerID := "test-container-1"
	sessionID := "session-123"

	manager.Register(containerID, sessionID, nil)

	// Force remove
	err := manager.ForceRemove(containerID)
	if err != nil {
		t.Errorf("ForceRemove failed: %v", err)
	}

	// Verify container was stopped and removed
	if !mock.WasStopped(containerID) {
		t.Error("Container should have been stopped")
	}

	if !mock.WasRemoved(containerID) {
		t.Error("Container should have been removed")
	}

	// Verify container was unregistered
	if manager.GetContainerCount() != 0 {
		t.Errorf("Expected 0 containers after force remove, got %d", manager.GetContainerCount())
	}
}

func TestForceRemoveNonExistent(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(5*time.Minute, mock)

	err := manager.ForceRemove("non-existent")
	if err == nil {
		t.Error("Expected error for force removing non-existent container")
	}
}

func TestGetActiveContainers(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(5*time.Minute, mock)

	// Register multiple containers
	containers := []string{"container-1", "container-2", "container-3"}
	for _, id := range containers {
		manager.Register(id, "session-"+id, nil)
	}

	active := manager.GetActiveContainers()
	if len(active) != len(containers) {
		t.Errorf("Expected %d active containers, got %d", len(containers), len(active))
	}

	for _, id := range containers {
		found := false
		for _, activeID := range active {
			if activeID == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Container %s not found in active list", id)
		}
	}
}

func TestGetIdleContainers(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(100*time.Millisecond, mock)

	// Register containers
	container1 := "container-1"
	container2 := "container-2"
	manager.Register(container1, "session-1", nil)
	manager.Register(container2, "session-2", nil)

	// Heartbeat one container to make it active
	manager.Heartbeat(container1)

	// Wait for other container to become idle
	time.Sleep(150 * time.Millisecond)

	// Get idle containers (threshold = 50ms)
	idleContainers := manager.GetIdleContainers(50 * time.Millisecond)

	// Should have at least 1 idle container
	if len(idleContainers) < 1 {
		t.Errorf("Expected at least 1 idle container, got %d", len(idleContainers))
	}
}

func TestCleanupLoop(t *testing.T) {
	mock := &MockDockerClient{}
	idleTimeout := 100 * time.Millisecond
	manager := NewManager(idleTimeout, mock)

	// Override check interval for faster testing
	manager.checkInterval = 50 * time.Millisecond

	// Register container
	containerID := "test-container-1"
	sessionID := "session-123"
	manager.Register(containerID, sessionID, nil)

	// Start manager
	manager.Start()
	defer manager.Stop()

	// Wait for container to become idle and be cleaned up
	// Need to wait for: idle time (100ms) + check interval (50ms) + processing time
	time.Sleep(300 * time.Millisecond)

	// Verify container was removed
	if !mock.WasRemoved(containerID) {
		t.Error("Container should have been removed by cleanup loop")
	}

	// Verify container was unregistered
	if manager.GetContainerCount() != 0 {
		t.Errorf("Expected 0 containers after cleanup, got %d", manager.GetContainerCount())
	}
}

func TestExtendIdleTime(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(100*time.Millisecond, mock)

	containerID := "test-container-1"
	manager.Register(containerID, "session-123", nil)

	// Get initial state
	state1, _ := manager.GetState(containerID)
	initialActive := state1.LastActive

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Extend idle time
	err := manager.ExtendIdleTime(containerID, 200*time.Millisecond)
	if err != nil {
		t.Errorf("ExtendIdleTime failed: %v", err)
	}

	// Check that LastActive was extended
	state2, _ := manager.GetState(containerID)
	expectedActive := initialActive.Add(200 * time.Millisecond)

	// Allow some tolerance for timing
	diff := state2.LastActive.Sub(expectedActive)
	if diff < -10*time.Millisecond || diff > 10*time.Millisecond {
		t.Errorf("LastActive not extended correctly. Expected around %v, got %v (diff: %v)",
			expectedActive, state2.LastActive, diff)
	}
}

func TestSetIdleTimeout(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(5*time.Minute, mock)

	newTimeout := 15 * time.Minute
	manager.SetIdleTimeout(newTimeout)

	if manager.GetIdleTimeout() != newTimeout {
		t.Errorf("Expected idle timeout %v, got %v", newTimeout, manager.GetIdleTimeout())
	}
}

func TestGetStats(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(1*time.Minute, mock)

	// Register containers
	manager.Register("container-1", "session-1", nil)
	manager.Register("container-2", "session-2", nil)

	// Both are active initially (just registered)
	stats := manager.GetStats()

	if stats["total_containers"].(int) != 2 {
		t.Errorf("Expected 2 total containers, got %v", stats["total_containers"])
	}

	// Both should be active (neither has been idle for > 1 minute)
	if stats["active_containers"].(int) != 2 {
		t.Errorf("Expected 2 active containers initially, got %v", stats["active_containers"])
	}

	if stats["idle_containers"].(int) != 0 {
		t.Errorf("Expected 0 idle containers initially, got %v", stats["idle_containers"])
	}

	if stats["idle_timeout"].(string) != "1m0s" {
		t.Errorf("Expected idle timeout 1m0s, got %v", stats["idle_timeout"])
	}
}

func TestDockerClientErrors(t *testing.T) {
	mock := &MockDockerClient{
		removeError: errors.New("remove failed"),
	}

	manager := NewManager(5*time.Minute, mock)

	containerID := "test-container-1"
	manager.Register(containerID, "session-123", nil)

	// Try to force remove - should fail
	err := manager.ForceRemove(containerID)
	if err == nil {
		t.Error("Expected error when Docker client returns error")
	}

	// Container should still be registered since removal failed
	if manager.GetContainerCount() != 1 {
		t.Errorf("Container should still be registered after failed removal, got count %d", manager.GetContainerCount())
	}
}

func TestConcurrentOperations(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(5*time.Minute, mock)

	done := make(chan bool)

	// Launch multiple goroutines doing concurrent operations
	for i := 0; i < 10; i++ {
		go func(id int) {
			containerID := "container-" + string(rune('0'+id))
			sessionID := "session-" + string(rune('0'+id))

			// Register
			manager.Register(containerID, sessionID, nil)

			// Heartbeat
			for j := 0; j < 10; j++ {
				manager.Heartbeat(containerID)
				time.Sleep(time.Millisecond)
			}

			// Get state
			manager.GetState(containerID)
			manager.GetIdleTime(containerID)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we got here without deadlock or race condition, test passed
	if manager.GetContainerCount() != 10 {
		t.Errorf("Expected 10 containers, got %d", manager.GetContainerCount())
	}
}

func TestMultipleHeartbeats(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(5*time.Minute, mock)

	containerID := "test-container-1"
	manager.Register(containerID, "session-123", nil)

	// Send multiple heartbeats
	for i := 0; i < 100; i++ {
		err := manager.Heartbeat(containerID)
		if err != nil {
			t.Errorf("Heartbeat %d failed: %v", i, err)
		}
	}

	// Verify state is still valid
	state, err := manager.GetState(containerID)
	if err != nil {
		t.Errorf("GetState failed after multiple heartbeats: %v", err)
	}

	if state.ContainerID != containerID {
		t.Error("Container ID should still be valid")
	}
}

func TestStop(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(5*time.Minute, mock)

	containerID := "test-container-1"
	manager.Register(containerID, "session-123", nil)

	// Start manager
	manager.Start()

	// Stop manager
	manager.Stop()

	// Wait a bit to ensure goroutine has stopped
	time.Sleep(100 * time.Millisecond)

	// Register a new container - should still work
	containerID2 := "container-2"
	manager.Register(containerID2, "session-456", nil)

	if manager.GetContainerCount() != 2 {
		t.Errorf("Expected 2 containers, got %d", manager.GetContainerCount())
	}
}

func TestContextCancellation(t *testing.T) {
	mock := &MockDockerClient{}
	manager := NewManager(50*time.Millisecond, mock)

	containerID := "test-container-1"
	manager.Register(containerID, "session-123", nil)

	// Start manager
	manager.Start()

	// Stop immediately
	manager.Stop()

	// Wait for cleanup loop to exit
	time.Sleep(200 * time.Millisecond)

	// Container should still be registered (no cleanup happened)
	if manager.GetContainerCount() != 1 {
		t.Errorf("Expected 1 container after stop, got %d", manager.GetContainerCount())
	}
}
