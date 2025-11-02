package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	spawnShell bool
)

var switchCmd = &cobra.Command{
	Use:   "switch <branch>",
	Short: "Switch to a worktree",
	Long:  `Changes to the worktree directory for the specified branch.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureGitRepo(); err != nil {
			return err
		}

		branch := args[0]

		// Find worktree
		wt, err := findWorktreeByBranch(branch)
		if err != nil {
			return err
		}
		if wt == nil {
			return fmt.Errorf("no worktree found for branch '%s'", branch)
		}

		if spawnShell {
			// Spawn a new shell in the worktree
			return spawnWorktreeShell(wt.Path, branch)
		}

		// Just print the path for shell integration
		fmt.Println(wt.Path)
		return nil
	},
}

var resumeCmd = &cobra.Command{
	Use:   "resume <branch>",
	Short: "Resume work in a worktree",
	Long:  `Switches to a worktree and displays session information.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureGitRepo(); err != nil {
			return err
		}

		branch := args[0]

		// Find worktree
		wt, err := findWorktreeByBranch(branch)
		if err != nil {
			return err
		}
		if wt == nil {
			return fmt.Errorf("no worktree found for branch '%s'", branch)
		}

		// Load session
		session, err := loadSession(wt.Path)
		if err != nil {
			color.Yellow("⚠ Failed to load session: %v", err)
		}

		// Display session info
		if session != nil {
			fmt.Println()
			color.Cyan("═══ Resuming: %s ═══", branch)
			if session.Task != "" {
				fmt.Printf("Task: %s\n", session.Task)
			}
			fmt.Printf("Created: %s\n", session.Created.Format("2006-01-02 15:04"))
			fmt.Printf("Base: %s\n", session.BaseBranch)
			fmt.Println()
		}

		if spawnShell {
			return spawnWorktreeShell(wt.Path, branch)
		}

		// Just print the path for shell integration
		fmt.Println(wt.Path)
		return nil
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
	rootCmd.AddCommand(resumeCmd)

	switchCmd.Flags().BoolVarP(&spawnShell, "shell", "s", false, "Spawn a new shell in the worktree")
	resumeCmd.Flags().BoolVarP(&spawnShell, "shell", "s", false, "Spawn a new shell in the worktree")
}
