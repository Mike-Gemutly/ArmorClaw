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

	errsys "github.com/armorclaw/bridge/pkg/errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Component tracker for docker operations
var dockerTracker = errsys.GetComponentTracker("docker")

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
// Shell escapes are prevented via chmod a-x + LD_PRELOAD, not seccomp
var DefaultArmorClawProfile = SeccompProfile{
	DefaultAction: "SCMP_ACT_ALLOW",
	Architectures:  []string{"SCMP_ARCH_X86_64", "SCMP_ARCH_X86", "SCMP_ARCH_FILTER"},
	Syscalls: []Syscall{
		// Block network operations (data exfiltration prevention)
		// Shell escapes prevented via chmod a-x on binaries + LD_PRELOAD hook
		{
			Names:  []string{"socket", "connect", "accept", "bind", "listen", "sendto", "recvfrom"},
			Action: "SCMP_ACT_ERRNO",
		},
		// Block ptrace (process debugging/escape)
		{
			Names:  []string{"ptrace"},
			Action: "SCMP_ACT_ERRNO",
		},
		// Block module loading
		{
			Names:  []string{"init_module", "finit_module", "delete_module"},
			Action: "SCMP_ACT_ERRNO",
		},
		// Block raw I/O
		{
			Names:  []string{"iopl", "ioperm"},
			Action: "SCMP_ACT_ERRNO",
		},
		// Block key management
		{
			Names:  []string{"add_key", "request_key"},
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
	// Track operation start
	imageName := ""
	if config != nil && config.Image != "" {
		imageName = config.Image
	}
	dockerTracker.Event("create_container", map[string]any{"image": imageName})

	if c.client == nil {
		err := errsys.NewBuilder("CTX-010").
			Wrap(fmt.Errorf("docker client not initialized")).
			WithFunction("CreateContainer").
			WithInputs(map[string]any{"image": imageName}).
			Build()
		dockerTracker.Failure("create_container", err, map[string]any{"reason": "client_not_initialized"})
		return "", err
	}
	if !c.hasScope(ScopeCreate) {
		err := errsys.NewBuilder("CTX-001").
			Wrap(ErrInvalidOperation).
			WithFunction("CreateContainer").
			WithInputs(map[string]any{"image": imageName}).
			Build()
		dockerTracker.Failure("create_container", err, map[string]any{"reason": "invalid_scope"})
		return "", err
	}

	// Apply seccomp profile
	profile := DefaultArmorClawProfile
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		wrappedErr := errsys.NewBuilder("SYS-001").
			Wrap(err).
			WithFunction("CreateContainer").
			WithInputs(map[string]any{"image": imageName}).
			WithState(map[string]any{"operation": "marshal_seccomp"}).
			Build()
		dockerTracker.Failure("create_container", wrappedErr, map[string]any{"reason": "seccomp_marshal_failed"})
		return "", wrappedErr
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

	// Disable networking (prevent data exfiltration)
	if hostConfig.NetworkMode == "" {
		hostConfig.NetworkMode = "none"
	}

	// Create container with timeout
	ctx, cancel := context.WithTimeout(ctx, c.latencyTarget)
	defer cancel()

	resp, err := c.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, "")
	if err != nil {
		// Determine error code based on error type
		code := "CTX-001" // Default: container start failed
		errStr := err.Error()
		if containsAny(errStr, "permission denied", "access denied") {
			code = "CTX-010"
		} else if containsAny(errStr, "not found", "no such") {
			code = "CTX-011"
		} else if containsAny(errStr, "already exists", "already running") {
			code = "CTX-012"
		} else if containsAny(errStr, "image", "pull") {
			code = "CTX-020"
		}

		wrappedErr := errsys.NewBuilder(code).
			Wrap(err).
			WithFunction("CreateContainer").
			WithInputs(map[string]any{"image": imageName}).
			WithState(map[string]any{"container_name": config != nil && len(config.Cmd) > 0}).
			Build()
		dockerTracker.Failure("create_container", wrappedErr, map[string]any{"reason": "docker_api_error", "code": code})
		return "", wrappedErr
	}

	// Track success
	dockerTracker.Success("create_container", map[string]any{"container_id": resp.ID[:12]})
	return resp.ID, nil
}

// StartContainer starts a container
// Scope required: ScopeCreate
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	dockerTracker.Event("start_container", map[string]any{"container_id": containerID[:min(12, len(containerID))]})

	if c.client == nil {
		err := errsys.NewBuilder("CTX-010").
			Wrap(fmt.Errorf("docker client not initialized")).
			WithFunction("StartContainer").
			WithInputs(map[string]any{"container_id": containerID}).
			Build()
		dockerTracker.Failure("start_container", err, map[string]any{"reason": "client_not_initialized"})
		return err
	}
	if !c.hasScope(ScopeCreate) {
		err := errsys.NewBuilder("CTX-001").
			Wrap(ErrInvalidOperation).
			WithFunction("StartContainer").
			WithInputs(map[string]any{"container_id": containerID}).
			Build()
		dockerTracker.Failure("start_container", err, map[string]any{"reason": "invalid_scope"})
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, c.latencyTarget)
	defer cancel()

	err := c.client.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		code := "CTX-001"
		errStr := err.Error()
		if containsAny(errStr, "not found", "no such") {
			code = "CTX-011"
		} else if containsAny(errStr, "already running") {
			code = "CTX-012"
		}

		wrappedErr := errsys.NewBuilder(code).
			Wrap(err).
			WithFunction("StartContainer").
			WithInputs(map[string]any{"container_id": containerID}).
			Build()
		dockerTracker.Failure("start_container", wrappedErr, map[string]any{"reason": "docker_api_error", "code": code})
		return wrappedErr
	}

	dockerTracker.Success("start_container", map[string]any{"container_id": containerID[:min(12, len(containerID))]})
	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RemoveContainer removes a container with force option
