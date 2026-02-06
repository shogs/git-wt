package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	printPath bool
)

var switchCmd = &cobra.Command{
	Use:   "switch [branch]",
	Short: "Switch to a worktree",
	Long:  `Changes to the worktree directory for the specified branch. If no branch specified, shows interactive selection.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureGitRepo(); err != nil {
			return err
		}

		var branch string

		if len(args) == 0 {
			// Interactive mode - show single-select
			worktrees, err := getWorktrees()
			if err != nil {
				return fmt.Errorf("failed to get worktrees: %w", err)
			}

			// Get git root to filter out main worktree
			gitRoot, err := getGitRoot()
			if err != nil {
				return fmt.Errorf("failed to get git root: %w", err)
			}

			// Build list of selectable worktrees (exclude main)
			var items []string
			for _, wt := range worktrees {
				if wt.Path == gitRoot {
					continue // Skip main worktree
				}
				if wt.Branch == "" {
					continue // Skip detached worktrees
				}
				items = append(items, wt.Branch)
			}

			if len(items) == 0 {
				fmt.Println("No worktrees available to switch to.")
				return nil
			}

			// Show single-select
			selected, confirmed, err := promptSingleSelect("Select worktree:", items)
			if err != nil {
				return fmt.Errorf("failed to get selection: %w", err)
			}

			if !confirmed {
				fmt.Println("Cancelled.")
				return nil
			}

			branch = selected
		} else {
			branch = args[0]
		}

		// Find worktree
		wt, err := findWorktreeByBranch(branch)
		if err != nil {
			return err
		}
		if wt == nil {
			return fmt.Errorf("no worktree found for branch '%s'", branch)
		}

		if printPath {
			// Just print the path for shell integration
			fmt.Println(wt.Path)
			return nil
		}

		// Spawn a new shell in the worktree (default)
		return spawnWorktreeShell(wt.Path, branch)
	},
}

func spawnWorktreeShell(path, branch string) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	color.Green("Starting shell in worktree: %s", branch)
	fmt.Println("Type 'exit' to return")
	fmt.Println()

	cmd := exec.Command(shell)
	cmd.Dir = path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables
	cwd, _ := os.Getwd()
	cmd.Env = append(os.Environ(),
		"GIT_WT_SHELL=1",
		fmt.Sprintf("GIT_WT_BRANCH=%s", branch),
		fmt.Sprintf("GIT_WT_ORIGINAL_DIR=%s", cwd),
	)

	return cmd.Run()
}

func init() {
	rootCmd.AddCommand(switchCmd)
	switchCmd.Flags().BoolVarP(&printPath, "path", "p", false, "Print worktree path instead of spawning shell")
}
