package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CreateWorktree creates a new git worktree
func CreateWorktree(repoPath, workspaceName, branch string) error {
	// Change to the repository directory
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create the workspace directory
	workspacePath := filepath.Join(absRepoPath, "..", fmt.Sprintf("workspace-%s", workspaceName))

	// Ensure branch exists
	checkBranchCmd := exec.Command("git", "branch", "--list", branch)
	checkBranchCmd.Dir = absRepoPath
	output, err := checkBranchCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check branch: %w", err)
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		return fmt.Errorf("branch '%s' does not exist", branch)
	}

	// Create a new worktree
	createCmd := exec.Command("git", "worktree", "add", workspacePath, branch)
	createCmd.Dir = absRepoPath
	createCmd.Stdout = os.Stdout
	createCmd.Stderr = os.Stderr

	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	return nil
}

// RemoveWorktree removes a git worktree
func RemoveWorktree(repoPath, workspaceName string) error {
	// Change to the repository directory
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Workspace path
	workspacePath := filepath.Join(absRepoPath, "..", fmt.Sprintf("workspace-%s", workspaceName))

	// Remove the worktree
	removeCmd := exec.Command("git", "worktree", "remove", workspacePath)
	removeCmd.Dir = absRepoPath
	removeCmd.Stdout = os.Stdout
	removeCmd.Stderr = os.Stderr

	if err := removeCmd.Run(); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	return nil
}

// ListWorktrees lists all git worktrees
func ListWorktrees(repoPath string) ([]string, error) {
	// Change to the repository directory
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// List all worktrees
	listCmd := exec.Command("git", "worktree", "list", "--porcelain")
	listCmd.Dir = absRepoPath

	output, err := listCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Parse the output
	worktrees := []string{}
	sections := strings.Split(string(output), "\n\n")
	for _, section := range sections {
		if section == "" {
			continue
		}

		lines := strings.Split(section, "\n")
		if len(lines) == 0 {
			continue
		}

		// Extract the path from the first line
		pathLine := lines[0]
		if !strings.HasPrefix(pathLine, "worktree ") {
			continue
		}

		worktreePath := strings.TrimPrefix(pathLine, "worktree ")
		if strings.Contains(worktreePath, "workspace-") {
			// Extract the workspace name from the path
			base := filepath.Base(worktreePath)
			workspaceName := strings.TrimPrefix(base, "workspace-")
			worktrees = append(worktrees, workspaceName)
		}
	}

	return worktrees, nil
}
