package studio

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

//=============================================================================
// Agent Factory - Container Spawning
//=============================================================================

// DockerClient interface for testability
type DockerClient interface {
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig any, platform any, name string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error)
}

// KeystoreProvider interface for accessing secrets
type KeystoreProvider interface {
	IsUnsealed() bool
	Get(key string) (string, error)
}

// AgentFactory spawns containers from agent definitions
type AgentFactory struct {
	docker   DockerClient
	store    Store
	keystore KeystoreProvider
}

// FactoryConfig configures the agent factory
type FactoryConfig struct {
	DockerClient DockerClient
	Store        Store
	Keystore     KeystoreProvider
	DefaultImage string
}

// NewAgentFactory creates a new agent factory
func NewAgentFactory(cfg FactoryConfig) *AgentFactory {
	return &AgentFactory{
		docker:   cfg.DockerClient,
		store:    cfg.Store,
		keystore: cfg.Keystore,
	}
}

// SpawnRequest contains parameters for spawning an agent
type SpawnRequest struct {
	DefinitionID    string          `json:"definition_id"`
	TaskDescription string          `json:"task_description"`
	UserID          string          `json:"user_id"`
	RoomID          string          `json:"room_id,omitempty"`
	Config          json.RawMessage `json:"config,omitempty"`
}

// SpawnResult contains the result of spawning an agent
type SpawnResult struct {
	Instance   *AgentInstance   `json:"instance"`
	Definition *AgentDefinition `json:"definition"`
	Warnings   []string         `json:"warnings,omitempty"`
}

// Spawn creates a container from an agent definition
func (f *AgentFactory) Spawn(ctx context.Context, req *SpawnRequest) (*SpawnResult, error) {
	// 1. Get the agent definition
	def, err := f.store.GetDefinition(req.DefinitionID)
	if err != nil {
		return nil, fmt.Errorf("agent definition not found: %s", req.DefinitionID)
	}

	if !def.IsActive {
		return nil, fmt.Errorf("agent definition is inactive: %s", req.DefinitionID)
	}

	// 2. Get resource profile
	profile := GetProfile(def.ResourceTier)

	// 3. Build environment variables
	env, warnings := f.buildEnvironment(def, req.TaskDescription)

	if len(req.Config) > 0 {
		env = append(env, "STEP_CONFIG="+string(req.Config))
	}

	// 4. Create container config
	config := &container.Config{
		Image: "armorclaw/agent-base:latest",
		Env:   env,
		Labels: map[string]string{
			"armorclaw.agent_id":   def.ID,
			"armorclaw.agent_name": def.Name,
			"armorclaw.tier":       def.ResourceTier,
			"armorclaw.created_by": req.UserID,
			"armorclaw.task":       truncateLabel(req.TaskDescription, 63),
		},
		User:       "10001:10001", // Non-root user
		StopSignal: "SIGTERM",
	}

	// 5. Create host config with security hardening
	stateDir := fmt.Sprintf("/var/lib/armorclaw/agent-state/%s", def.ID)
	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:     int64(profile.MemoryMB) * 1024 * 1024,
			MemorySwap: int64(profile.MemoryMB) * 1024 * 1024, // Disable swap
			CPUShares:  int64(profile.CPUShares),
		},
		AutoRemove:     false,  // We manage removal explicitly
		NetworkMode:    "none", // Isolated by default
		ReadonlyRootfs: true,
		Binds:          []string{fmt.Sprintf("%s:/home/claw/.openclaw", stateDir)},
		SecurityOpt:    []string{"no-new-privileges:true"},
		CapDrop:        []string{"ALL"},
		Privileged:     false,
	}

	// 5b. Ensure host state directory exists for persistent agent sessions
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create agent state directory: %w", err)
	}

	// 6. Generate instance ID
	instanceID := generateID("instance")

	// 7. Create container
	createResp, err := f.docker.ContainerCreate(
		ctx,
		config,
		hostConfig,
		nil, // networking config
		nil, // platform
		"armorclaw-"+instanceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// 8. Start container
	if err := f.docker.ContainerStart(ctx, createResp.ID, container.StartOptions{}); err != nil {
		// Clean up on start failure
		_ = f.docker.ContainerRemove(ctx, createResp.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// 9. Track instance in database
	now := time.Now()
	instance := &AgentInstance{
		ID:              instanceID,
		DefinitionID:    def.ID,
		ContainerID:     createResp.ID,
		Status:          StatusRunning,
		TaskDescription: req.TaskDescription,
		SpawnedBy:       req.UserID,
		StartedAt:       &now,
		RoomID:          req.RoomID,
	}

	if err := f.store.CreateInstance(instance); err != nil {
		// Log but don't fail - container is running
		warnings = append(warnings, fmt.Sprintf("failed to track instance: %v", err))
	}

	return &SpawnResult{
		Instance:   instance,
		Definition: def,
		Warnings:   warnings,
	}, nil
}

// buildEnvironment creates environment variables for the container
func (f *AgentFactory) buildEnvironment(def *AgentDefinition, task string) ([]string, []string) {
	var env []string
	var warnings []string

	// Core agent configuration
	env = append(env,
		fmt.Sprintf("AGENT_ID=%s", def.ID),
		fmt.Sprintf("AGENT_NAME=%s", def.Name),
		fmt.Sprintf("ENABLED_SKILLS=%s", strings.Join(def.Skills, ",")),
		fmt.Sprintf("RESOURCE_TIER=%s", def.ResourceTier),
		fmt.Sprintf("TASK_DESCRIPTION=%s", task),
	)

	// PII access - inject values if keystore is available
	if f.keystore != nil && f.keystore.IsUnsealed() {
		for _, piiID := range def.PIIAccess {
			value, err := f.keystore.Get(piiID)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("PII field '%s' not found in keystore", piiID))
				continue
			}
			env = append(env, fmt.Sprintf("PII_%s=%s", strings.ToUpper(piiID), value))
		}
	}

	return env, warnings
}