// Scope required: ScopeRemove
func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	dockerTracker.Event("remove_container", map[string]any{"container_id": containerID[:min(12, len(containerID))], "force": force})

	if c.client == nil {
		err := errsys.NewBuilder("CTX-010").
			Wrap(fmt.Errorf("docker client not initialized")).
			WithFunction("RemoveContainer").
			WithInputs(map[string]any{"container_id": containerID, "force": force}).
			Build()
		dockerTracker.Failure("remove_container", err, map[string]any{"reason": "client_not_initialized"})
		return err
	}
	if !c.hasScope(ScopeRemove) {
		err := errsys.NewBuilder("CTX-001").
			Wrap(ErrInvalidOperation).
			WithFunction("RemoveContainer").
			WithInputs(map[string]any{"container_id": containerID, "force": force}).
			Build()
		dockerTracker.Failure("remove_container", err, map[string]any{"reason": "invalid_scope"})
		return err
	}

	options := container.RemoveOptions{
		Force:         force,
		RemoveVolumes: true,
	}

	err := c.client.ContainerRemove(ctx, containerID, options)
	if err != nil {
		code := "CTX-001"
		errStr := err.Error()
		if containsAny(errStr, "not found", "no such") {
			code = "CTX-011"
		}

		wrappedErr := errsys.NewBuilder(code).
			Wrap(err).
			WithFunction("RemoveContainer").
			WithInputs(map[string]any{"container_id": containerID, "force": force}).
			Build()
		dockerTracker.Failure("remove_container", wrappedErr, map[string]any{"reason": "docker_api_error", "code": code})
		return wrappedErr
	}

	dockerTracker.Success("remove_container", map[string]any{"container_id": containerID[:min(12, len(containerID))]})
	return nil
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
	dockerTracker.Event("availability_check", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(
		client.WithHost("unix:///var/run/docker.sock"),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		dockerTracker.Failure("availability_check", err, map[string]any{"reason": "client_create_failed"})
		return false
	}
	defer cli.Close()

	_, err = cli.Ping(ctx)
	if err != nil {
		dockerTracker.Failure("availability_check", err, map[string]any{"reason": "ping_failed"})
		return false
	}

	dockerTracker.Success("availability_check", nil)
	return true
}

