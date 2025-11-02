package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fatih/color"
)

// runActions executes a list of actions
func runActions(actions []Action, worktreePath, worktreeName, baseBranch string) error {
	if len(actions) == 0 {
		return nil
	}

	gitRoot, err := getGitRoot()
	if err != nil {
		return err
	}

	for _, action := range actions {
		color.Blue("→ %s", action.Description)

		cmd := exec.Command("bash", "-c", action.Script)
		cmd.Dir = worktreePath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Set environment variables
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("GIT_ROOT=%s", gitRoot),
			fmt.Sprintf("WORKTREE_PATH=%s", worktreePath),
			fmt.Sprintf("WORKTREE_NAME=%s", worktreeName),
			fmt.Sprintf("BASE_BRANCH=%s", baseBranch),
		)

		if err := cmd.Run(); err != nil {
			color.Red("✗ Failed to execute %s: %v", action.Name, err)
			return fmt.Errorf("action %s failed: %w", action.Name, err)
		}

		color.Green("✓ %s completed", action.Description)
	}

	return nil
}

// findWorktreeByBranch finds a worktree by branch name
func findWorktreeByBranch(branch string) (*WorktreeInfo, error) {
	worktrees, err := getWorktrees()
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if wt.Branch == branch {
			return &wt, nil
		}
	}

	return nil, nil
}

// isInsideWorktree checks if the current directory is inside a specific worktree
func isInsideWorktree(worktreePath string) (bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}

	// Get absolute paths
	absWorktree, err := filepath.Abs(worktreePath)
	if err != nil {
		return false, err
	}

	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return false, err
	}

	// Check if cwd is inside worktree
	rel, err := filepath.Rel(absWorktree, absCwd)
	if err != nil {
		return false, nil
	}

	// If rel doesn't start with "..", we're inside the worktree
	return rel[0] != '.', nil
}
