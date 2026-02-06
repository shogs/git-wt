package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

// echoGitCmd prints the git command that will be executed
func echoGitCmd(args ...string) {
	color.New(color.Faint).Printf("$ git %s\n", strings.Join(args, " "))
}

// getGitRoot returns the root directory of the git repository
func getGitRoot() (string, error) {
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

// getCurrentBranch returns the current git branch name
func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// getWorktrees returns a list of all worktrees
func getWorktrees() ([]WorktreeInfo, error) {
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

// WorktreeInfo represents information about a git worktree
type WorktreeInfo struct {
	Path     string
	Branch   string
	Head     string
	Detached bool
}

// branchExists checks if a branch exists
func branchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", fmt.Sprintf("refs/heads/%s", branch))
	err := cmd.Run()
	return err == nil
}

// remoteBranchExists checks if a remote branch exists
func remoteBranchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", fmt.Sprintf("refs/remotes/origin/%s", branch))
	err := cmd.Run()
	return err == nil
}

// hasUncommittedChanges checks if the worktree has uncommitted changes
func hasUncommittedChanges(path string) (bool, error) {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false, err
	}
	return out.Len() > 0, nil
}

// getDefaultBranch returns the default branch (main or master)
func getDefaultBranch() (string, error) {
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
	if branchExists("main") {
		return "main", nil
	}
	if branchExists("master") {
		return "master", nil
	}

	return "", fmt.Errorf("could not determine default branch")
}

// createWorktree creates a new git worktree
func createWorktree(branch, baseBranch, worktreePath string, newBranch bool) error {
	args := []string{"worktree", "add"}

	if newBranch {
		args = append(args, "-b", branch, worktreePath, baseBranch)
	} else {
		args = append(args, worktreePath, branch)
	}

	echoGitCmd(args...)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// removeWorktree removes a git worktree
func removeWorktree(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)

	echoGitCmd(args...)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// deleteBranch deletes a local git branch
func deleteBranch(branch string, force bool) error {
	args := []string{"branch"}
	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, branch)

	echoGitCmd(args...)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// getWorktreeDir returns the .worktrees directory path
func getWorktreeDir() (string, error) {
	gitRoot, err := getGitRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(gitRoot, ".worktrees"), nil
}

// ensureWorktreeDir creates the .worktrees directory if it doesn't exist
func ensureWorktreeDir() (string, error) {
	worktreeDir, err := getWorktreeDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create worktree directory: %w", err)
	}

	// Add to .gitignore if not already there
	gitRoot, _ := getGitRoot()
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

// getGitStatus returns the git status output for display
func getGitStatus(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "status", "--short")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

// stashChanges stashes all changes in the worktree (including untracked files)
func stashChanges(path string) error {
	args := []string{"-C", path, "stash", "push", "--include-untracked", "-m", "git-wt: stashed before remove"}
	echoGitCmd(args...)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// stashAll stages everything and stashes it (for use after reset)
func stashAll(path string) error {
	// First stage everything including untracked files
	addArgs := []string{"-C", path, "add", "-A"}
	echoGitCmd(addArgs...)
	cmd := exec.Command("git", addArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Now stash the staged changes
	stashArgs := []string{"-C", path, "stash", "push", "-m", "git-wt: stashed all before remove"}
	echoGitCmd(stashArgs...)
	cmd = exec.Command("git", stashArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// mixedResetToBase resets the branch to the base, putting all changes in working directory (unstaged)
func mixedResetToBase(path, branch, baseBranch string) error {
	// Determine what to reset to: remote branch or base branch
	target := ""
	cmd := exec.Command("git", "-C", path, "rev-parse", "--verify", fmt.Sprintf("origin/%s", branch))
	if cmd.Run() == nil {
		target = fmt.Sprintf("origin/%s", branch)
	} else if baseBranch != "" {
		target = baseBranch
	} else {
		// Try to get default branch
		defaultBranch, err := getDefaultBranch()
		if err != nil {
			return fmt.Errorf("cannot determine base to reset to: %w", err)
		}
		target = defaultBranch
	}

	args := []string{"-C", path, "reset", "--mixed", target}
	echoGitCmd(args...)
	cmd = exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// pushToRemote pushes the branch to its remote tracking branch
func pushToRemote(path, branch string) error {
	args := []string{"-C", path, "push", "origin", branch}
	echoGitCmd(args...)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// pushToNewRemote pushes and sets upstream for a new remote branch
func pushToNewRemote(path, localBranch, remoteBranch string) error {
	args := []string{"-C", path, "push", "-u", "origin", fmt.Sprintf("%s:%s", localBranch, remoteBranch)}
	echoGitCmd(args...)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// WorktreeStatus represents the status of a worktree
type WorktreeStatus struct {
	Modified  int
	Untracked int
	Staged    int
}

// getWorktreeStatus returns the status of a worktree (modified, untracked, staged files)
func getWorktreeStatus(path string) (*WorktreeStatus, error) {
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

// AheadBehindCount represents commits ahead/behind remote
type AheadBehindCount struct {
	Ahead  int
	Behind int
}

// getAheadBehindCount returns how many commits the branch is ahead/behind its remote tracking branch
func getAheadBehindCount(path, branch string) (*AheadBehindCount, error) {
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

// getUnpushedCommitCount returns the number of commits not on remote
// For branches with a remote tracking branch: counts commits ahead
// For new branches without remote: counts commits since base branch
func getUnpushedCommitCount(path, branch, baseBranch string) (int, error) {
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
		baseBranch, _ = getDefaultBranch()
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
