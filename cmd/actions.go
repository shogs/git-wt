package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

// checkCondition runs a shell command and returns true if exit code is 0
func checkCondition(condition string, worktreePath string) bool {
	if condition == "" {
		return true // No condition = always run
	}
	cmd := exec.Command("bash", "-c", condition)
	cmd.Dir = worktreePath
	return cmd.Run() == nil
}

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
		// Check condition first
		if !checkCondition(action.Condition, worktreePath) {
			// Silently skip - condition not met
			continue
		}

		var inputValue string
		var selectedOption *Option

		// Handle input if configured
		if action.Input != nil {
			switch action.Input.Type {
			case "boolean":
				proceed, err := promptBoolean(action.Input.Message)
				if err != nil {
					return fmt.Errorf("input error for %s: %w", action.Name, err)
				}
				if !proceed {
					color.Yellow("⊘ Skipping %s", action.Description)
					continue
				}
			case "option":
				val, opt, err := promptOption(action.Input.Message, action.Input.Options)
				if err != nil {
					return fmt.Errorf("input error for %s: %w", action.Name, err)
				}
				inputValue = val
				selectedOption = opt
			case "text":
				val, err := promptText(action.Input.Message)
				if err != nil {
					return fmt.Errorf("input error for %s: %w", action.Name, err)
				}
				inputValue = val
			default:
				return fmt.Errorf("unknown input type '%s' for action %s", action.Input.Type, action.Name)
			}
		}

		color.Blue("→ %s", action.Description)

		// Build environment variables
		env := append(os.Environ(),
			fmt.Sprintf("GIT_ROOT=%s", gitRoot),
			fmt.Sprintf("WORKTREE_PATH=%s", worktreePath),
			fmt.Sprintf("WORKTREE_NAME=%s", worktreeName),
			fmt.Sprintf("BASE_BRANCH=%s", baseBranch),
			fmt.Sprintf("INPUT_VALUE=%s", inputValue),
		)

		// Run option's script first (if defined)
		if selectedOption != nil && selectedOption.Script != "" {
			cmd := exec.Command("bash", "-c", selectedOption.Script)
			cmd.Dir = worktreePath
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Env = env

			if err := cmd.Run(); err != nil {
				color.Red("✗ Failed to execute option script for %s: %v", action.Name, err)
				return fmt.Errorf("option script for %s failed: %w", action.Name, err)
			}
		}

		// Then run action's main script (with input value as argument if any)
		// Only append argument for single-line scripts; multi-line scripts should use $INPUT_VALUE
		script := action.Script
		if inputValue != "" && !strings.Contains(action.Script, "\n") {
			script = fmt.Sprintf("%s %q", action.Script, inputValue)
		}

		cmd := exec.Command("bash", "-c", script)
		cmd.Dir = worktreePath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env

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
