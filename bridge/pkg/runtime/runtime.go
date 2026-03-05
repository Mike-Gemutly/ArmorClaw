// Package runtime provides a container runtime abstraction layer for ArmorClaw.
// This allows swapping between Docker, containerd, and Firecracker without
// modifying the bridge code.
//
// Architecture:
//
//	Bridge → Runtime Interface → Docker/containerd/Firecracker Adapter
//
// Security: Runtime abstraction isolates the bridge from container runtime details,
// allowing secure defaults to be enforced at the adapter level.
package runtime

import (
	"context"
	"time"
)

// Status represents container runtime status.
type Status string

const (
	StatusCreated Status = "created"
	StatusRunning Status = "running"
	StatusPaused  Status = "paused"
	StatusStopped Status = "stopped"
	StatusExited  Status = "exited"
	StatusDead    Status = "dead"
	StatusUnknown Status = "unknown"
)

// RuntimeType specifies which container runtime to use.
type RuntimeType string

const (
	RuntimeDocker     RuntimeType = "docker"
	RuntimeContainerd RuntimeType = "containerd"
	RuntimeFirecracker RuntimeType = "firecracker"
)

// ContainerSpec defines the configuration for creating a container.
// It includes security hardening options that map to container runtime settings.
type ContainerSpec struct {
	// Required
	Image string `json:"image"` // Container image to run

	// Identification
	Name    string            `json:"name,omitempty"`    // Container name
	Labels  map[string]string `json:"labels,omitempty"`  // Container labels
	EnvVars map[string]string `json:"env_vars,omitempty"` // Environment variables

	// Resource Limits
	CPUQuota   int64 `json:"cpu_quota,omitempty"`   // CPU quota in microseconds (e.g., 100000 = 1 CPU)
	Memory     int64 `json:"memory,omitempty"`      // Memory limit in bytes (e.g., 512*1024*1024 = 512MB)
	PidsLimit  int64 `json:"pids_limit,omitempty"`  // Maximum number of processes (e.g., 100)

	// Security Hardening
	SeccompProfile string   `json:"seccomp_profile,omitempty"` // Seccomp profile name or JSON
	AppArmorProfile string  `json:"apparmor_profile,omitempty"` // AppArmor profile name
	ReadOnlyRootFS  bool    `json:"read_only_root_fs,omitempty"` // Mount root filesystem as read-only
	CapDrop        []string `json:"cap_drop,omitempty"`        // Linux capabilities to drop
	CapAdd         []string `json:"cap_add,omitempty"`         // Linux capabilities to add (use sparingly)
	NoNewPrivileges bool    `json:"no_new_privileges,omitempty"` // Prevent privilege escalation

	// Networking
	NetworkDisabled bool     `json:"network_disabled,omitempty"` // Disable network access
	DNS             []string `json:"dns,omitempty"`              // Custom DNS servers
	DNSSearch       []string `json:"dns_search,omitempty"`       // DNS search domains

	// Storage
	Volumes    []VolumeSpec `json:"volumes,omitempty"`    // Volume mounts
	TmpFS      []TmpFSSpec  `json:"tmpfs,omitempty"`      // Tmpfs mounts
	WorkingDir string       `json:"working_dir,omitempty"` // Working directory

	// Execution
	Cmd        []string `json:"cmd,omitempty"`        // Command to run (overrides ENTRYPOINT)
	Entrypoint []string `json:"entrypoint,omitempty"` // Override ENTRYPOINT
	User       string   `json:"user,omitempty"`       // User to run as (e.g., "10001:10001")

	// Lifecycle
	AutoRemove   bool          `json:"auto_remove,omitempty"`   // Remove container when it exits
	StopTimeout  time.Duration `json:"stop_timeout,omitempty"`  // Timeout for graceful stop
	RestartPolicy string       `json:"restart_policy,omitempty"` // "no", "always", "on-failure"
}

// VolumeSpec defines a volume mount.
type VolumeSpec struct {
	Source   string `json:"source,omitempty"`   // Host path or volume name
	Target   string `json:"target"`             // Container path
	ReadOnly bool   `json:"read_only,omitempty"` // Mount read-only
	Type     string `json:"type,omitempty"`     // "bind", "volume", "tmpfs"
}

// TmpFSSpec defines a tmpfs mount.
type TmpFSSpec struct {
	Target string `json:"target"`             // Container path
	Size   int64  `json:"size,omitempty"`     // Size in bytes
}

// ContainerInfo contains runtime information about a container.
type ContainerInfo struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Status    Status            `json:"status"`
	Created   time.Time         `json:"created"`
	Labels    map[string]string `json:"labels,omitempty"`
	IPAddress string            `json:"ip_address,omitempty"`
	Ports     []PortMapping     `json:"ports,omitempty"`
}

// PortMapping represents a port mapping.
type PortMapping struct {
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"` // "tcp" or "udp"
}

