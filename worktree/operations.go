package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CreateWorktree creates a new git worktree.
// If newBranch is true, creates a new branch from baseBranch.
// If newBranch is false, checks out an existing branch.
func CreateWorktree(branch, baseBranch, worktreePath string, newBranch bool) error {
	args := []string{"worktree", "add"}

	if newBranch {
		args = append(args, "-b", branch, worktreePath, baseBranch)
	} else {
		args = append(args, worktreePath, branch)
	}

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RemoveWorktree removes a git worktree.
// If force is true, removes even if there are uncommitted changes.
func RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// DeleteBranch deletes a local git branch.
// If force is true, uses -D (force delete), otherwise uses -d (safe delete).
func DeleteBranch(branch string, force bool) error {
	args := []string{"branch"}
	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, branch)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// EnsureWorktreeDir creates the .worktrees directory if it doesn't exist
// and adds it to .gitignore if not already present.
func EnsureWorktreeDir() (string, error) {
	worktreeDir, err := GetWorktreeDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create worktree directory: %w", err)
	}

	// Add to .gitignore if not already there
	gitRoot, _ := GetGitRoot()
	gitignorePath := filepath.Join(gitRoot, ".gitignore")

	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return worktreeDir, nil // Ignore errors reading .gitignore
	}

	if !strings.Contains(string(content), ".worktrees/") {
		f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			f.WriteString("\n# Git worktrees\n.worktrees/\n")
		}
	}

	return worktreeDir, nil
}

// StashChanges stashes all changes in the worktree (including untracked files)
func StashChanges(path string) error {
	args := []string{"-C", path, "stash", "push", "--include-untracked", "-m", "git-wt: stashed before remove"}
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// StashAll stages everything and stashes it (for use after reset)
func StashAll(path string) error {
	// First stage everything including untracked files
	addArgs := []string{"-C", path, "add", "-A"}
	cmd := exec.Command("git", addArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Now stash the staged changes
	stashArgs := []string{"-C", path, "stash", "push", "-m", "git-wt: stashed all before remove"}
	cmd = exec.Command("git", stashArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// MixedResetToBase resets the branch to the base, putting all changes in working directory (unstaged).
// It determines the reset target in this order:
// 1. Remote tracking branch (origin/branch)
// 2. Provided baseBranch
// 3. Default branch (main/master)
func MixedResetToBase(path, branch, baseBranch string) error {
	// Determine what to reset to: remote branch or base branch
	target := ""
	cmd := exec.Command("git", "-C", path, "rev-parse", "--verify", fmt.Sprintf("origin/%s", branch))
	if cmd.Run() == nil {
		target = fmt.Sprintf("origin/%s", branch)
	} else if baseBranch != "" {
		target = baseBranch
	} else {
		// Try to get default branch
		defaultBranch, err := GetDefaultBranch()
		if err != nil {
			return fmt.Errorf("cannot determine base to reset to: %w", err)
		}
		target = defaultBranch
	}

	args := []string{"-C", path, "reset", "--mixed", target}
	cmd = exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PushToRemote pushes the branch to its remote tracking branch
func PushToRemote(path, branch string) error {
	args := []string{"-C", path, "push", "origin", branch}
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PushToNewRemote pushes and sets upstream for a new remote branch
func PushToNewRemote(path, localBranch, remoteBranch string) error {
	args := []string{"-C", path, "push", "-u", "origin", fmt.Sprintf("%s:%s", localBranch, remoteBranch)}
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
