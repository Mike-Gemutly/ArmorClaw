package toolsidecar

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
)

type DockerClient interface {
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig any, platform any, name string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
}

type Config struct {
	DockerClient DockerClient
	DefaultImage string
}

type Provisioner struct {
	docker       DockerClient
	defaultImage string
}

func NewProvisioner(cfg Config) (*Provisioner, error) {
	if cfg.DockerClient == nil {
		return nil, fmt.Errorf("toolsidecar: docker client is required")
	}
	img := cfg.DefaultImage
	if img == "" {
		img = "armorclaw/toolsidecar:latest"
	}
	return &Provisioner{
		docker:       cfg.DockerClient,
		defaultImage: img,
	}, nil
}

type ToolSidecar struct {
	ID        string
	SkillName string
	SessionID string
	CreatedAt time.Time
	Status    string
}

func (p *Provisioner) SpawnToolSidecar(ctx context.Context, skillName, sessionID string) (*ToolSidecar, error) {
	containerName := fmt.Sprintf("toolsidecar-%s-%s", skillName, sessionID)

	config := &container.Config{
		Image: p.defaultImage,
		Cmd:   []string{"/bin/start-tool", skillName},
		Labels: map[string]string{
			"armorclaw.type":    "toolsidecar",
			"armorclaw.skill":   skillName,
			"armorclaw.session": sessionID,
			"armorclaw.created": time.Now().UTC().Format(time.RFC3339),
		},
	}

	hostConfig := &container.HostConfig{
		NetworkMode:    "none",
		AutoRemove:     true,
		Tmpfs:          map[string]string{"/workspace": "rw,noexec,nosuid,size=100m"},
		ReadonlyRootfs: true,
		Resources: container.Resources{
			Memory:     512 * 1024 * 1024,
			MemorySwap: 512 * 1024 * 1024,
		},
		SecurityOpt: []string{"no-new-privileges:true"},
		CapDrop:     []string{"ALL"},
	}

	createResp, err := p.docker.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("toolsidecar: failed to create container for skill %s: %w", skillName, err)
	}

	if err := p.docker.ContainerStart(ctx, createResp.ID, container.StartOptions{}); err != nil {
		_ = p.docker.ContainerRemove(ctx, createResp.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("toolsidecar: failed to start container %s for skill %s: %w", createResp.ID[:12], skillName, err)
	}

	return &ToolSidecar{
		ID:        createResp.ID,
		SkillName: skillName,
		SessionID: sessionID,
		CreatedAt: time.Now(),
		Status:    "running",
	}, nil
}

func (p *Provisioner) StopToolSidecar(ctx context.Context, containerID string) error {
	if err := p.docker.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		return p.docker.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
	}
	return nil
}
