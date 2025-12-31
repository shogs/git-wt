package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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

// WorktreeChanges holds counts of different change types
type WorktreeChanges struct {
	Staged    int
	Unstaged  int
	Untracked int
}

// getWorktreeChangeCounts returns counts of staged, unstaged, and untracked files
func getWorktreeChangeCounts(path string) (*WorktreeChanges, error) {
	status, err := getGitStatus(path)
	if err != nil {
		return nil, err
	}

	changes := &WorktreeChanges{}
	lines := strings.Split(status, "\n")

	for _, line := range lines {
		if len(line) < 2 {
			continue
		}

		// First char = staged status, second char = unstaged status
		staged := line[0]
		unstaged := line[1]

		// Untracked files: "??"
		if staged == '?' && unstaged == '?' {
			changes.Untracked++
			continue
		}

		// Staged changes: first char is not space or ?
		if staged != ' ' && staged != '?' {
			changes.Staged++
		}

		// Unstaged changes: second char is not space or ?
		if unstaged != ' ' && unstaged != '?' {
			changes.Unstaged++
		}
	}

	return changes, nil
}

// stashChanges stashes all changes in the worktree (including untracked files)
func stashChanges(path string) error {
	cmd := exec.Command("git", "-C", path, "stash", "push", "--include-untracked", "-m", "git-wt: stashed before remove")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// stashAll stages everything and stashes it (for use after reset)
func stashAll(path string) error {
	// First stage everything including untracked files
	cmd := exec.Command("git", "-C", path, "add", "-A")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Now stash the staged changes
	cmd = exec.Command("git", "-C", path, "stash", "push", "-m", "git-wt: stashed all before remove")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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

	// Use --mixed (default) to put all changes in working directory
	cmd = exec.Command("git", "-C", path, "reset", "--mixed", target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// pushToRemote pushes the branch to its remote tracking branch
func pushToRemote(path, branch string) error {
	cmd := exec.Command("git", "-C", path, "push", "origin", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// pushToNewRemote pushes and sets upstream for a new remote branch
func pushToNewRemote(path, localBranch, remoteBranch string) error {
	cmd := exec.Command("git", "-C", path, "push", "-u", "origin", fmt.Sprintf("%s:%s", localBranch, remoteBranch))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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

// ChangedFile represents a file that has been modified, added, or deleted
type ChangedFile struct {
	Path    string // File path relative to repo root
	Status  string // M=Modified, A=Added, D=Deleted, R=Renamed, C=Copied
	OldPath string // For renames/copies, the original path
}

// getChangedFiles returns files changed between baseRef and HEAD (or working tree)
// If staged is true, shows only staged changes
// If unstaged is true, shows only unstaged changes
// If both are false, compares baseRef...HEAD for committed changes
func getChangedFiles(path, baseRef string, staged, unstaged bool) ([]ChangedFile, error) {
	var args []string

	if staged {
		// Show staged changes
		args = []string{"-C", path, "diff", "--name-status", "--staged"}
	} else if unstaged {
		// Show unstaged changes (working tree vs index)
		args = []string{"-C", path, "diff", "--name-status"}
	} else {
		// Show changes between base and HEAD
		args = []string{"-C", path, "diff", "--name-status", fmt.Sprintf("%s...HEAD", baseRef)}
	}

	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git diff failed: %s", stderr.String())
	}

	files := parseChangedFiles(out.String())

	// For unstaged mode, also include untracked files
	if unstaged {
		untrackedFiles, err := getUntrackedFiles(path)
		if err == nil {
			files = append(files, untrackedFiles...)
		}
	}

	return files, nil
}

// parseChangedFiles parses git diff --name-status output
func parseChangedFiles(output string) []ChangedFile {
	var files []ChangedFile
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: STATUS\tPATH or STATUS\tOLDPATH\tNEWPATH (for renames)
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		file := ChangedFile{
			Status: string(status[0]), // First char is the status (R100 -> R)
		}

		if status[0] == 'R' || status[0] == 'C' {
			// Rename or copy: has old and new path
			if len(parts) >= 3 {
				file.OldPath = parts[1]
				file.Path = parts[2]
			}
		} else {
			file.Path = parts[1]
		}

		files = append(files, file)
	}

	return files
}

// getUntrackedFiles returns untracked files in the repository
func getUntrackedFiles(path string) ([]ChangedFile, error) {
	cmd := exec.Command("git", "-C", path, "ls-files", "--others", "--exclude-standard")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git ls-files failed: %s", stderr.String())
	}

	var files []ChangedFile
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		files = append(files, ChangedFile{
			Path:   line,
			Status: "?", // Untracked
		})
	}

	return files, nil
}

// getFileDiff returns the unified diff for a specific file
// If staged is true, shows staged diff
// If unstaged is true, shows unstaged diff
// Otherwise shows diff between baseRef and HEAD
func getFileDiff(path, baseRef, filePath string, staged, unstaged bool) (string, error) {
	var args []string

	if staged {
		args = []string{"-C", path, "diff", "--staged", "--", filePath}
	} else if unstaged {
		args = []string{"-C", path, "diff", "--", filePath}
	} else {
		args = []string{"-C", path, "diff", fmt.Sprintf("%s...HEAD", baseRef), "--", filePath}
	}

	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git diff failed: %s", stderr.String())
	}

	return out.String(), nil
}

// getFileContent returns the content of a file at a specific ref
// Use "HEAD" for current commit, or a branch/commit name
func getFileContent(path, ref, filePath string) (string, error) {
	cmd := exec.Command("git", "-C", path, "show", fmt.Sprintf("%s:%s", ref, filePath))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git show failed: %s", stderr.String())
	}

	return out.String(), nil
}

