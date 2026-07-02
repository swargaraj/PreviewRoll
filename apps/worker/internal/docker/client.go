package docker

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Client wraps the Docker client
type Client struct {
	client *client.Client
	logger *slog.Logger
}

// NewClient creates a new Docker client
func NewClient(socketPath string) (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(socketPath),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Client{
		client: cli,
		logger: slog.With("component", "docker"),
	}, nil
}

// Close closes the Docker client
func (c *Client) Close() error {
	return c.client.Close()
}

// ContainerConfig holds configuration for creating a container
type ContainerConfig struct {
	Image       string
	Command     []string
	Env         []string
	Ports       []string
	Labels      map[string]string
	Memory      int64
	CPU         int64
	Volumes     []string
	NetworkMode string
}

// ContainerResult holds the result of creating a container
type ContainerResult struct {
	ID        string
	IPAddress string
	Ports     nat.PortMap
}

// Run creates and starts a container
func (c *Client) Run(ctx context.Context, cfg ContainerConfig) (*ContainerResult, error) {
	c.logger.Info("pulling image", "image", cfg.Image)

	// Pull image
	reader, err := c.client.ImagePull(ctx, cfg.Image, image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Wait for pull to complete
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	// Create port bindings
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}

	for _, port := range cfg.Ports {
		natPort, err := nat.NewPort("tcp", port)
		if err != nil {
			return nil, fmt.Errorf("failed to parse port: %w", err)
		}
		exposedPorts[natPort] = struct{}{}
		portBindings[natPort] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: port,
			},
		}
	}

	// Create container config
	containerConfig := &container.Config{
		Image:        cfg.Image,
		Cmd:          cfg.Command,
		Env:          cfg.Env,
		ExposedPorts: exposedPorts,
		Labels:       cfg.Labels,
	}

	// Create host config
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	// Set resource limits
	if cfg.Memory > 0 || cfg.CPU > 0 {
		hostConfig.Resources = container.Resources{}
		if cfg.Memory > 0 {
			hostConfig.Resources.Memory = cfg.Memory
		}
		if cfg.CPU > 0 {
			hostConfig.Resources.NanoCPUs = cfg.CPU
		}
	}

	// Create network config
	networkConfig := &network.NetworkingConfig{}

	// Generate container name
	containerName := fmt.Sprintf("previewroll-%d", time.Now().UnixNano())

	// Create container
	resp, err := c.client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := c.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Get container info
	info, err := c.client.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return &ContainerResult{
		ID:        resp.ID,
		IPAddress: info.NetworkSettings.IPAddress,
		Ports:     info.NetworkSettings.Ports,
	}, nil
}

// Stop stops a container
func (c *Client) Stop(ctx context.Context, containerID string, timeout time.Duration) error {
	c.logger.Info("stopping container", "container_id", containerID)

	timeoutSeconds := int(timeout.Seconds())
	return c.client.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeoutSeconds,
	})
}

// Remove removes a container
func (c *Client) Remove(ctx context.Context, containerID string, force bool) error {
	c.logger.Info("removing container", "container_id", containerID)

	return c.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: force,
	})
}

// Logs returns the logs for a container
func (c *Client) Logs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	return c.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
}

// Inspect returns container information
func (c *Client) Inspect(ctx context.Context, containerID string) (*container.InspectResponse, error) {
	resp, err := c.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// List lists containers
func (c *Client) List(ctx context.Context, all bool) ([]container.Summary, error) {
	return c.client.ContainerList(ctx, container.ListOptions{
		All: all,
	})
}

// WaitForHealth waits for a container to become healthy
func (c *Client) WaitForHealth(ctx context.Context, containerID string, timeout time.Duration) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeoutCh := time.After(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeoutCh:
			return fmt.Errorf("timeout waiting for container health")
		case <-ticker.C:
			info, err := c.client.ContainerInspect(ctx, containerID)
			if err != nil {
				continue
			}

			if info.State.Health != nil {
				switch info.State.Health.Status {
				case "healthy":
					return nil
				case "unhealthy":
					return fmt.Errorf("container is unhealthy")
				}
			}

			// If no health check configured, check if running
			if info.State.Running {
				return nil
			}
		}
	}
}

// Cleanup removes stopped containers
func (c *Client) Cleanup(ctx context.Context, labelFilter string) error {
	containers, err := c.client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", labelFilter)),
	})
	if err != nil {
		return err
	}

	for _, ctr := range containers {
		if err := c.Remove(ctx, ctr.ID, true); err != nil {
			c.logger.Error("failed to remove container", "container_id", ctr.ID, "error", err)
		}
	}

	return nil
}
