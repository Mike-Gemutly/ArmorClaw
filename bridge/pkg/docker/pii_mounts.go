// Package docker provides PII mount helpers for container configuration.
// PII is mounted via tmpfs for memory-only access.
package docker

import (
	"fmt"
	"os"

	"github.com/docker/docker/api/types/mount"
)

const (
	// PIIMountPath is the path inside the container where PII is accessible
	PIIMountPath = "/run/armorclaw/pii"

	// PIIHostSocketDir is the host directory for PII sockets
	PIIHostSocketDir = "/run/armorclaw/pii"
)

// PIIMountConfig configures how PII is mounted into containers
type PIIMountConfig struct {
	// SocketPath is the path to the PII socket on the host
	SocketPath string `json:"socket_path"`

	// ContainerPath is where the socket is mounted in the container
	ContainerPath string `json:"container_path"`

	// ReadOnly indicates if the mount should be read-only
	ReadOnly bool `json:"read_only"`
}

// PreparePIIMounts creates Docker mount configurations for PII sockets
// These mounts are tmpfs-based to ensure memory-only access
func PreparePIIMounts(config *PIIMountConfig) ([]mount.Mount, error) {
	if config == nil {
		return nil, nil
	}

	mounts := []mount.Mount{}

	// Mount the PII socket
	if config.SocketPath != "" {
		// Verify socket exists
		if _, err := os.Stat(config.SocketPath); err != nil {
			return nil, fmt.Errorf("PII socket not found: %w", err)
		}

		containerPath := config.ContainerPath
		if containerPath == "" {
			containerPath = PIIMountPath + "/socket.sock"
		}

		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   config.SocketPath,
			Target:   containerPath,
			ReadOnly: config.ReadOnly,
			BindOptions: &mount.BindOptions{
				Propagation: mount.PropagationPrivate,
			},
		})
	}

	return mounts, nil
}

// PreparePIITmpfsMount creates a tmpfs mount for in-memory PII storage
// This is useful for storing PII data temporarily without writing to disk
func PreparePIITmpfsMount() mount.Mount {
	return mount.Mount{
		Type:   mount.TypeTmpfs,
		Target: PIIMountPath,
		TmpfsOptions: &mount.TmpfsOptions{
			SizeBytes: 1024 * 1024, // 1MB max
			Mode:      0770,
		},
	}
}

// PreparePIISocketMount creates a bind mount for the PII socket
func PreparePIISocketMount(socketPath string) mount.Mount {
	return mount.Mount{
		Type:     mount.TypeBind,
		Source:   socketPath,
		Target:   PIIMountPath + "/socket.sock",
		ReadOnly: true,
		BindOptions: &mount.BindOptions{
			Propagation: mount.PropagationPrivate,
		},
	}
}

// PreparePIIEnvironment creates environment variable assignments for PII injection
// This is used when setting up container environment variables
func PreparePIIEnvironment(piiVars map[string]string, prefix string) []string {
	if prefix == "" {
		prefix = "PII_"
	}

	envVars := []string{}

	// Add PII socket path
	envVars = append(envVars, fmt.Sprintf("%sSOCKET_PATH=%s/socket.sock", prefix, PIIMountPath))

	// Add individual PII variables
	for key, value := range piiVars {
		envVars = append(envVars, fmt.Sprintf("%s%s=%s", prefix, key, value))
	}

	return envVars
}

// ValidatePIIMountConfig validates the PII mount configuration
func ValidatePIIMountConfig(config *PIIMountConfig) error {
	if config == nil {
		return nil
	}

	if config.SocketPath != "" {
		if _, err := os.Stat(config.SocketPath); err != nil {
			return fmt.Errorf("PII socket not accessible: %w", err)
		}
	}

	return nil
}
