package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:     "new [branch-name] [base-branch]",
	Aliases: []string{"add"},
	Short:   "Create a new worktree",
	Long:    `Creates a new git worktree with optional new actions from .git-wt.yaml.`,
	Args:    cobra.RangeArgs(0, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureGitRepo(); err != nil {
			return err
		}

		var branchName, baseBranch string

		// Get branch name
		if len(args) > 0 {
			branchName = args[0]
		} else {
			fmt.Print("Enter branch name: ")
			fmt.Scanln(&branchName)
		}

		if branchName == "" {
			return fmt.Errorf("branch name is required")
		}

		// Get base branch
		if len(args) > 1 {
			baseBranch = args[1]
		} else {
			var err error
			baseBranch, err = getDefaultBranch()
			if err != nil {
				baseBranch = "main" // fallback
			}
		}

		// Check if branch already exists
		if branchExists(branchName) {
			return fmt.Errorf("branch '%s' already exists", branchName)
		}

		// Check if worktree already exists
		existingWt, err := findWorktreeByBranch(branchName)
		if err != nil {
			return err
		}
		if existingWt != nil {
			return fmt.Errorf("worktree for branch '%s' already exists at %s", branchName, existingWt.Path)
		}

		// Ensure .worktrees directory exists
		worktreeDir, err := ensureWorktreeDir()
		if err != nil {
			return err
		}

		worktreePath := filepath.Join(worktreeDir, branchName)

		// Create worktree
		color.Blue("Creating worktree for branch '%s' from '%s'...", branchName, baseBranch)
		if err := createWorktree(branchName, baseBranch, worktreePath, true); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}

		color.Green("✓ Worktree created at %s", worktreePath)

		// Create session file
		session := &Session{
			Branch:     branchName,
			Created:    time.Now(),
			BaseBranch: baseBranch,
		}
		if err := saveSession(worktreePath, session); err != nil {
			color.Yellow("⚠ Failed to create session file: %v", err)
		}

		// Run new actions
		config, err := loadConfig()
		if err != nil {
			color.Yellow("⚠ Failed to load config: %v", err)
		} else if len(config.New) > 0 {
			fmt.Println("\nRunning new actions...")
			if err := runActions(config.New, worktreePath, branchName, baseBranch); err != nil {
				color.Yellow("⚠ New actions completed with errors")
			}
		}

		fmt.Println()
		color.Green("✓ Ready to work in %s", worktreePath)

		// Ask if user wants to switch to the new worktree
		fmt.Println()
		if confirmWithDefault("Switch to new worktree?", true) {
			return spawnWorktreeShell(worktreePath, branchName)
		}

		fmt.Println("\nTo switch to this worktree later:")
		color.Cyan("  git-wt switch %s", branchName)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
}