// getWorkingTreeFileContent returns the content of a file from the working tree
func getWorkingTreeFileContent(repoPath, filePath string) (string, error) {
	fullPath := filepath.Join(repoPath, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// getStagedFileContent returns the content of a file from the staging area (index)
func getStagedFileContent(repoPath, filePath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "show", fmt.Sprintf(":%s", filePath))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git show failed: %s", stderr.String())
	}

	return out.String(), nil
}

// stageFile stages a file using git add
func stageFile(repoPath, filePath string) error {
	cmd := exec.Command("git", "-C", repoPath, "add", "--", filePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %s", stderr.String())
	}
	return nil
}

// unstageFile unstages a file using git restore --staged
func unstageFile(repoPath, filePath string) error {
	cmd := exec.Command("git", "-C", repoPath, "restore", "--staged", "--", filePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git restore --staged failed: %s", stderr.String())
	}
	return nil
}

// extractHunkPatch extracts a single hunk from a unified diff as a valid patch
// hunkIndex is 0-based
func extractHunkPatch(rawDiff string, hunkIndex int) (string, error) {
	lines := strings.Split(rawDiff, "\n")
	var header []string
	var hunks [][]string
	var currentHunk []string
	inHunk := false

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") ||
			strings.HasPrefix(line, "old mode") ||
			strings.HasPrefix(line, "new mode") ||
			strings.HasPrefix(line, "new file mode") ||
			strings.HasPrefix(line, "deleted file mode") ||
			strings.HasPrefix(line, "similarity index") ||
			strings.HasPrefix(line, "rename from") ||
			strings.HasPrefix(line, "rename to") {
			header = append(header, line)
		} else if strings.HasPrefix(line, "@@") {
			if inHunk && len(currentHunk) > 0 {
				hunks = append(hunks, currentHunk)
			}
			currentHunk = []string{line}
			inHunk = true
		} else if inHunk {
			currentHunk = append(currentHunk, line)
		}
	}
	// Don't forget the last hunk
	if inHunk && len(currentHunk) > 0 {
		hunks = append(hunks, currentHunk)
	}

	if hunkIndex < 0 || hunkIndex >= len(hunks) {
		return "", fmt.Errorf("hunk index %d out of range (have %d hunks)", hunkIndex, len(hunks))
	}

	// Build patch with header + single hunk
	var patch strings.Builder
	for _, h := range header {
		patch.WriteString(h)
		patch.WriteString("\n")
	}
	for _, h := range hunks[hunkIndex] {
		patch.WriteString(h)
		patch.WriteString("\n")
	}

	return patch.String(), nil
}

