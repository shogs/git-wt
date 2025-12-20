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
	Long:  `Safely removes a worktree after running remove actions.`,
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

		// Load session early to get baseBranch for unpushed commit check
		session, _ := loadSession(wt.Path)
		baseBranch := ""
		if session != nil {
			baseBranch = session.BaseBranch
		}

		// Check for uncommitted changes
		hasChanges, err := hasUncommittedChanges(wt.Path)
		if err != nil {
			return fmt.Errorf("failed to check for uncommitted changes: %w", err)
		}

		// Check for unpushed commits (works for both pushed and unpushed branches)
		unpushedCount, _ := getUnpushedCommitCount(wt.Path, branch, baseBranch)

		changesStashed := false
		if (hasChanges || unpushedCount > 0) && !forceRemove {
			// Show the changes if any
			if hasChanges {
				status, err := getGitStatus(wt.Path)
				if err != nil {
					color.Yellow("⚠ Failed to get status: %v", err)
				} else {
					color.Yellow("Uncommitted changes in worktree:")
					fmt.Println(status)
				}
			}

			// Show unpushed commits if any
			if unpushedCount > 0 {
				color.Yellow("Branch has %d unpushed commit(s).", unpushedCount)
				fmt.Println()
			}

			// Build prompt message
			var promptMsg string
			if hasChanges && unpushedCount > 0 {
				promptMsg = "Worktree has uncommitted changes and unpushed commits."
			} else if hasChanges {
				promptMsg = "Worktree has uncommitted changes."
			} else {
				promptMsg = "Worktree has unpushed commits."
			}

			// Build options based on what issues exist
			var options []Option
			if hasChanges && unpushedCount > 0 {
				// Both issues: offer combined options
				options = []Option{
					{Option: "f", Description: "Force remove (discard all)"},
					{Option: "b", Description: "Stash and push before removing"},
					{Option: "a", Description: "Stash all (reset branch, stash everything)"},
					{Option: "n", Description: "Cancel", Default: true},
				}
			} else if hasChanges {
				options = []Option{
					{Option: "f", Description: "Force remove (discard changes)"},
					{Option: "s", Description: "Stash changes before removing"},
					{Option: "n", Description: "Cancel", Default: true},
				}
			} else {
				// Only unpushed commits
				options = []Option{
					{Option: "f", Description: "Force remove"},
					{Option: "p", Description: "Push to remote first"},
					{Option: "a", Description: "Stash all (reset branch, stash commits)"},
					{Option: "n", Description: "Cancel", Default: true},
				}
			}

			choice, _, err := promptOption(promptMsg, options)
			if err != nil {
				return fmt.Errorf("failed to get user input: %w", err)
			}

			switch choice {
			case "n":
				fmt.Println("Cancelled.")
				return nil
			case "s":
				// Stash only (no unpushed commits)
				color.Blue("Stashing changes...")
				if err := stashChanges(wt.Path); err != nil {
					return fmt.Errorf("failed to stash changes: %w", err)
				}
				color.Green("✓ Changes stashed")
				changesStashed = true
			case "p":
				// Push only (no uncommitted changes)
				if err := doPush(wt.Path, branch); err != nil {
					return err
				}
			case "b":
				// Stash and push (both uncommitted changes and unpushed commits)
				color.Blue("Stashing changes...")
				if err := stashChanges(wt.Path); err != nil {
					return fmt.Errorf("failed to stash changes: %w", err)
				}
				color.Green("✓ Changes stashed")
				changesStashed = true

				if err := doPush(wt.Path, branch); err != nil {
					return err
				}
			case "a":
				// Stash all: mixed reset to base, then stage and stash everything
				color.Blue("Resetting branch to base...")
				if err := mixedResetToBase(wt.Path, branch, baseBranch); err != nil {
					return fmt.Errorf("failed to reset branch: %w", err)
				}
				color.Green("✓ Branch reset")

				color.Blue("Staging and stashing all changes...")
				if err := stashAll(wt.Path); err != nil {
					return fmt.Errorf("failed to stash changes: %w", err)
				}
				color.Green("✓ All changes stashed")
				changesStashed = true
				forceRemove = true // Need force delete since branch appears unmerged after reset
			case "f":
				forceRemove = true
			}
		}

		// Load config for remove actions
		config, err := loadConfig()
		if err != nil {
			color.Yellow("⚠ Failed to load config: %v", err)
		} else if len(config.Remove) > 0 {
			fmt.Println("Running remove actions...")
			if err := runActions(config.Remove, wt.Path, branch, baseBranch); err != nil {
				color.Yellow("⚠ Remove actions completed with errors")
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

		// Delete branch if worktree is clean, force is used, or changes were stashed
		if !hasChanges || forceRemove || changesStashed {
			color.Blue("Deleting branch '%s'...", branch)
			if err := deleteBranch(branch, forceRemove); err != nil {
				color.Yellow("⚠ Failed to delete branch: %v", err)
			} else {
				color.Green("✓ Branch deleted: %s", branch)
			}
		}

		return nil
	},
}

// doPush handles pushing to remote, prompting for branch name if needed
func doPush(path, branch string) error {
	if remoteBranchExists(branch) {
		color.Blue("Pushing to origin/%s...", branch)
		if err := pushToRemote(path, branch); err != nil {
			return fmt.Errorf("failed to push: %w", err)
		}
		color.Green("✓ Pushed to remote")
	} else {
		// Prompt for remote branch name with current branch as default
		remoteBranch, err := promptTextWithDefault("Remote branch name", branch)
		if err != nil {
			return fmt.Errorf("failed to get branch name: %w", err)
		}
		color.Blue("Pushing to origin/%s...", remoteBranch)
		if err := pushToNewRemote(path, branch, remoteBranch); err != nil {
			return fmt.Errorf("failed to push: %w", err)
		}
		color.Green("✓ Pushed to new remote branch: %s", remoteBranch)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().BoolVarP(&forceRemove, "force", "f", false, "Force removal even with uncommitted changes")
}
