package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	detailed bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all worktrees",
	Long:  `Displays all git worktrees with optional detailed information.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureGitRepo(); err != nil {
			return err
		}

		worktrees, err := getWorktrees()
		if err != nil {
			return fmt.Errorf("failed to get worktrees: %w", err)
		}

		if len(worktrees) == 0 {
			color.Yellow("No worktrees found")
			return nil
		}

		gitRoot, _ := getGitRoot()
		cwd, _ := os.Getwd()

		for i, wt := range worktrees {
			// Determine if this is the main worktree
			isMain := wt.Path == gitRoot

			// Determine if we're currently in this worktree
			isCurrent := strings.HasPrefix(cwd, wt.Path)

			// Format branch name
			branchName := wt.Branch
			if wt.Detached {
				branchName = color.RedString("(detached HEAD)")
			} else if branchName == "" {
				branchName = color.YellowString("(no branch)")
			}

			// Print worktree info
			if isCurrent {
				color.Green("* %s", branchName)
			} else {
				fmt.Printf("  %s", branchName)
			}

			if isMain {
				color.Cyan(" [main]")
			}
			fmt.Println()

			if detailed {
				fmt.Printf("    Path: %s\n", wt.Path)
				fmt.Printf("    HEAD: %s\n", wt.Head[:8])

				// Load and display session info
				if !isMain {
					session, err := loadSession(wt.Path)
					if err == nil && session != nil {
						if session.BaseBranch != "" {
							fmt.Printf("    Base: %s\n", session.BaseBranch)
						}
						if session.Task != "" {
							fmt.Printf("    Task: %s\n", session.Task)
						}
						fmt.Printf("    Created: %s\n", session.Created.Format("2006-01-02 15:04"))
					}
				}

				// Add spacing between entries
				if i < len(worktrees)-1 {
					fmt.Println()
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "Show detailed information")
}