// stageHunk stages a single hunk using git apply --cached
func stageHunk(repoPath, patch string) error {
	cmd := exec.Command("git", "-C", repoPath, "apply", "--cached", "--unidiff-zero")
	cmd.Stdin = strings.NewReader(patch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git apply --cached failed: %s", stderr.String())
	}
	return nil
}

// unstageHunk unstages a single hunk using git apply --cached -R
func unstageHunk(repoPath, patch string) error {
	cmd := exec.Command("git", "-C", repoPath, "apply", "--cached", "-R", "--unidiff-zero")
	cmd.Stdin = strings.NewReader(patch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git apply --cached -R failed: %s", stderr.String())
	}
	return nil
}

// --- Preview command helper functions ---

// getGitVersion returns the Git version as major, minor, patch components
func getGitVersion() (major, minor, patch int, err error) {
	cmd := exec.Command("git", "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, 0, 0, err
	}
	// Output: "git version 2.38.1" or "git version 2.38.1.windows.1"
	output := strings.TrimSpace(out.String())
	parts := strings.Fields(output)
	if len(parts) < 3 {
		return 0, 0, 0, fmt.Errorf("unexpected git version format: %s", output)
	}
	version := parts[2]
	// Handle versions like "2.38.1.windows.1"
	versionParts := strings.Split(version, ".")
	if len(versionParts) >= 1 {
		fmt.Sscanf(versionParts[0], "%d", &major)
	}
	if len(versionParts) >= 2 {
		fmt.Sscanf(versionParts[1], "%d", &minor)
	}
	if len(versionParts) >= 3 {
		fmt.Sscanf(versionParts[2], "%d", &patch)
	}
	return major, minor, patch, nil
}

// getCommitCountBetween returns the number of commits in branch that are not in baseBranch
func getCommitCountBetween(baseBranch, branch string) (int, error) {
	cmd := exec.Command("git", "rev-list", "--count", fmt.Sprintf("%s..%s", baseBranch, branch))
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, err
	}
	var count int
	fmt.Sscanf(strings.TrimSpace(out.String()), "%d", &count)
	return count, nil
}

// getBranchHead returns the commit SHA for a branch
func getBranchHead(branch string) (string, error) {
	cmd := exec.Command("git", "rev-parse", branch)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// MergeConflictResult represents the result of a merge conflict check
type MergeConflictResult struct {
	HasConflicts     bool
	ConflictingFiles []string
	TreeOID          string
}

// checkMergeConflicts uses git merge-tree --write-tree to detect conflicts without modifying the working directory
// Requires Git 2.38+
// Checks each branch against the base, and each pair of branches against each other
func checkMergeConflicts(baseBranch string, branches []string) (*MergeConflictResult, error) {
	if len(branches) == 0 {
		return &MergeConflictResult{HasConflicts: false}, nil
	}

	result := &MergeConflictResult{}

	// Check each branch against the base branch
	for _, branch := range branches {
		conflicts, err := checkPairMergeConflict(baseBranch, branch)
		if err != nil {
			// Ignore errors for identical branches
			continue
		}
		if len(conflicts) > 0 {
			result.HasConflicts = true
			result.ConflictingFiles = append(result.ConflictingFiles, conflicts...)
		}
	}

	// Check each pair of branches against each other
	for i := 0; i < len(branches); i++ {
		for j := i + 1; j < len(branches); j++ {
			conflicts, err := checkPairMergeConflict(branches[i], branches[j])
			if err != nil {
				continue
			}
			if len(conflicts) > 0 {
				result.HasConflicts = true
				result.ConflictingFiles = append(result.ConflictingFiles, conflicts...)
			}
		}
	}

	// Deduplicate conflict files
	if len(result.ConflictingFiles) > 0 {
		seen := make(map[string]bool)
		var unique []string
		for _, f := range result.ConflictingFiles {
			if !seen[f] {
				seen[f] = true
				unique = append(unique, f)
			}
		}
		result.ConflictingFiles = unique
	}

	return result, nil
}

// checkPairMergeConflict checks for conflicts between two branches
func checkPairMergeConflict(branch1, branch2 string) ([]string, error) {
	args := []string{"merge-tree", "--write-tree", branch1, branch2}
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()

	if err != nil {
		// Exit code 1 means conflicts
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			conflicts := parseMergeTreeConflicts(output)
			return conflicts, nil
		}
		return nil, fmt.Errorf("merge-tree failed: %s", stderr.String())
	}

	return nil, nil
}

