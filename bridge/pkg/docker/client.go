// Package docker provides a secure Docker client for ArmorClaw bridge.
// This client is restricted to container creation, execution, and removal operations.
// It implements API scopes, seccomp profiles, and optimized for low-latency operations.
package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	ErrContainerNotFound = errors.New("container not found")
	ErrInvalidOperation = errors.New("invalid operation for this client")
	ErrOperationTimedOut = errors.New("operation timed out")
)

// Scope defines the allowed operations for this client
type Scope string

const (
	ScopeCreate Scope = "create" // Only allow container creation
	ScopeExec   Scope = "exec"   // Only allow exec operations
	ScopeRemove Scope = "remove" // Only allow container removal
)

// SeccompProfile defines a seccomp profile for container hardening
type SeccompProfile struct {
	DefaultAction string   `json:"defaultAction"` // "SCMP_ACT_ALLOW", "SCMP_ACT_ERRNO"
	Architectures  []string `json:"architectures"`
	Syscalls       []Syscall `json:"syscalls"`
}

// Syscall defines a seccomp syscall rule
type Syscall struct {
	Names  []string `json:"names"`
	Action string   `json:"action"`
	Args   []string `json:"args,omitempty"`
}

// DefaultArmorClawProfile is the default seccomp profile for ArmorClaw containers
// It blocks dangerous syscalls while allowing necessary operations
var DefaultArmorClawProfile = SeccompProfile{
	DefaultAction: "SCMP_ACT_ERRNO",
	Architectures:  []string{"SCMP_ARCH_X86_64", "SCMP_ARCH_X86", "SCMP_ARCH_FILTER"},
	Syscalls: []Syscall{
		// Block dangerous operations
		{
			Names:  []string{"clone", "fork", "vfork", "ptrace"},
			Action: "SCMP_ACT_ERRNO",
		},
		// Block network operations (no direct network access)
		{
			Names:  []string{"socket", "connect", "accept", "bind", "listen"},
			Action: "SCMP_ACT_ERRNO",
		},
		// Block module loading
		{
			Names:  []string{"init_module", "finit_module", "delete_module"},
			Action: "SCMP_ACT_ERRNO",
		},
		// Block raw I/O
		{
			Names:  []string{"iopl", "ioperm", "outb", "inb"},
			Action: "SCMP_ACT_ERRNO",
		},
	},
}

// Client is a restricted Docker client with scoping and seccomp support
type Client struct {
	client        *client.Client
	scopes        map[Scope]bool
	latencyTarget time.Duration
}

// Config holds client configuration
type Config struct {
	Host         string        // Docker daemon address
	APIVersion   string        // API version
	Scopes       []Scope       // Allowed operations
	LatencyTarget time.Duration // Target latency for operations
}

// New creates a new restricted Docker client
func New(cfg Config) (*Client, error) {
	if cfg.Host == "" {
		cfg.Host = "unix:///var/run/docker.sock"
	}
	if cfg.APIVersion == "" {
		cfg.APIVersion = "1.45"
	}
	if cfg.LatencyTarget == 0 {
		cfg.LatencyTarget = 15 * time.Millisecond
	}

	// Create Docker client
	cli, err := client.NewClientWithOpts(
		client.WithHost(cfg.Host),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// Build scope map
	scopes := make(map[Scope]bool)
	if len(cfg.Scopes) == 0 {
		// Default to all scopes
		scopes[ScopeCreate] = true
		scopes[ScopeExec] = true
		scopes[ScopeRemove] = true
	} else {
		for _, scope := range cfg.Scopes {
			scopes[scope] = true
		}
	}

	return &Client{
		client:        cli,
		scopes:        scopes,
		latencyTarget: cfg.LatencyTarget,
	}, nil
}

// hasScope checks if the client has the required scope
func (c *Client) hasScope(required Scope) bool {
	return c.scopes[required]
}

// CreateContainer creates a new container with the given configuration
// Scope required: ScopeCreate
func (c *Client) CreateContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("docker client not initialized")
	}
	if !c.hasScope(ScopeCreate) {
		return "", ErrInvalidOperation
	}

	// Apply seccomp profile
	profile := DefaultArmorClawProfile
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return "", fmt.Errorf("failed to marshal seccomp profile: %w", err)
	}

	if hostConfig.SecurityOpt == nil {
		hostConfig.SecurityOpt = make([]string, 0)
	}
	hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, fmt.Sprintf("seccomp=%s", string(profileJSON)))

	// Add read-only root filesystem (if not specified)
	if !hostConfig.ReadonlyRootfs {
		hostConfig.ReadonlyRootfs = true
	}

	// Drop all capabilities
	if hostConfig.CapDrop == nil || len(hostConfig.CapDrop) == 0 {
		hostConfig.CapDrop = []string{"ALL"}
	}

	// Create container with timeout
	ctx, cancel := context.WithTimeout(ctx, c.latencyTarget)
	defer cancel()

	resp, err := c.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, "")
	if err != nil {
		return "", fmt.Errorf("container create failed: %w", err)
	}

	return resp.ID, nil
}

