// Package docker provides a Docker runtime adapter for ArmorClaw.
// It implements the runtime.Runtime interface using the Docker Engine API.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/armorclaw/bridge/pkg/runtime"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// DockerRuntime implements runtime.Runtime using Docker Engine.
type DockerRuntime struct {
	client *client.Client
}

// newDockerRuntime creates a new Docker runtime adapter.
func newDockerRuntime(cfg *runtime.RuntimeConfig) (runtime.Runtime, error) {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}

	if cfg.SocketPath != "" {
		opts = append(opts, client.WithHost("unix://"+cfg.SocketPath))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerRuntime{client: cli}, nil
}

func init() {
	// Register Docker runtime factory
	runtime.RegisterRuntime(runtime.RuntimeDocker, func(cfg *runtime.RuntimeConfig) (runtime.Runtime, error) {
		return newDockerRuntime(cfg)
	})
}

// Type returns the runtime type.
func (d *DockerRuntime) Type() runtime.RuntimeType {
	return runtime.RuntimeDocker
}

// Version returns the Docker server version.
func (d *DockerRuntime) Version(ctx context.Context) (string, error) {
	ver, err := d.client.ServerVersion(ctx)
	if err != nil {
		return "", err
	}
	return ver.Version, nil
}

// IsHealthy checks if Docker daemon is accessible.
func (d *DockerRuntime) IsHealthy(ctx context.Context) bool {
	_, err := d.client.Ping(ctx)
	return err == nil
}

