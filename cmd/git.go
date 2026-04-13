package cmd

import (
	"strings"

	"github.com/fatih/color"
	"github.com/shogs/git-wt/worktree"
)

// Type aliases for backward compatibility - no changes needed in other cmd/*.go files
type WorktreeInfo = worktree.WorktreeInfo
type WorktreeStatus = worktree.WorktreeStatus
type AheadBehindCount = worktree.AheadBehindCount

// echoGitCmd prints the git command that will be executed (CLI-specific)
func echoGitCmd(args ...string) {
	color.New(color.Faint).Printf("$ git %s\n", strings.Join(args, " "))
}

// Pure query functions - direct delegation to library

func getGitRoot() (string, error) {
	return worktree.GetGitRoot()
}

func getCurrentBranch() (string, error) {
	return worktree.GetCurrentBranch()
}

func getWorktrees() ([]WorktreeInfo, error) {
	return worktree.GetWorktrees()
}

func getDefaultBranch() (string, error) {
	return worktree.GetDefaultBranch()
}

func getWorktreeDir() (string, error) {
	return worktree.GetWorktreeDir()
}

func branchExists(branch string) bool {
	return worktree.BranchExists(branch)
}

func remoteBranchExists(branch string) bool {
	return worktree.RemoteBranchExists(branch)
}

func hasUncommittedChanges(path string) (bool, error) {
	return worktree.HasUncommittedChanges(path)
}

func getGitStatus(path string) (string, error) {
	return worktree.GetGitStatus(path)
}

func getWorktreeStatus(path string) (*WorktreeStatus, error) {
	return worktree.GetWorktreeStatus(path)
}

func getAheadBehindCount(path, branch string) (*AheadBehindCount, error) {
	return worktree.GetAheadBehindCount(path, branch)
}

func getUnpushedCommitCount(path, branch, baseBranch string) (int, error) {
	return worktree.GetUnpushedCommitCount(path, branch, baseBranch)
}

// State-changing operations - wrappers that echo command then call library

func createWorktree(branch, baseBranch, worktreePath string, newBranch bool) error {
	args := []string{"worktree", "add"}
	if newBranch {
		args = append(args, "-b", branch, worktreePath, baseBranch)
	} else {
		args = append(args, worktreePath, branch)
	}
	echoGitCmd(args...)
	return worktree.CreateWorktree(branch, baseBranch, worktreePath, newBranch)
}

func removeWorktree(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	echoGitCmd(args...)
	return worktree.RemoveWorktree(path, force)
}

func deleteBranch(branch string, force bool) error {
	args := []string{"branch"}
	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, branch)
	echoGitCmd(args...)
	return worktree.DeleteBranch(branch, force)
}

func ensureWorktreeDir() (string, error) {
	return worktree.EnsureWorktreeDir()
}

func stashChanges(path string) error {
	args := []string{"-C", path, "stash", "push", "--include-untracked", "-m", "git-wt: stashed before remove"}
	echoGitCmd(args...)
	return worktree.StashChanges(path)
}

func stashAll(path string) error {
	// Echo both commands that will be run
	addArgs := []string{"-C", path, "add", "-A"}
	echoGitCmd(addArgs...)
	stashArgs := []string{"-C", path, "stash", "push", "-m", "git-wt: stashed all before remove"}
	echoGitCmd(stashArgs...)
	return worktree.StashAll(path)
}

func mixedResetToBase(path, branch, baseBranch string) error {
	// We need to determine the target for echoing (same logic as library)
	target := ""
	if worktree.RemoteBranchExists(branch) {
		target = "origin/" + branch
	} else if baseBranch != "" {
		target = baseBranch
	} else {
		defaultBranch, err := worktree.GetDefaultBranch()
		if err != nil {
			target = "???"
		} else {
			target = defaultBranch
		}
	}
	args := []string{"-C", path, "reset", "--mixed", target}
	echoGitCmd(args...)
	return worktree.MixedResetToBase(path, branch, baseBranch)
}

func pushToRemote(path, branch string) error {
	args := []string{"-C", path, "push", "origin", branch}
	echoGitCmd(args...)
	return worktree.PushToRemote(path, branch)
}

func pushToNewRemote(path, localBranch, remoteBranch string) error {
	args := []string{"-C", path, "push", "-u", "origin", localBranch + ":" + remoteBranch}
	echoGitCmd(args...)
	return worktree.PushToNewRemote(path, localBranch, remoteBranch)
}
