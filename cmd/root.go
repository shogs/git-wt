package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var rootCmd = &cobra.Command{
	Use:   "git-wt",
	Short: "Git worktree management tool",
	Long: `git-wt is a comprehensive Git worktree management CLI tool.
It provides streamlined commands for creating, switching between, and managing
isolated work environments within a single repository.`,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add version flag
	rootCmd.Version = "1.0.0"
	rootCmd.SetVersionTemplate("git-wt version {{.Version}}\n")
}

// Helper function to check if we're in a git repository
func ensureGitRepo() error {
	gitRoot, err := getGitRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository")
	}
	if gitRoot == "" {
		return fmt.Errorf("not in a git repository")
	}
	return nil
}

// Helper to get user confirmation (single keystroke, default No)
func confirm(prompt string) bool {
	return confirmWithDefault(prompt, false)
}

// Helper to get user confirmation with configurable default (single keystroke)
func confirmWithDefault(prompt string, defaultYes bool) bool {
	if defaultYes {
		fmt.Printf("%s [Y/n]: ", prompt)
	} else {
		fmt.Printf("%s [y/N]: ", prompt)
	}

	// Get terminal into raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// Fallback to line-based input
		var response string
		fmt.Scanln(&response)
		if response == "" {
			return defaultYes
		}
		return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Read single byte
	b := make([]byte, 1)
	for {
		_, err := os.Stdin.Read(b)
		if err != nil {
			return defaultYes
		}

		switch b[0] {
		case 'y', 'Y':
			fmt.Println("y")
			return true
		case 'n', 'N':
			fmt.Println("n")
			return false
		case '\r', '\n': // Enter - use default
			if defaultYes {
				fmt.Println("y")
			} else {
				fmt.Println("n")
			}
			return defaultYes
		case 3: // Ctrl+C
			fmt.Println()
			return false
		}
		// Ignore other keys
	}
}
