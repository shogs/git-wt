// Package worktree provides a library for managing Git worktrees programmatically.
// It can be imported by other Go projects to work with worktrees without any CLI output.
package worktree

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeInfo represents information about a git worktree
type WorktreeInfo struct {
	Path     string
	Branch   string
	Head     string
	Detached bool
}

// WorktreeStatus represents the status of a worktree
type WorktreeStatus struct {
	Modified  int
	Untracked int
	Staged    int
}

// AheadBehindCount represents commits ahead/behind remote
type AheadBehindCount struct {
	Ahead  int
	Behind int
}

// GetGitRoot returns the root directory of the git repository
func GetGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// GetCurrentBranch returns the current git branch name
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// GetWorktrees returns a list of all worktrees
func GetWorktrees() ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	var worktrees []WorktreeInfo
	lines := strings.Split(out.String(), "\n")
	var current WorktreeInfo

	for _, line := range lines {
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		switch parts[0] {
		case "worktree":
			current.Path = parts[1]
		case "HEAD":
			current.Head = parts[1]
		case "branch":
			// Format: "branch refs/heads/branch-name"
			current.Branch = strings.TrimPrefix(parts[1], "refs/heads/")
		case "detached":
			current.Detached = true
		}
	}

	// Add last worktree if exists
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// GetDefaultBranch returns the default branch (main or master)
func GetDefaultBranch() (string, error) {
	// Try to get from remote
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		branch := strings.TrimSpace(out.String())
		branch = strings.TrimPrefix(branch, "refs/remotes/origin/")
		if branch != "" {
			return branch, nil
		}
	}

	// Fallback: check if main exists, otherwise master
	if BranchExists("main") {
		return "main", nil
	}
	if BranchExists("master") {
		return "master", nil
	}

	return "", fmt.Errorf("could not determine default branch")
}

// GetWorktreeDir returns the .worktrees directory path
func GetWorktreeDir() (string, error) {
	gitRoot, err := GetGitRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(gitRoot, ".worktrees"), nil
}

// BranchExists checks if a branch exists
func BranchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", fmt.Sprintf("refs/heads/%s", branch))
	err := cmd.Run()
	return err == nil
}

// RemoteBranchExists checks if a remote branch exists
func RemoteBranchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", fmt.Sprintf("refs/remotes/origin/%s", branch))
	err := cmd.Run()
	return err == nil
}

// HasUncommittedChanges checks if the worktree has uncommitted changes
func HasUncommittedChanges(path string) (bool, error) {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false, err
	}
	return out.Len() > 0, nil
}

// GetGitStatus returns the git status output for display
func GetGitStatus(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "status", "--short")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

// GetWorktreeStatus returns the status of a worktree (modified, untracked, staged files)
func GetWorktreeStatus(path string) (*WorktreeStatus, error) {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain=v1")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	status := &WorktreeStatus{}
	lines := strings.Split(out.String(), "\n")

	for _, line := range lines {
		if len(line) < 2 {
			continue
		}

		// Porcelain format: XY filename
		// X = index status, Y = working tree status
		x := line[0]
		y := line[1]

		// Staged changes (index has changes)
		if x == 'M' || x == 'A' || x == 'D' || x == 'R' || x == 'C' {
			status.Staged++
		}

		// Modified in working tree
		if y == 'M' || y == 'D' {
			status.Modified++
		}

		// Untracked files
		if x == '?' && y == '?' {
			status.Untracked++
		}
	}

	return status, nil
}

// GetAheadBehindCount returns how many commits the branch is ahead/behind its remote tracking branch
func GetAheadBehindCount(path, branch string) (*AheadBehindCount, error) {
	// Check if remote tracking branch exists
	cmd := exec.Command("git", "-C", path, "rev-parse", "--verify", fmt.Sprintf("origin/%s", branch))
	if err := cmd.Run(); err != nil {
		// No remote tracking branch
		return nil, nil
	}

	// Get ahead/behind count
	cmd = exec.Command("git", "-C", path, "rev-list", "--left-right", "--count", fmt.Sprintf("%s...origin/%s", branch, branch))
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	// Output format: "ahead\tbehind\n"
	parts := strings.Fields(strings.TrimSpace(out.String()))
	if len(parts) != 2 {
		return nil, nil
	}

	var count AheadBehindCount
	fmt.Sscanf(parts[0], "%d", &count.Ahead)
	fmt.Sscanf(parts[1], "%d", &count.Behind)

	return &count, nil
}

// GetUnpushedCommitCount returns the number of commits not on remote
// For branches with a remote tracking branch: counts commits ahead
// For new branches without remote: counts commits since base branch
func GetUnpushedCommitCount(path, branch, baseBranch string) (int, error) {
	// First try: check if remote tracking branch exists
	cmd := exec.Command("git", "-C", path, "rev-parse", "--verify", fmt.Sprintf("origin/%s", branch))
	if cmd.Run() == nil {
		// Has remote - count commits ahead
		cmd = exec.Command("git", "-C", path, "rev-list", "--count", fmt.Sprintf("origin/%s..%s", branch, branch))
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return 0, err
		}
		var count int
		fmt.Sscanf(strings.TrimSpace(out.String()), "%d", &count)
		return count, nil
	}

	// No remote - count commits since base branch
	if baseBranch == "" {
		baseBranch, _ = GetDefaultBranch()
	}
	if baseBranch == "" {
		return 0, nil // Can't determine, assume 0
	}

	cmd = exec.Command("git", "-C", path, "rev-list", "--count", fmt.Sprintf("%s..%s", baseBranch, branch))
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, err
	}
	var count int
	fmt.Sscanf(strings.TrimSpace(out.String()), "%d", &count)
	return count, nil
}
