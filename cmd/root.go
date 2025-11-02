package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
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

// Helper to get user confirmation
func confirm(prompt string) bool {
	var response string
	fmt.Printf("%s [y/N]: ", prompt)
	fmt.Scanln(&response)
	return response == "y" || response == "Y" || response == "yes" || response == "Yes"
}

// Helper to get user confirmation with configurable default
func confirmWithDefault(prompt string, defaultYes bool) bool {
	var response string
	if defaultYes {
		fmt.Printf("%s [Y/n]: ", prompt)
	} else {
		fmt.Printf("%s [y/N]: ", prompt)
	}
	fmt.Scanln(&response)

	// Empty response uses default
	if response == "" {
		return defaultYes
	}

	return response == "y" || response == "Y" || response == "yes" || response == "Yes"
}
