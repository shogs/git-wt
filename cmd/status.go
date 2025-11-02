package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show repository and worktree status",
	Long:  `Displays the overall git repository status and information about all worktrees.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureGitRepo(); err != nil {
			return err
		}

		gitRoot, _ := getGitRoot()
		currentBranch, _ := getCurrentBranch()

		// Display header
		color.Cyan("═══ Git Worktree Status ═══\n")
		fmt.Printf("Repository: %s\n", gitRoot)
		fmt.Printf("Current branch: %s\n", currentBranch)
		fmt.Println()

		// Get worktrees
		worktrees, err := getWorktrees()
		if err != nil {
			return fmt.Errorf("failed to get worktrees: %w", err)
		}

		if len(worktrees) == 0 {
			color.Yellow("No worktrees found")
			return nil
		}

		// Display worktrees
		color.Cyan("Worktrees (%d):\n", len(worktrees))
		cwd, _ := os.Getwd()

		for _, wt := range worktrees {
			isMain := wt.Path == gitRoot
			isCurrent := wt.Path == cwd || (wt.Path != gitRoot && len(cwd) > len(wt.Path) && cwd[:len(wt.Path)] == wt.Path)

			prefix := "  "
			if isCurrent {
				prefix = "* "
				color.Green(prefix + wt.Branch)
			} else {
				fmt.Print(prefix + wt.Branch)
			}

			if isMain {
				color.Cyan(" [main]")
			}

			// Check for uncommitted changes
			if !isMain {
				hasChanges, err := hasUncommittedChanges(wt.Path)
				if err == nil && hasChanges {
					color.Yellow(" (uncommitted changes)")
				}
			}

			fmt.Println()
		}

		// Display git status
		fmt.Println()
		color.Cyan("Git Status:\n")
		gitCmd := exec.Command("git", "status", "--short", "--branch")
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		gitCmd.Run()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