// parseMergeTreeConflicts parses git merge-tree output to extract conflicting file paths
func parseMergeTreeConflicts(output string) []string {
	var conflicts []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Look for conflict markers - format varies by Git version
		// Common patterns:
		// "CONFLICT (content): Merge conflict in path/to/file"
		// "CONFLICT (add/add): Merge conflict in path/to/file"
		if strings.HasPrefix(line, "CONFLICT") {
			if idx := strings.Index(line, " in "); idx != -1 {
				path := strings.TrimSpace(line[idx+4:])
				conflicts = append(conflicts, path)
			}
		}
	}
	return conflicts
}

// createOrResetBranch creates a new branch or resets an existing branch to the given starting point
func createOrResetBranch(branchName, startPoint string) error {
	cmd := exec.Command("git", "checkout", "-B", branchName, startPoint)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create/reset branch: %s", stderr.String())
	}
	return nil
}

// mergeBranch merges a branch into the current HEAD
func mergeBranch(branch string, noFF bool) error {
	args := []string{"merge"}
	if noFF {
		args = append(args, "--no-ff")
	}
	args = append(args, "-m", fmt.Sprintf("Merge %s into preview", branch), branch)

	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("merge failed: %s", stderr.String())
	}
	return nil
}

// checkoutBranch switches to an existing branch
func checkoutBranch(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("checkout failed: %s", stderr.String())
	}
	return nil
}

// --- WIP (Work-in-Progress) helper functions ---

// UntrackedFile represents an untracked file with its content
type UntrackedFile struct {
	Path    string
	Content []byte
}

// WIPChanges represents uncommitted changes from a worktree
type WIPChanges struct {
	WorktreePath string
	Branch       string
	Diff         string          // Combined staged+unstaged diff
	Untracked    []UntrackedFile // Untracked files with content
	HasChanges   bool
}

// getWorktreeDiff returns the combined diff of staged and unstaged changes in a worktree
func getWorktreeDiff(worktreePath string) (string, error) {
	// git diff HEAD shows all changes (staged + unstaged) compared to HEAD
	cmd := exec.Command("git", "-C", worktreePath, "diff", "HEAD")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git diff HEAD failed: %s", stderr.String())
	}
	return out.String(), nil
}

// getStagedDiff returns only staged changes
func getStagedDiff(worktreePath string) (string, error) {
	cmd := exec.Command("git", "-C", worktreePath, "diff", "--cached")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git diff --cached failed: %s", stderr.String())
	}
	return out.String(), nil
}

// getUnstagedDiff returns only unstaged changes (excludes staged)
func getUnstagedDiff(worktreePath string) (string, error) {
	cmd := exec.Command("git", "-C", worktreePath, "diff")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git diff failed: %s", stderr.String())
	}
	return out.String(), nil
}

// getUntrackedFilesWithContent returns untracked files with their content
func getUntrackedFilesWithContent(worktreePath string) ([]UntrackedFile, error) {
	// Get list of untracked files
	cmd := exec.Command("git", "-C", worktreePath, "ls-files", "--others", "--exclude-standard")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git ls-files failed: %s", stderr.String())
	}

	var files []UntrackedFile
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Read file content
		fullPath := filepath.Join(worktreePath, line)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			// Skip files we can't read (might be binary or permissions issue)
			continue
		}

		// Skip binary files (simple heuristic: check for null bytes)
		if isBinaryContent(content) {
			continue
		}

		files = append(files, UntrackedFile{
			Path:    line,
			Content: content,
		})
	}

	return files, nil
}