// ExecConfig defines options for executing commands in a container.
type ExecConfig struct {
	Cmd          []string `json:"cmd"`
	Env          []string `json:"env,omitempty"`
	User         string   `json:"user,omitempty"`
	WorkingDir   string   `json:"working_dir,omitempty"`
	Detach       bool     `json:"detach,omitempty"`
	Tty          bool     `json:"tty,omitempty"`
	AttachStdout bool     `json:"attach_stdout,omitempty"`
	AttachStderr bool     `json:"attach_stderr,omitempty"`
}

// ExecResult contains the result of an exec operation.
type ExecResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   []byte `json:"stdout,omitempty"`
	Stderr   []byte `json:"stderr,omitempty"`
}

// Stats contains container resource usage statistics.
type Stats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   int64   `json:"memory_usage"`
	MemoryLimit   int64   `json:"memory_limit"`
	MemoryPercent float64 `json:"memory_percent"`
	NetworkRx     int64   `json:"network_rx"`
	NetworkTx     int64   `json:"network_tx"`
	BlockRead     int64   `json:"block_read"`
	BlockWrite    int64   `json:"block_write"`
	Pids          int64   `json:"pids"`
}

// Runtime is the interface for container runtime operations.
// Implementations must handle security hardening and resource enforcement.
type Runtime interface {
	// Lifecycle
	CreateContainer(ctx context.Context, spec *ContainerSpec) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string, timeout *time.Duration) error
	RemoveContainer(ctx context.Context, id string, force bool) error

	// Execution
	ExecContainer(ctx context.Context, id string, config *ExecConfig) (*ExecResult, error)
	ExecContainerStream(ctx context.Context, id string, config *ExecConfig) (stdout, stderr <-chan []byte, err error)

	// Inspection
	ContainerStatus(ctx context.Context, id string) (Status, error)
	ContainerInfo(ctx context.Context, id string) (*ContainerInfo, error)
	ContainerStats(ctx context.Context, id string) (*Stats, error)
	ListContainers(ctx context.Context, filter map[string]string) ([]*ContainerInfo, error)

	// Runtime Info
	Type() RuntimeType
	Version(ctx context.Context) (string, error)
	IsHealthy(ctx context.Context) bool
}

// RuntimeConfig contains configuration for creating a runtime instance.
type RuntimeConfig struct {
	Type       RuntimeType            `json:"type"`
	SocketPath string                 `json:"socket_path,omitempty"` // Path to runtime socket
	Options    map[string]interface{} `json:"options,omitempty"`     // Runtime-specific options
}

// RuntimeFactory is a function that creates a runtime instance.
type RuntimeFactory func(cfg *RuntimeConfig) (Runtime, error)

// registry holds registered runtime factories.
var registry = make(map[RuntimeType]RuntimeFactory)

// RegisterRuntime registers a runtime factory for the given type.
func RegisterRuntime(t RuntimeType, factory RuntimeFactory) {
	registry[t] = factory
}

// NewRuntime creates a new runtime instance based on the configuration.
// This is the factory function for runtime adapters.
func NewRuntime(cfg *RuntimeConfig) (Runtime, error) {
	factory, ok := registry[cfg.Type]
	if !ok {
		return nil, ErrUnsupportedRuntime{Type: cfg.Type}
	}
	return factory(cfg)
}

// ErrUnsupportedRuntime is returned when an unsupported runtime type is requested.
type ErrUnsupportedRuntime struct {
	Type RuntimeType
}

func (e ErrUnsupportedRuntime) Error() string {
	return "unsupported runtime type: " + string(e.Type)
}

// DefaultContainerSpec returns a ContainerSpec with ArmorClaw security defaults.
func DefaultContainerSpec(image string) *ContainerSpec {
	return &ContainerSpec{
		Image:   image,
		User:    "10001:10001", // Non-root user
		CPUQuota: 100000,        // 1 CPU
		Memory:   512 * 1024 * 1024, // 512MB
		PidsLimit: 100,
		ReadOnlyRootFS: true,
		NoNewPrivileges: true,
		CapDrop: []string{"ALL"},
		NetworkDisabled: false, // Agents need network for API calls
		AutoRemove: true,
		StopTimeout: 30 * time.Second,
		RestartPolicy: "no",
		Labels: map[string]string{
			"armorclaw.managed": "true",
			"armorclaw.version": "1.0",
		},
	}
}

// EnterpriseContainerSpec returns a ContainerSpec with enterprise-grade security.
// This is stricter than the default, suitable for regulated environments.
func EnterpriseContainerSpec(image string) *ContainerSpec {
	spec := DefaultContainerSpec(image)
	spec.SeccompProfile = "armorclaw-enterprise"
	spec.AppArmorProfile = "armorclaw-enterprise"
	spec.NetworkDisabled = true // No direct network - must proxy through bridge
	spec.Memory = 256 * 1024 * 1024 // 256MB - stricter limit
	spec.PidsLimit = 50
	spec.TmpFS = []TmpFSSpec{
		{Target: "/tmp", Size: 64 * 1024 * 1024}, // 64MB tmpfs
		{Target: "/run", Size: 16 * 1024 * 1024}, // 16MB for runtime
	}
	return spec
}
