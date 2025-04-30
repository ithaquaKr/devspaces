package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ithaquaKr/devspaces/internal/config"
	"github.com/ithaquaKr/devspaces/internal/git"
	"github.com/ithaquaKr/devspaces/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	configFile string
	repoPath   string
	branch     string
)

var createCmd = &cobra.Command{
	Use:   "create [workspace-name]",
	Short: "Create a new workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceName := args[0]

		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := git.CreateWorktree(repoPath, workspaceName, branch); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}

		ws := workspace.New(workspaceName, cfg, repoPath)
		if err := ws.Setup(); err != nil {
			return fmt.Errorf("failed to setup workspace: %w", err)
		}

		fmt.Printf("Workspace '%s' created successfully!\n", workspaceName)
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start [workspace-name]",
	Short: "Start an existing workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceName := args[0]

		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		ws := workspace.New(workspaceName, cfg, repoPath)
		if err := ws.Start(); err != nil {
			return fmt.Errorf("failed to start workspace: %w", err)
		}

		fmt.Printf("Workspace '%s' started successfully!\n", workspaceName)
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop [workspace-name]",
	Short: "Stop a running workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceName := args[0]

		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		ws := workspace.New(workspaceName, cfg, repoPath)
		if err := ws.Stop(); err != nil {
			return fmt.Errorf("failed to stop workspace: %w", err)
		}

		fmt.Printf("Workspace '%s' stopped successfully!\n", workspaceName)
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workspaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaces, err := workspace.List(repoPath)
		if err != nil {
			return fmt.Errorf("failed to list workspaces: %w", err)
		}

		fmt.Println("Available workspaces:")
		for _, ws := range workspaces {
			fmt.Printf("- %s\n", ws)
		}
		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove [workspace-name]",
	Short: "Remove a workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceName := args[0]

		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		ws := workspace.New(workspaceName, cfg, repoPath)
		if err := ws.Remove(); err != nil {
			return fmt.Errorf("failed to remove workspace: %w", err)
		}

		fmt.Printf("Workspace '%s' removed successfully!\n", workspaceName)
		return nil
	},
}

var rootCmd = &cobra.Command{
	Use:   "devspace",
	Short: "Devspace management tool",
	Long: `A command-line interface (CLI) tool assists in creating
	development environments based on Git worktrees, with dependencies explicitly
	defined.`,
}

func execute() {
	// Handle signal to prevent bad exit codes on "docker stop"
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGABRT)
	go func() {
		<-sigs
		os.Exit(0)
	}()

	if err := rootCmd.Execute(); err != nil {
		slog.Error(fmt.Sprintf("error executing command: %s", err.Error()))
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "config.yaml", "Path to the configuration file")
	rootCmd.PersistentFlags().StringVar(&repoPath, "repo", ".", "Path to the git repository")
	createCmd.Flags().StringVar(&branch, "branch", "main", "Branch to create the worktree from")
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
}

func main() {
	execute()
}