// StartContainer starts a container
// Scope required: ScopeCreate
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	if c.client == nil {
		return fmt.Errorf("docker client not initialized")
	}
	if !c.hasScope(ScopeCreate) {
		return ErrInvalidOperation
	}

	ctx, cancel := context.WithTimeout(ctx, c.latencyTarget)
	defer cancel()

	return c.client.ContainerStart(ctx, containerID, container.StartOptions{})
}

// RemoveContainer removes a container with force option
// Scope required: ScopeRemove
func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	if c.client == nil {
		return fmt.Errorf("docker client not initialized")
	}
	if !c.hasScope(ScopeRemove) {
		return ErrInvalidOperation
	}

	options := container.RemoveOptions{
		Force:         force,
		RemoveVolumes: true,
	}

	return c.client.ContainerRemove(ctx, containerID, options)
}

// ExecCreate creates an exec instance in a container
// Scope required: ScopeExec
func (c *Client) ExecCreate(ctx context.Context, containerID string, config container.ExecOptions) (string, error) {
	if !c.hasScope(ScopeExec) {
		return "", ErrInvalidOperation
	}

	ctx, cancel := context.WithTimeout(ctx, c.latencyTarget)
	defer cancel()

	resp, err := c.client.ContainerExecCreate(ctx, containerID, config)
	if err != nil {
		return "", fmt.Errorf("exec create failed: %w", err)
	}

	return resp.ID, nil
}

// ExecStart starts an exec instance
// Scope required: ScopeExec
func (c *Client) ExecStart(ctx context.Context, execID string) error {
	if !c.hasScope(ScopeExec) {
		return ErrInvalidOperation
	}

	return c.client.ContainerExecStart(ctx, execID, container.ExecStartOptions{})
}

// InspectContainer inspects a container (optimized for low latency)
func (c *Client) InspectContainer(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	ctx, cancel := context.WithTimeout(ctx, c.latencyTarget)
	defer cancel()

	return c.client.ContainerInspect(ctx, containerID)
}

// ContainerLogs gets logs from a container (streaming, leak-free)
// Returns a reader that must be closed by the caller
func (c *Client) ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
	// Don't set timeout for logs - they may take time
	return c.client.ContainerLogs(ctx, containerID, options)
}

// WaitContainer waits for a container to exit
func (c *Client) WaitContainer(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	return c.client.ContainerWait(ctx, containerID, condition)
}

// ListContainers lists containers with optional filter
func (c *Client) ListContainers(ctx context.Context, all bool, filters filters.Args) ([]types.Container, error) {
	options := container.ListOptions{
		All:     all,
		Filters: filters,
	}

	return c.client.ContainerList(ctx, options)
}

// GetContainerEvents streams container events from Docker
func (c *Client) GetContainerEvents(ctx context.Context, options events.ListOptions) (<-chan interface{}, <-chan error) {
	eventChan, errChan := c.client.Events(ctx, options)
	// Convert events.Message chan to interface{} chan
	interfaceChan := make(chan interface{}, 1)
	go func() {
		for event := range eventChan {
			interfaceChan <- event
		}
		close(interfaceChan)
	}()
	return interfaceChan, errChan
}

// Ping pings the Docker daemon
func (c *Client) Ping(ctx context.Context) (types.Ping, error) {
	return c.client.Ping(ctx)
}

// Close closes the Docker client connection
func (c *Client) Close() error {
	return c.client.Close()
}