// CreateContainer creates a new container with the specified configuration.
func (d *DockerRuntime) CreateContainer(ctx context.Context, spec *runtime.ContainerSpec) (string, error) {
	// Build container config
	config := &containertypes.Config{
		Image:  spec.Image,
		Labels: spec.Labels,
		Env:    envMapToSlice(spec.EnvVars),
	}

	if len(spec.Cmd) > 0 {
		config.Cmd = spec.Cmd
	}
	if len(spec.Entrypoint) > 0 {
		config.Entrypoint = spec.Entrypoint
	}
	if spec.User != "" {
		config.User = spec.User
	}
	if spec.WorkingDir != "" {
		config.WorkingDir = spec.WorkingDir
	}

	// Build host config with security hardening
	hostConfig := &containertypes.HostConfig{
		AutoRemove: spec.AutoRemove,
		Resources: containertypes.Resources{
			CPUQuota:  spec.CPUQuota,
			Memory:    spec.Memory,
			PidsLimit: &spec.PidsLimit,
		},
		ReadonlyRootfs: spec.ReadOnlyRootFS,
		SecurityOpt:    buildSecurityOpts(spec),
	}

	// Drop all capabilities by default, add specified ones
	if len(spec.CapDrop) > 0 {
		hostConfig.CapDrop = spec.CapDrop
	}
	if len(spec.CapAdd) > 0 {
		hostConfig.CapAdd = spec.CapAdd
	}

	// Network configuration
	if spec.NetworkDisabled {
		hostConfig.NetworkMode = "none"
	}
	if len(spec.DNS) > 0 {
		hostConfig.DNS = spec.DNS
	}
	if len(spec.DNSSearch) > 0 {
		hostConfig.DNSSearch = spec.DNSSearch
	}

	// Volume mounts
	hostConfig.Binds = buildBinds(spec.Volumes)
	hostConfig.Tmpfs = buildTmpfs(spec.TmpFS)

	// Restart policy
	if spec.RestartPolicy != "" {
		hostConfig.RestartPolicy = containertypes.RestartPolicy{Name: containertypes.RestartPolicyMode(spec.RestartPolicy)}
	}

	// Create container
	name := spec.Name

	resp, err := d.client.ContainerCreate(ctx, config, hostConfig, nil, nil, name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// StartContainer starts a created container.
func (d *DockerRuntime) StartContainer(ctx context.Context, id string) error {
	return d.client.ContainerStart(ctx, id, containertypes.StartOptions{})
}

// StopContainer stops a running container.
func (d *DockerRuntime) StopContainer(ctx context.Context, id string, timeout *time.Duration) error {
	var timeoutSec *int
	if timeout != nil {
		sec := int(timeout.Seconds())
		timeoutSec = &sec
	}
	return d.client.ContainerStop(ctx, id, containertypes.StopOptions{Timeout: timeoutSec})
}

// RemoveContainer removes a container.
func (d *DockerRuntime) RemoveContainer(ctx context.Context, id string, force bool) error {
	return d.client.ContainerRemove(ctx, id, containertypes.RemoveOptions{
		Force:         force,
		RemoveVolumes: true,
	})
}

// ExecContainer executes a command in a running container.
// TODO: Fix for Docker API v28 - type signatures changed
func (d *DockerRuntime) ExecContainer(ctx context.Context, id string, config *runtime.ExecConfig) (*runtime.ExecResult, error) {
	return nil, fmt.Errorf("ExecContainer not yet implemented for Docker API v28")
}

// ExecContainerStream executes a command and streams output.
// TODO: Fix for Docker API v28 - type signatures changed
func (d *DockerRuntime) ExecContainerStream(ctx context.Context, id string, config *runtime.ExecConfig) (<-chan []byte, <-chan []byte, error) {
	return nil, nil, fmt.Errorf("ExecContainerStream not yet implemented for Docker API v28")
}

// ExecContainerStream executes a command and streams output.
// TODO: Fix for Docker API v28 - type signatures changed
// func (d *DockerRuntime) ExecContainerStream(ctx context.Context, id string, config *runtime.ExecConfig) (<-chan []byte, <-chan []byte, error) {
// 	// Create exec instance
// 	execConfig := containertypes.ExecConfig{
// 		Cmd:          config.Cmd,
// 		Env:          config.Env,
// 		User:         config.User,
// 		WorkingDir:   config.WorkingDir,
// 		Tty:          config.Tty,
// 		AttachStdout: true,
// 		AttachStderr: true,
// 	}
//
// 	execResp, err := d.client.ContainerExecCreate(ctx, id, execConfig)
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("failed to create exec: %w", err)
// 	}
//
// 	// Attach to exec
// 	attachResp, err := d.client.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{
// 		Tty: config.Tty,
// 	})
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("failed to attach exec: %w", err)
// 	}
//
// 	// Create channels for streaming
// 	stdoutCh := make(chan []byte, 100)
// 	stderrCh := make(chan []byte, 100)
//
// 	go func() {
// 		defer close(stdoutCh)
// 		defer close(stderrCh)
// 		defer attachResp.Close()
//
// 		buf := make([]byte, 4096)
// 		for {
// 			n, err := attachResp.Reader.Read(buf)
// 			if n > 0 {
// 				// Docker multiplexes stdout/stderr with an 8-byte header
// 				// For simplicity, send all to stdout
// 				data := make([]byte, n)
// 				copy(data, buf[:n])
// 				stdoutCh <- data
// 			}
// 			if err != nil {
// 				break
// 			}
// 		}
// 	}()
//
// 	return stdoutCh, stderrCh, nil
// }

// ContainerStatus returns the current status of a container.
func (d *DockerRuntime) ContainerStatus(ctx context.Context, id string) (runtime.Status, error) {
	inspect, err := d.client.ContainerInspect(ctx, id)
	if err != nil {
		return runtime.StatusUnknown, err
	}

	return mapDockerStatus(inspect.State.Status), nil
}

// ContainerInfo returns detailed information about a container.
func (d *DockerRuntime) ContainerInfo(ctx context.Context, id string) (*runtime.ContainerInfo, error) {
	inspect, err := d.client.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}

	// Parse created timestamp
	var created time.Time
	if inspect.Created != "" {
		created, _ = time.Parse(time.RFC3339Nano, inspect.Created)
	}

	info := &runtime.ContainerInfo{
		ID:      inspect.ID,
		Name:    inspect.Name,
		Image:   inspect.Config.Image,
		Status:  mapDockerStatus(inspect.State.Status),
		Created: created,
		Labels:  inspect.Config.Labels,
	}

	// Get IP address
	if inspect.NetworkSettings != nil {
		info.IPAddress = inspect.NetworkSettings.IPAddress
	}

	// Get port mappings
	for port, bindings := range inspect.NetworkSettings.Ports {
		for _, binding := range bindings {
			info.Ports = append(info.Ports, runtime.PortMapping{
				HostPort:      mustParseInt(binding.HostPort),
				ContainerPort: mustParseInt(port.Port()),
				Protocol:      port.Proto(),
			})
		}
	}

	return info, nil
}

// ContainerStats returns resource usage statistics.
func (d *DockerRuntime) ContainerStats(ctx context.Context, id string) (*runtime.Stats, error) {
	resp, err := d.client.ContainerStats(ctx, id, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats containertypes.Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	// Calculate CPU percentage (simplified)
	cpuPercent := 0.0
	if stats.CPUStats.CPUUsage.TotalUsage > 0 && stats.PreCPUStats.CPUUsage.TotalUsage > 0 {
		cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
		if systemDelta > 0 {
			cpuPercent = (cpuDelta / systemDelta) * 100.0
		}
	}

	// Calculate memory percentage
	memPercent := 0.0
	if stats.MemoryStats.Limit > 0 {
		memPercent = float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
	}

	return &runtime.Stats{
		CPUPercent:    cpuPercent,
		MemoryUsage:   int64(stats.MemoryStats.Usage),
		MemoryLimit:   int64(stats.MemoryStats.Limit),
		MemoryPercent: memPercent,
		NetworkRx:     sumNetworkStats(stats.Networks),
		NetworkTx:     0, // Simplified
		BlockRead:     sumBlockIOStats(stats.BlkioStats.IoServiceBytesRecursive, "Read"),
		BlockWrite:    sumBlockIOStats(stats.BlkioStats.IoServiceBytesRecursive, "Write"),
		Pids:          int64(stats.PidsStats.Current),
	}, nil
}

// ListContainers lists containers matching the filter.
func (d *DockerRuntime) ListContainers(ctx context.Context, filter map[string]string) ([]*runtime.ContainerInfo, error) {
	filterArgs := filters.NewArgs()
	for k, v := range filter {
		filterArgs.Add(k, v)
	}

	containers, err := d.client.ContainerList(ctx, containertypes.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*runtime.ContainerInfo, len(containers))
	for i, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
		}
		result[i] = &runtime.ContainerInfo{
			ID:      c.ID,
			Name:    name,
			Image:   c.Image,
			Status:  mapDockerStatus(c.State),
			Created: time.Unix(c.Created, 0),
			Labels:  c.Labels,
		}
	}

	return result, nil
}

