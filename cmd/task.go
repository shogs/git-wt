package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task <description> [branch]",
	Short: "Create a worktree and launch Claude Code",
	Long:  `Creates a new worktree with a task description and optionally launches Claude Code.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureGitRepo(); err != nil {
			return err
		}

		taskDesc := args[0]
		var branchName string

		if len(args) > 1 {
			branchName = args[1]
		} else {
			// Generate branch name from task description
			branchName = generateBranchName(taskDesc)
		}

		// Check if branch already exists
		if branchExists(branchName) {
			return fmt.Errorf("branch '%s' already exists", branchName)
		}

		// Get base branch
		baseBranch, err := getDefaultBranch()
		if err != nil {
			baseBranch = "main"
		}

		// Ensure .worktrees directory exists
		worktreeDir, err := ensureWorktreeDir()
		if err != nil {
			return err
		}

		worktreePath := filepath.Join(worktreeDir, branchName)

		// Create worktree
		color.Blue("Creating worktree for task: %s", taskDesc)
		if err := createWorktree(branchName, baseBranch, worktreePath, true); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}

		color.Green("✓ Worktree created at %s", worktreePath)

		// Create session file with task description
		session := &Session{
			Branch:     branchName,
			Created:    time.Now(),
			BaseBranch: baseBranch,
			Task:       taskDesc,
		}
		if err := saveSession(worktreePath, session); err != nil {
			color.Yellow("⚠ Failed to create session file: %v", err)
		}

		// Run setup actions
		config, err := loadConfig()
		if err != nil {
			color.Yellow("⚠ Failed to load config: %v", err)
		} else if len(config.Setup) > 0 {
			fmt.Println("\nRunning setup actions...")
			if err := runActions(config.Setup, worktreePath, branchName, baseBranch); err != nil {
				color.Yellow("⚠ Setup actions completed with errors")
			}
		}

		// Launch Claude Code if available
		claudePath, err := exec.LookPath("claude")
		if err == nil {
			fmt.Println()
			color.Blue("Launching Claude Code...")
			claudeCmd := exec.Command(claudePath)
			claudeCmd.Dir = worktreePath
			claudeCmd.Stdin = os.Stdin
			claudeCmd.Stdout = os.Stdout
			claudeCmd.Stderr = os.Stderr
			claudeCmd.Run()
		} else {
			fmt.Println()
			color.Yellow("Claude Code not found in PATH. Worktree ready at:")
			fmt.Println(worktreePath)
		}

		return nil
	},
}

func generateBranchName(task string) string {
	// Simple branch name generation: lowercase, replace spaces with hyphens
	name := strings.ToLower(task)
	name = strings.ReplaceAll(name, " ", "-")
	// Remove special characters
	var result []rune
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result = append(result, r)
		}
	}
	name = string(result)
	// Limit length
	if len(name) > 50 {
		name = name[:50]
	}
	// Remove trailing hyphens
	name = strings.TrimRight(name, "-")
	return name
}

func init() {
	rootCmd.AddCommand(taskCmd)
}
