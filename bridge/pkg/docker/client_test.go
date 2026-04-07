package docker

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"
)

// TestTerminateContainer_Success tests successful container termination
func TestTerminateContainer_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode - requires Docker daemon")
	}

	ctx := context.Background()

	client, err := New(Config{
		Host:   "unix:///var/run/docker.sock",
		Scopes: []Scope{ScopeRemove, ScopeCreate},
	})
	if err != nil {
		t.Fatalf("Failed to create docker client: %v", err)
	}
	defer client.Close()

	config := &container.Config{
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "10"},
		Labels: map[string]string{
			"armorclaw.test": "true",
		},
	}
	hostConfig := &container.HostConfig{
		AutoRemove: false,
	}

	containerID, err := client.CreateContainer(ctx, config, hostConfig, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create test container: %v", err)
	}
	t.Logf("Created test container: %s", containerID)

	if err := client.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("Failed to start test container: %v", err)
	}

	if err := client.TerminateContainer(ctx, containerID); err != nil {
		t.Errorf("TerminateContainer failed: %v", err)
	}

	_ = client.RemoveContainer(ctx, containerID, true)
}

// TestTerminateContainer_UnauthorizedScope tests that termination fails without proper scope
func TestTerminateContainer_UnauthorizedScope(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode - requires Docker daemon")
	}

	ctx := context.Background()

	client, err := New(Config{
		Host:   "unix:///var/run/docker.sock",
		Scopes: []Scope{ScopeCreate},
	})
	if err != nil {
		t.Fatalf("Failed to create docker client: %v", err)
	}
	defer client.Close()

	err = client.TerminateContainer(ctx, "test-container-id")
	if err == nil {
		t.Error("Expected error when terminating without ScopeRemove, got nil")
	}
}

// TestTerminateContainer_NonExistent tests termination of non-existent container
func TestTerminateContainer_NonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode - requires Docker daemon")
	}

	ctx := context.Background()

	client, err := New(Config{
		Host:   "unix:///var/run/docker.sock",
		Scopes: []Scope{ScopeRemove, ScopeCreate},
	})
	if err != nil {
		t.Fatalf("Failed to create docker client: %v", err)
	}
	defer client.Close()

	err = client.TerminateContainer(ctx, "nonexistent-container-id")
	if err == nil {
		t.Error("Expected error when terminating non-existent container, got nil")
	}
}