// Helper functions

func mapDockerStatus(status string) runtime.Status {
	switch status {
	case "created":
		return runtime.StatusCreated
	case "running":
		return runtime.StatusRunning
	case "paused":
		return runtime.StatusPaused
	case "exited":
		return runtime.StatusExited
	case "dead":
		return runtime.StatusDead
	default:
		return runtime.StatusUnknown
	}
}

func envMapToSlice(env map[string]string) []string {
	if env == nil {
		return nil
	}
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}

func buildSecurityOpts(spec *runtime.ContainerSpec) []string {
	opts := []string{}

	if spec.NoNewPrivileges {
		opts = append(opts, "no-new-privileges")
	}

	if spec.SeccompProfile != "" {
		opts = append(opts, "seccomp="+spec.SeccompProfile)
	}

	if spec.AppArmorProfile != "" {
		opts = append(opts, "apparmor="+spec.AppArmorProfile)
	}

	return opts
}

func buildBinds(volumes []runtime.VolumeSpec) []string {
	if volumes == nil {
		return nil
	}
	binds := make([]string, len(volumes))
	for i, v := range volumes {
		bind := v.Source + ":" + v.Target
		if v.ReadOnly {
			bind += ":ro"
		}
		binds[i] = bind
	}
	return binds
}

func buildTmpfs(tmpfs []runtime.TmpFSSpec) map[string]string {
	if tmpfs == nil {
		return nil
	}
	result := make(map[string]string)
	for _, t := range tmpfs {
		result[t.Target] = "size=" + strconv.FormatInt(t.Size, 10)
	}
	return result
}

func mustParseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func sumNetworkStats(networks map[string]containertypes.NetworkStats) int64 {
	var total int64
	for _, stats := range networks {
		total += int64(stats.RxBytes)
	}
	return total
}

func sumBlockIOStats(stats []containertypes.BlkioStatEntry, op string) int64 {
	var total int64
	for _, s := range stats {
		if s.Op == op {
			total += int64(s.Value)
		}
	}
	return total
}