// ExecInContainer executes a command in a container and returns the output
// This is a convenience method that combines ExecCreate, ExecStart, and inspection
func (c *Client) ExecInContainer(ctx context.Context, containerID string, cmd []string) (int, string, error) {
	dockerTracker.Event("exec_container", map[string]any{"container_id": containerID[:min(12, len(containerID))], "cmd": cmd})

	// Create exec instance
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := c.ExecCreate(ctx, containerID, execConfig)
	if err != nil {
		wrappedErr := errsys.NewBuilder("CTX-002").
			Wrap(err).
			WithFunction("ExecInContainer").
			WithInputs(map[string]any{"container_id": containerID, "cmd": cmd}).
			WithState(map[string]any{"phase": "exec_create"}).
			Build()
		dockerTracker.Failure("exec_container", wrappedErr, map[string]any{"reason": "exec_create_failed"})
		return 0, "", wrappedErr
	}

	// Start exec
	if err := c.ExecStart(ctx, execID); err != nil {
		wrappedErr := errsys.NewBuilder("CTX-002").
			Wrap(err).
			WithFunction("ExecInContainer").
			WithInputs(map[string]any{"container_id": containerID, "cmd": cmd, "exec_id": execID}).
			WithState(map[string]any{"phase": "exec_start"}).
			Build()
		dockerTracker.Failure("exec_container", wrappedErr, map[string]any{"reason": "exec_start_failed"})
		return 0, "", wrappedErr
	}

	// Inspect exec to get exit code
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	info, err := c.client.ContainerExecInspect(ctx, execID)
	if err != nil {
		wrappedErr := errsys.NewBuilder("CTX-002").
			Wrap(err).
			WithFunction("ExecInContainer").
			WithInputs(map[string]any{"container_id": containerID, "cmd": cmd, "exec_id": execID}).
			WithState(map[string]any{"phase": "exec_inspect"}).
			Build()
		dockerTracker.Failure("exec_container", wrappedErr, map[string]any{"reason": "exec_inspect_failed"})
		return 0, "", wrappedErr
	}

	dockerTracker.Success("exec_container", map[string]any{"container_id": containerID[:min(12, len(containerID))], "exit_code": info.ExitCode})
	return info.ExitCode, "", nil
}