// Stop stops a running agent instance
func (f *AgentFactory) Stop(ctx context.Context, instanceID string, timeout time.Duration) error {
	// 1. Get instance from store
	instance, err := f.store.GetInstance(instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	if instance.Status != StatusRunning {
		return fmt.Errorf("instance is not running: %s (status: %s)", instanceID, instance.Status)
	}

	// 2. Stop container
	stopTimeout := int(timeout.Seconds())
	if stopTimeout == 0 {
		stopTimeout = 30 // Default 30 seconds
	}

	if err := f.docker.ContainerStop(ctx, instance.ContainerID, container.StopOptions{Timeout: &stopTimeout}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// 3. Update instance status
	now := time.Now()
	instance.Status = StatusCompleted
	instance.CompletedAt = &now

	if err := f.store.UpdateInstance(instance); err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	return nil
}

// Remove removes a stopped instance and its container
func (f *AgentFactory) Remove(ctx context.Context, instanceID string) error {
	// 1. Get instance from store
	instance, err := f.store.GetInstance(instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	// 2. Remove container (force if still running)
	if err := f.docker.ContainerRemove(ctx, instance.ContainerID, container.RemoveOptions{
		Force:         instance.Status == StatusRunning,
		RemoveVolumes: true,
	}); err != nil {
		// Log but continue - container may already be gone
	}

	// 3. Delete instance from store
	if err := f.store.DeleteInstance(instanceID); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	return nil
}

// GetStatus returns the current status of an instance
func (f *AgentFactory) GetStatus(ctx context.Context, instanceID string) (*AgentInstance, error) {
	instance, err := f.store.GetInstance(instanceID)
	if err != nil {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}

	// If we think it's running, check actual container status
	if instance.Status == StatusRunning {
		inspect, err := f.docker.ContainerInspect(ctx, instance.ContainerID)
		if err != nil {
			// Container is gone, mark as failed
			instance.Status = StatusFailed
			now := time.Now()
			instance.CompletedAt = &now
			_ = f.store.UpdateInstance(instance)
		} else if !inspect.State.Running {
			// Container stopped
			if inspect.State.ExitCode == 0 {
				instance.Status = StatusCompleted
			} else {
				instance.Status = StatusFailed
			}
			now := time.Now()
			instance.CompletedAt = &now
			_ = f.store.UpdateInstance(instance)
		}
	}

	return instance, nil
}

// ListInstances lists all instances, optionally filtered by definition ID
func (f *AgentFactory) ListInstances(definitionID string) ([]*AgentInstance, error) {
	return f.store.ListInstances(definitionID, "") // Empty status means all statuses
}

// GetRunningInstance returns the single running instance for a definition, or nil if none
func (f *AgentFactory) GetRunningInstance(definitionID string) (*AgentInstance, error) {
	instances, err := f.store.ListInstances(definitionID, StatusRunning)
	if err != nil {
		return nil, fmt.Errorf("failed to list running instances: %w", err)
	}
	if len(instances) == 0 {
		return nil, nil
	}
	return instances[0], nil
}

// CleanupStale removes instances whose containers are no longer running
func (f *AgentFactory) CleanupStale(ctx context.Context) ([]string, error) {
	var cleaned []string

	instances, err := f.store.ListInstances("", StatusRunning)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	for _, instance := range instances {
		if instance.Status != StatusRunning {
			continue
		}

		inspect, err := f.docker.ContainerInspect(ctx, instance.ContainerID)
		if err != nil {
			// Container is gone
			instance.Status = StatusFailed
			now := time.Now()
			instance.CompletedAt = &now
			if err := f.store.UpdateInstance(instance); err == nil {
				cleaned = append(cleaned, instance.ID)
			}
			continue
		}

		if !inspect.State.Running {
			// Container stopped
			if inspect.State.ExitCode == 0 {
				instance.Status = StatusCompleted
			} else {
				instance.Status = StatusFailed
			}
			now := time.Now()
			instance.CompletedAt = &now
			if err := f.store.UpdateInstance(instance); err == nil {
				cleaned = append(cleaned, instance.ID)
			}
		}
	}

	return cleaned, nil
}

//=============================================================================
// Helper Functions
//=============================================================================

// truncateLabel truncates a string to maxLen for use as a container label
func truncateLabel(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
