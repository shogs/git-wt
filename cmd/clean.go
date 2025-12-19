package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Interactive cleanup of merged branches",
	Long:  `Interactively removes worktrees for branches that have been merged.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureGitRepo(); err != nil {
			return err
		}

		// Get merged branches
		mergedBranches, err := getMergedBranches()
		if err != nil {
			return fmt.Errorf("failed to get merged branches: %w", err)
		}

		if len(mergedBranches) == 0 {
			color.Green("No merged branches found")
			return nil
		}

		// Get worktrees
		worktrees, err := getWorktrees()
		if err != nil {
			return err
		}

		// Build a map of worktree branches
		wtMap := make(map[string]*WorktreeInfo)
		for i := range worktrees {
			wtMap[worktrees[i].Branch] = &worktrees[i]
		}

		// Find merged worktrees
		var mergedWorktrees []string
		for _, branch := range mergedBranches {
			if _, exists := wtMap[branch]; exists {
				mergedWorktrees = append(mergedWorktrees, branch)
			}
		}

		if len(mergedWorktrees) == 0 {
			color.Green("No worktrees with merged branches found")
			return nil
		}

		// Display merged worktrees
		color.Cyan("Found %d merged worktree(s):\n", len(mergedWorktrees))
		for _, branch := range mergedWorktrees {
			fmt.Printf("  • %s\n", branch)
		}
		fmt.Println()

		// Confirm cleanup
		if !confirm(color.YellowString("Remove these worktrees?")) {
			color.Yellow("Cleanup cancelled")
			return nil
		}

		// Remove each worktree
		removed := 0
		for _, branch := range mergedWorktrees {
			wt := wtMap[branch]

			// Check if inside worktree
			inside, _ := isInsideWorktree(wt.Path)
			if inside {
				color.Yellow("⚠ Skipping %s (currently inside this worktree)", branch)
				continue
			}

			// Check for uncommitted changes
			hasChanges, err := hasUncommittedChanges(wt.Path)
			if err != nil {
				color.Yellow("⚠ Skipping %s (error checking changes: %v)", branch, err)
				continue
			}
			if hasChanges {
				color.Yellow("⚠ Skipping %s (uncommitted changes)", branch)
				continue
			}

			// Run remove actions
			config, err := loadConfig()
			if err == nil && len(config.Remove) > 0 {
				session, _ := loadSession(wt.Path)
				baseBranch := ""
				if session != nil {
					baseBranch = session.BaseBranch
				}
				runActions(config.Remove, wt.Path, branch, baseBranch)
			}

			// Remove session
			deleteSession(wt.Path)

			// Remove worktree
			if err := removeWorktree(wt.Path, false); err != nil {
				color.Red("✗ Failed to remove %s: %v", branch, err)
				continue
			}

			color.Green("✓ Removed %s", branch)
			removed++
		}

		fmt.Println()
		color.Green("Cleanup complete: removed %d worktree(s)", removed)
		return nil
	},
}

func getMergedBranches() ([]string, error) {
	// Get default branch
	defaultBranch, err := getDefaultBranch()
	if err != nil {
		defaultBranch = "main"
	}

	// Get merged branches
	cmd := exec.Command("git", "branch", "--merged", defaultBranch, "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	var merged []string

	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		// Skip empty, main, master, and current branch
		if branch == "" || branch == "main" || branch == "master" || branch == defaultBranch {
			continue
		}
		merged = append(merged, branch)
	}

	return merged, nil
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
