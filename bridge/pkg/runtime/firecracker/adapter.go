// Package firecracker provides a Firecracker microVM runtime adapter for ArmorClaw.
// This is a placeholder for future implementation.
//
// Status: NOT IMPLEMENTED
// Planned: On enterprise demand
//
// Firecracker provides microVM-level isolation:
// - VM-grade isolation between agents
// - Strong tenant separation
// - Prevents container escape
// - Used by AWS Lambda and Fly.io
//
// Requirements:
// - KVM access (/dev/kvm)
// - Linux host
// - Additional resource overhead (~200ms startup, higher memory)
package firecracker

import (
	"context"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/runtime"
)

// FirecrackerRuntime implements runtime.Runtime using Firecracker microVMs.
// TODO: Implement when enterprise customers require it
type FirecrackerRuntime struct {
	// socketPath string
	// vmManager  *FirecrackerClient
}

func init() {
	// Register Firecracker runtime factory
	runtime.RegisterRuntime(runtime.RuntimeFirecracker, func(cfg *runtime.RuntimeConfig) (runtime.Runtime, error) {
		return newFirecrackerRuntime(cfg)
	})
}

func newFirecrackerRuntime(cfg *runtime.RuntimeConfig) (runtime.Runtime, error) {
	return nil, fmt.Errorf("firecracker runtime not implemented - available on enterprise request")
}

// Type returns the runtime type.
func (f *FirecrackerRuntime) Type() runtime.RuntimeType {
	return runtime.RuntimeFirecracker
}

// Version returns the Firecracker version.
func (f *FirecrackerRuntime) Version(ctx context.Context) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// IsHealthy checks if Firecracker/KVM is accessible.
func (f *FirecrackerRuntime) IsHealthy(ctx context.Context) bool {
	return false
}

// CreateContainer creates a new microVM.
func (f *FirecrackerRuntime) CreateContainer(ctx context.Context, spec *runtime.ContainerSpec) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// StartContainer starts a microVM.
func (f *FirecrackerRuntime) StartContainer(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

// StopContainer stops a microVM.
func (f *FirecrackerRuntime) StopContainer(ctx context.Context, id string, timeout *time.Duration) error {
	return fmt.Errorf("not implemented")
}

// RemoveContainer removes a microVM.
func (f *FirecrackerRuntime) RemoveContainer(ctx context.Context, id string, force bool) error {
	return fmt.Errorf("not implemented")
}

// ExecContainer executes a command in a microVM.
func (f *FirecrackerRuntime) ExecContainer(ctx context.Context, id string, config *runtime.ExecConfig) (*runtime.ExecResult, error) {
	return nil, fmt.Errorf("not implemented")
}

// ExecContainerStream executes a command and streams output.
func (f *FirecrackerRuntime) ExecContainerStream(ctx context.Context, id string, config *runtime.ExecConfig) (<-chan []byte, <-chan []byte, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

// ContainerStatus returns microVM status.
func (f *FirecrackerRuntime) ContainerStatus(ctx context.Context, id string) (runtime.Status, error) {
	return runtime.StatusUnknown, fmt.Errorf("not implemented")
}

// ContainerInfo returns microVM information.
func (f *FirecrackerRuntime) ContainerInfo(ctx context.Context, id string) (*runtime.ContainerInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

// ContainerStats returns microVM statistics.
func (f *FirecrackerRuntime) ContainerStats(ctx context.Context, id string) (*runtime.Stats, error) {
	return nil, fmt.Errorf("not implemented")
}

// ListContainers lists microVMs.
func (f *FirecrackerRuntime) ListContainers(ctx context.Context, filter map[string]string) ([]*runtime.ContainerInfo, error) {
	return nil, fmt.Errorf("not implemented")
}
