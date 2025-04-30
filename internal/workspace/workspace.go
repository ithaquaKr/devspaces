package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ithaquaKr/devspaces/internal/config"
	"github.com/ithaquaKr/devspaces/internal/docker"
	"github.com/ithaquaKr/devspaces/internal/git"
)

// Workspace represents a development workspace
type Workspace struct {
	Name      string
	Config    *config.Config
	RepoPath  string
	WorkPath  string
	DockerMgr *docker.Manager
}

// New creates a new workspace
func New(name string, cfg *config.Config, repoPath string) *Workspace {
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		absRepoPath = repoPath
	}

	workPath := filepath.Join(absRepoPath, "..", fmt.Sprintf("workspace-%s", name))

	dockerMgr, err := docker.NewManager()
	if err != nil {
		fmt.Printf("Warning: failed to create Docker manager: %v\n", err)
	}

	return &Workspace{
		Name:      name,
		Config:    cfg,
		RepoPath:  absRepoPath,
		WorkPath:  workPath,
		DockerMgr: dockerMgr,
	}
}

// Setup sets up a new workspace
func (w *Workspace) Setup() error {
	// Start Docker containers
	return w.startContainers()
}

// Start starts an existing workspace
func (w *Workspace) Start() error {
	// Check if workspace exists
	if _, err := os.Stat(w.WorkPath); os.IsNotExist(err) {
		return fmt.Errorf("workspace does not exist: %s", w.Name)
	}

	// Start Docker containers
	return w.startContainers()
}

// Stop stops a running workspace
func (w *Workspace) Stop() error {
	if w.DockerMgr == nil {
		return fmt.Errorf("Docker manager is not available")
	}

	// Get all containers for this workspace
	containers, err := w.DockerMgr.GetContainersByWorkspace(w.Name)
	if err != nil {
		return fmt.Errorf("failed to get containers: %w", err)
	}

	// Stop all containers
	for _, container := range containers {
		if err := w.DockerMgr.StopContainer(container.ID); err != nil {
			return fmt.Errorf("failed to stop container %s: %w", container.ID, err)
		}
		fmt.Printf("Stopped container %s\n", container.Names[0])
	}

	return nil
}

// Remove removes a workspace
func (w *Workspace) Remove() error {
	// Stop and remove containers
	if err := w.removeContainers(); err != nil {
		return fmt.Errorf("failed to remove containers: %w", err)
	}

	// Remove git worktree
	if err := git.RemoveWorktree(w.RepoPath, w.Name); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	return nil
}

// List lists all workspaces
func List(repoPath string) ([]string, error) {
	return git.ListWorktrees(repoPath)
}

// startContainers starts all Docker containers for the workspace
func (w *Workspace) startContainers() error {
	if w.DockerMgr == nil {
		return fmt.Errorf("Docker manager is not available")
	}

	// Check if containers already exist
	existingContainers, err := w.DockerMgr.GetContainersByWorkspace(w.Name)
	if err != nil {
		return fmt.Errorf("failed to get containers: %w", err)
	}

	// If containers exist, start them
	if len(existingContainers) > 0 {
		for _, container := range existingContainers {
			if err := w.DockerMgr.StartContainer(container.ID); err != nil {
				return fmt.Errorf("failed to start container %s: %w", container.Names[0], err)
			}
			fmt.Printf("Started container %s\n", container.Names[0])
		}
		return nil
	}

	// Otherwise, create and start new containers
	for serviceName, serviceConfig := range w.Config.Services {
		// Update volume paths to be relative to the workspace
		volumes := make([]string, len(serviceConfig.Volumes))
		for i, volume := range serviceConfig.Volumes {
			parts := strings.Split(volume, ":")
			if len(parts) >= 2 && !filepath.IsAbs(parts[0]) {
				parts[0] = filepath.Join(w.WorkPath, parts[0])
				volumes[i] = strings.Join(parts, ":")
			} else {
				volumes[i] = volume
			}
		}
		serviceConfig.Volumes = volumes

		containerID, err := w.DockerMgr.CreateContainer(w.Name, serviceName, serviceConfig)
		if err != nil {
			return fmt.Errorf("failed to create container for service %s: %w", serviceName, err)
		}

		if err := w.DockerMgr.StartContainer(containerID); err != nil {
			return fmt.Errorf("failed to start container for service %s: %w", serviceName, err)
		}

		fmt.Printf("Started container for service %s\n", serviceName)
	}

	return nil
}

// removeContainers removes all Docker containers for the workspace
func (w *Workspace) removeContainers() error {
	if w.DockerMgr == nil {
		return fmt.Errorf("Docker manager is not available")
	}

	// Get all containers for this workspace
	containers, err := w.DockerMgr.GetContainersByWorkspace(w.Name)
	if err != nil {
		return fmt.Errorf("failed to get containers: %w", err)
	}

	// Remove all containers
	for _, container := range containers {
		if err := w.DockerMgr.RemoveContainer(container.ID); err != nil {
			return fmt.Errorf("failed to remove container %s: %w", container.Names[0], err)
		}
		fmt.Printf("Removed container %s\n", container.Names[0])
	}

	return nil
}