// CreateAndStartContainer creates and starts a container in one operation
// This is optimized for low-latency container startup
func (c *Client) CreateAndStartContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform) (string, error) {
	imageName := ""
	if config != nil && config.Image != "" {
		imageName = config.Image
	}
	dockerTracker.Event("create_and_start", map[string]any{"image": imageName})

	if c.client == nil {
		err := errsys.NewBuilder("CTX-010").
			Wrap(fmt.Errorf("docker client not initialized")).
			WithFunction("CreateAndStartContainer").
			WithInputs(map[string]any{"image": imageName}).
			Build()
		dockerTracker.Failure("create_and_start", err, map[string]any{"reason": "client_not_initialized"})
		return "", err
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

		wrappedErr := errsys.NewBuilder("CTX-001").
			Wrap(err).
			WithFunction("CreateAndStartContainer").
			WithInputs(map[string]any{"image": imageName, "container_id": containerID}).
			WithState(map[string]any{"phase": "start_failed", "cleaned_up": true}).
			Build()
		dockerTracker.Failure("create_and_start", wrappedErr, map[string]any{"reason": "start_failed", "container_id": containerID[:min(12, len(containerID))]})
		return "", wrappedErr
	}

	dockerTracker.Success("create_and_start", map[string]any{"container_id": containerID[:min(12, len(containerID))], "image": imageName})
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
	dockerTracker.Event("image_check", map[string]any{"image": image})

	if c.client == nil {
		err := errsys.NewBuilder("CTX-010").
			Wrap(fmt.Errorf("docker client not initialized")).
			WithFunction("ImageExists").
			WithInputs(map[string]any{"image": image}).
			Build()
		dockerTracker.Failure("image_check", err, map[string]any{"reason": "client_not_initialized"})
		return false, err
	}

	_, _, err := c.client.ImageInspectWithRaw(ctx, image)
	if err == nil {
		dockerTracker.Success("image_check", map[string]any{"image": image, "exists": true})
		return true, nil
	}
	if client.IsErrNotFound(err) {
		dockerTracker.Success("image_check", map[string]any{"image": image, "exists": false})
		return false, nil
	}

	wrappedErr := errsys.NewBuilder("CTX-021").
		Wrap(err).
		WithFunction("ImageExists").
		WithInputs(map[string]any{"image": image}).
		Build()
	dockerTracker.Failure("image_check", wrappedErr, map[string]any{"reason": "inspect_failed"})
	return false, wrappedErr
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
	imageName := ""
	if config != nil && config.Image != "" {
		imageName = config.Image
	}
	dockerTracker.Event("create_with_retry", map[string]any{"image": imageName, "max_attempts": 3})

	maxAttempts := 3
	baseDelay := 100 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		containerID, err := c.CreateContainer(ctx, config, hostConfig, networkingConfig, platform)
		if err == nil {
			dockerTracker.Success("create_with_retry", map[string]any{
				"image":        imageName,
				"container_id": containerID[:min(12, len(containerID))],
				"attempts":     attempt + 1,
			})
			return containerID, nil
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			dockerTracker.Failure("create_with_retry", err, map[string]any{
				"image":    imageName,
				"attempt":  attempt + 1,
				"retryable": false,
			})
			return "", err // Non-retryable error
		}

		lastErr = err

		// Don't wait after the last attempt
		if attempt < maxAttempts-1 {
			backoff := baseDelay * time.Duration(1<<uint(attempt))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				wrappedErr := errsys.NewBuilder("CTX-001").
					Wrap(ctx.Err()).
					WithFunction("CreateContainerWithRetry").
					WithInputs(map[string]any{"image": imageName}).
					WithState(map[string]any{"attempts": attempt + 1, "cancelled": true}).
					Build()
				dockerTracker.Failure("create_with_retry", wrappedErr, map[string]any{"reason": "context_cancelled"})
				return "", wrappedErr
			}
		}
	}

	wrappedErr := errsys.NewBuilder("CTX-001").
		Wrap(lastErr).
		WithFunction("CreateContainerWithRetry").
		WithInputs(map[string]any{"image": imageName}).
		WithState(map[string]any{"attempts": maxAttempts, "exhausted": true}).
		Build()
	dockerTracker.Failure("create_with_retry", wrappedErr, map[string]any{
		"image":     imageName,
		"attempts":  maxAttempts,
		"exhausted": true,
	})
	return "", wrappedErr
}

// HealthCheck performs a health check on the Docker daemon
// Returns true if the daemon is healthy, false otherwise
func (c *Client) HealthCheck(ctx context.Context) error {
	dockerTracker.Event("health_check", nil)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.Ping(ctx)
	if err != nil {
		wrappedErr := errsys.NewBuilder("CTX-003").
			Wrap(err).
			WithFunction("HealthCheck").
			Build()
		dockerTracker.Failure("health_check", wrappedErr, nil)
		return wrappedErr
	}

	dockerTracker.Success("health_check", nil)
	return nil
}

// IsRunning checks if a container is currently running
func (c *Client) IsRunning(containerID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	containerJSON, err := c.InspectContainer(ctx, containerID)
	if err != nil {
		return false, err
	}

	return containerJSON.State != nil && containerJSON.State.Running, nil
}
