package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/ithaquaKr/devspaces/internal/config"
)

// Manager handles Docker operations
type Manager struct {
	client *client.Client
}

// NewManager creates a new Docker manager
func NewManager() (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Manager{client: cli}, nil
}

// PullImage pulls a Docker image
func (m *Manager) PullImage(imageName string) error {
	ctx := context.Background()
	resp, err := m.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer resp.Close()

	// Write the output to stdout
	_, err = io.Copy(os.Stdout, resp)
	return err
}

// CreateContainer creates a new Docker container
func (m *Manager) CreateContainer(workspaceName, serviceName string, service config.DockerService) (string, error) {
	ctx := context.Background()

	// Pull the image
	if err := m.PullImage(service.Image); err != nil {
		return "", err
	}

	// Prepare port bindings
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	for _, port := range service.Ports {
		parts := strings.Split(port, ":")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid port mapping format: %s", port)
		}
		hostPort := parts[0]
		containerPort := parts[1]
		port := nat.Port(fmt.Sprintf("%s/tcp", containerPort))
		exposedPorts[port] = struct{}{}
		portBindings[port] = []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: hostPort}}
	}

	// Prepare environment variables
	env := []string{}
	for key, value := range service.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Create the container
	containerName := fmt.Sprintf("%s-%s", workspaceName, serviceName)
	resp, err := m.client.ContainerCreate(
		ctx,
		&container.Config{
			Image:        service.Image,
			ExposedPorts: exposedPorts,
			Env:          env,
			Cmd:          service.Command,
		},
		&container.HostConfig{
			PortBindings: portBindings,
			Binds:        service.Volumes,
		},
		&network.NetworkingConfig{},
		nil,
		containerName,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container %s: %w", containerName, err)
	}

	return resp.ID, nil
}

// StartContainer starts a Docker container
func (m *Manager) StartContainer(containerID string) error {
	ctx := context.Background()
	return m.client.ContainerStart(ctx, containerID, container.StartOptions{})
}

// StopContainer stops a Docker container
func (m *Manager) StopContainer(containerID string) error {
	ctx := context.Background()
	timeout := int(30) // 30 seconds timeout
	return m.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

// RemoveContainer removes a Docker container
func (m *Manager) RemoveContainer(containerID string) error {
	ctx := context.Background()
	return m.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}

// GetContainersByWorkspace returns all containers for a workspace
func (m *Manager) GetContainersByWorkspace(workspaceName string) ([]container.Summary, error) {
	ctx := context.Background()
	return m.client.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("name", fmt.Sprintf("%s-", workspaceName)),
		),
	})
}