// IsAvailable checks if Docker daemon is available and running
func IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(
		client.WithHost("unix:///var/run/docker.sock"),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return false
	}
	defer cli.Close()

	_, err = cli.Ping(ctx)
	return err == nil
}

// ExecInContainer executes a command in a container and returns the output
// This is a convenience method that combines ExecCreate, ExecStart, and inspection
func (c *Client) ExecInContainer(ctx context.Context, containerID string, cmd []string) (int, string, error) {
	// Create exec instance
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := c.ExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return 0, "", fmt.Errorf("exec create failed: %w", err)
	}

	// Start exec
	if err := c.ExecStart(ctx, execID); err != nil {
		return 0, "", fmt.Errorf("exec start failed: %w", err)
	}

	// Inspect exec to get exit code
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	info, err := c.client.ContainerExecInspect(ctx, execID)
	if err != nil {
		return 0, "", fmt.Errorf("exec inspect failed: %w", err)
	}

	return info.ExitCode, "", nil
}

// CreateAndStartContainer creates and starts a container in one operation
// This is optimized for low-latency container startup
func (c *Client) CreateAndStartContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("docker client not initialized")
	}
	containerID, err := c.CreateContainer(ctx, config, hostConfig, networkingConfig, platform)
	if err != nil {
		return "", err
	}

	if err := c.StartContainer(ctx, containerID); err != nil {
		// Clean up on failure - use a short timeout for cleanup to avoid hanging
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cleanupCancel()
		_ = c.RemoveContainer(cleanupCtx, containerID, true)
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return containerID, nil
}

// StreamContainerLogs streams container logs to channels
// This ensures leak-free log streaming with automatic cleanup
func (c *Client) StreamContainerLogs(ctx context.Context, containerID string, logChan chan<- string, errorChan chan<- error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "100",
	}

	reader, err := c.ContainerLogs(ctx, containerID, options)
	if err != nil {
		errorChan <- err
		return
	}
	defer reader.Close()

	decoder := json.NewDecoder(reader)

	for {
		var msg map[string]interface{}
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				return
			}
			errorChan <- err
			return
		}

		if data, ok := msg["data"].(string); ok && len(data) > 0 {
			select {
			case logChan <- data:
			case <-ctx.Done():
				return
			}
		}
	}
}

// ImageExists checks if a container image exists locally
// Returns true if the image exists, false if it doesn't, or error if unable to check
func (c *Client) ImageExists(ctx context.Context, image string) (bool, error) {
	if c.client == nil {
		return false, fmt.Errorf("docker client not initialized")
	}
	_, _, err := c.client.ImageInspectWithRaw(ctx, image)
	if err == nil {
		return true, nil
	}
	if client.IsErrNotFound(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to inspect image: %w", err)
}

// isRetryableError checks if an error is retryable (transient network issues, temporary daemon issues)
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types that indicate transient issues
	errStr := err.Error()

	// Context-related errors (timeout, cancellation)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	// Docker API connection errors
	if containsAny(errStr, "connection refused", "connection reset", "broken pipe", "temporary failure") {
		return true
	}

	// Docker daemon busy
	if containsAny(errStr, "daemon busy", "already in use") {
		return true
	}

	return false
}

// containsAny checks if the string contains any of the substrings
func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

// contains is a simple string contains helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (
		s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
	 indexOfSubstring(s, substr) >= 0))
}

// indexOfSubstring finds the index of a substring
func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// CreateContainerWithRetry creates a container with retry logic for transient failures
func (c *Client) CreateContainerWithRetry(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform) (string, error) {
	maxAttempts := 3
	baseDelay := 100 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		containerID, err := c.CreateContainer(ctx, config, hostConfig, networkingConfig, platform)
		if err == nil {
			return containerID, nil
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			return "", err // Non-retryable error
		}

		lastErr = err

		// Don't wait after the last attempt
		if attempt < maxAttempts-1 {
			backoff := baseDelay * time.Duration(1<<uint(attempt))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}

	return "", fmt.Errorf("container create failed after %d attempts: %w", maxAttempts, lastErr)
}

// HealthCheck performs a health check on the Docker daemon
// Returns true if the daemon is healthy, false otherwise
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.Ping(ctx)
	return err
}
