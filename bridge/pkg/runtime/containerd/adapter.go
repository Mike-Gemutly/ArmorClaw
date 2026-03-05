// Package containerd provides a containerd runtime adapter for ArmorClaw.
// This is a placeholder for future implementation.
//
// Status: NOT IMPLEMENTED
// Planned: v5.0
//
// containerd offers improved security and Kubernetes compatibility:
// - Reduced attack surface (no Docker daemon)
// - Native container runtime
// - Kubernetes-compatible
// - Better isolation primitives
package containerd

import (
	"context"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/runtime"
)

// ContainerdRuntime implements runtime.Runtime using containerd.
// TODO: Implement for v5.0
type ContainerdRuntime struct {
	// socketPath string
	// client     *containerd.Client
}

func init() {
	// Register containerd runtime factory
	runtime.RegisterRuntime(runtime.RuntimeContainerd, func(cfg *runtime.RuntimeConfig) (runtime.Runtime, error) {
		return newContainerdRuntime(cfg)
	})
}

func newContainerdRuntime(cfg *runtime.RuntimeConfig) (runtime.Runtime, error) {
	return nil, fmt.Errorf("containerd runtime not implemented - planned for v5.0")
}

// Type returns the runtime type.
func (c *ContainerdRuntime) Type() runtime.RuntimeType {
	return runtime.RuntimeContainerd
}

// Version returns the containerd version.
func (c *ContainerdRuntime) Version(ctx context.Context) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// IsHealthy checks if containerd is accessible.
func (c *ContainerdRuntime) IsHealthy(ctx context.Context) bool {
	return false
}

// CreateContainer creates a new container.
func (c *ContainerdRuntime) CreateContainer(ctx context.Context, spec *runtime.ContainerSpec) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// StartContainer starts a container.
func (c *ContainerdRuntime) StartContainer(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

// StopContainer stops a container.
func (c *ContainerdRuntime) StopContainer(ctx context.Context, id string, timeout *time.Duration) error {
	return fmt.Errorf("not implemented")
}

// RemoveContainer removes a container.
func (c *ContainerdRuntime) RemoveContainer(ctx context.Context, id string, force bool) error {
	return fmt.Errorf("not implemented")
}

// ExecContainer executes a command in a container.
func (c *ContainerdRuntime) ExecContainer(ctx context.Context, id string, config *runtime.ExecConfig) (*runtime.ExecResult, error) {
	return nil, fmt.Errorf("not implemented")
}

// ExecContainerStream executes a command and streams output.
func (c *ContainerdRuntime) ExecContainerStream(ctx context.Context, id string, config *runtime.ExecConfig) (<-chan []byte, <-chan []byte, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

// ContainerStatus returns container status.
func (c *ContainerdRuntime) ContainerStatus(ctx context.Context, id string) (runtime.Status, error) {
	return runtime.StatusUnknown, fmt.Errorf("not implemented")
}

// ContainerInfo returns container information.
func (c *ContainerdRuntime) ContainerInfo(ctx context.Context, id string) (*runtime.ContainerInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

// ContainerStats returns container statistics.
func (c *ContainerdRuntime) ContainerStats(ctx context.Context, id string) (*runtime.Stats, error) {
	return nil, fmt.Errorf("not implemented")
}

// ListContainers lists containers.
func (c *ContainerdRuntime) ListContainers(ctx context.Context, filter map[string]string) ([]*runtime.ContainerInfo, error) {
	return nil, fmt.Errorf("not implemented")
}