// isBinaryContent checks if content appears to be binary (contains null bytes)
func isBinaryContent(content []byte) bool {
	// Check first 8KB for null bytes
	checkLen := len(content)
	if checkLen > 8192 {
		checkLen = 8192
	}
	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

// applyPatch applies a patch to the current working directory
func applyPatch(patch string) error {
	if patch == "" {
		return nil
	}
	cmd := exec.Command("git", "apply", "--3way")
	cmd.Stdin = strings.NewReader(patch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git apply failed: %s", stderr.String())
	}
	return nil
}

// addFileContent writes a file and stages it
func addFileContent(repoPath, filePath string, content []byte) error {
	fullPath := filepath.Join(repoPath, filePath)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Stage file
	cmd := exec.Command("git", "-C", repoPath, "add", "--", filePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %s", stderr.String())
	}

	return nil
}

// --- Preview worktree helper functions ---

// createPreviewWorktree creates or resets a worktree for the preview branch
// Returns the worktree path
func createPreviewWorktree(gitRoot, branchName, baseBranch string) (string, error) {
	worktreePath := filepath.Join(gitRoot, ".worktrees", branchName)

	// Check if worktree already exists
	worktrees, err := getWorktrees()
	if err != nil {
		return "", fmt.Errorf("failed to list worktrees: %w", err)
	}

	var existingWorktree *WorktreeInfo
	for _, wt := range worktrees {
		if wt.Path == worktreePath {
			existingWorktree = &wt
			break
		}
	}

	if existingWorktree != nil {
		// Worktree exists - fully clean it (reset + remove untracked)
		if err := cleanWorktree(worktreePath, baseBranch); err != nil {
			return "", fmt.Errorf("failed to clean existing worktree: %w", err)
		}
		return worktreePath, nil
	}

	// Check if branch exists
	branchExists := false
	cmd := exec.Command("git", "rev-parse", "--verify", branchName)
	if cmd.Run() == nil {
		branchExists = true
	}

	// Create worktree
	if branchExists {
		// Use existing branch
		cmd = exec.Command("git", "worktree", "add", worktreePath, branchName)
	} else {
		// Create new branch from base
		cmd = exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, baseBranch)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create worktree: %s", stderr.String())
	}

	// If branch existed, fully clean it to base (reset + remove untracked)
	if branchExists {
		if err := cleanWorktree(worktreePath, baseBranch); err != nil {
			return "", fmt.Errorf("failed to clean worktree to base: %w", err)
		}
	}

	return worktreePath, nil
}

// mergeBranchInWorktree merges a branch into HEAD within a worktree
func mergeBranchInWorktree(worktreePath, branch string, noFF bool) error {
	args := []string{"-C", worktreePath, "merge"}
	if noFF {
		args = append(args, "--no-ff")
	}
	args = append(args, "-m", fmt.Sprintf("Merge %s into preview", branch), branch)

	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("merge failed: %s", stderr.String())
	}
	return nil
}

// applyPatchInWorktree applies a patch within a worktree (leaves changes unstaged)
// Note: We don't use --3way here because it implies --index which would stage the changes
func applyPatchInWorktree(worktreePath, patch string) error {
	if patch == "" {
		return nil
	}
	cmd := exec.Command("git", "-C", worktreePath, "apply")
	cmd.Stdin = strings.NewReader(patch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git apply failed: %s", stderr.String())
	}
	return nil
}

// applyPatchAndStageInWorktree applies a patch to both index and working tree (staged)
func applyPatchAndStageInWorktree(worktreePath, patch string) error {
	if patch == "" {
		return nil
	}
	// Apply the patch with --index to apply to both index and working tree
	// This stages only the files from this patch, not everything
	cmd := exec.Command("git", "-C", worktreePath, "apply", "--index", "--3way")
	cmd.Stdin = strings.NewReader(patch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git apply --index failed: %s", stderr.String())
	}
	return nil
}

// resetWorktreeToBase resets a worktree to the base branch
func resetWorktreeToBase(worktreePath, baseBranch string) error {
	cmd := exec.Command("git", "-C", worktreePath, "reset", "--hard", baseBranch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reset worktree: %s", stderr.String())
	}
	return nil
}

// cleanWorktree removes all untracked files and directories, and resets staged/unstaged changes
func cleanWorktree(worktreePath, baseBranch string) error {
	// First, reset any staged/unstaged changes
	cmd := exec.Command("git", "-C", worktreePath, "reset", "--hard", baseBranch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reset worktree: %s", stderr.String())
	}

	// Then clean untracked files and directories
	cmd = exec.Command("git", "-C", worktreePath, "clean", "-fd")
	stderr.Reset()
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clean worktree: %s", stderr.String())
	}

	return nil
}
