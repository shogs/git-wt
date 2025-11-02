package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	forceRemove bool
)

var removeCmd = &cobra.Command{
	Use:   "remove <branch>",
	Short: "Remove a worktree",
	Long:  `Safely removes a worktree after running teardown actions.`,
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

		// Check if we're inside the worktree
		inside, err := isInsideWorktree(wt.Path)
		if err != nil {
			return err
		}
		if inside {
			return fmt.Errorf("cannot remove worktree while inside it. Please navigate out first")
		}

		// Check for uncommitted changes
		if !forceRemove {
			hasChanges, err := hasUncommittedChanges(wt.Path)
			if err != nil {
				return fmt.Errorf("failed to check for uncommitted changes: %w", err)
			}
			if hasChanges {
				return fmt.Errorf("worktree has uncommitted changes. Commit or stash them first, or use --force")
			}
		}

		// Load config for teardown actions
		config, err := loadConfig()
		if err != nil {
			color.Yellow("⚠ Failed to load config: %v", err)
		} else if len(config.Teardown) > 0 {
			fmt.Println("Running teardown actions...")
			// Load session for environment variables
			session, _ := loadSession(wt.Path)
			baseBranch := ""
			if session != nil {
				baseBranch = session.BaseBranch
			}
			if err := runActions(config.Teardown, wt.Path, branch, baseBranch); err != nil {
				color.Yellow("⚠ Teardown actions completed with errors")
			}
		}

		// Remove session file
		if err := deleteSession(wt.Path); err != nil {
			color.Yellow("⚠ Failed to remove session file: %v", err)
		}

		// Remove worktree
		color.Blue("Removing worktree...")
		if err := removeWorktree(wt.Path, forceRemove); err != nil {
			return fmt.Errorf("failed to remove worktree: %w", err)
		}

		color.Green("✓ Worktree removed: %s", branch)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().BoolVarP(&forceRemove, "force", "f", false, "Force removal even with uncommitted changes")
}
