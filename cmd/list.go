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

			// Get and display status indicators
			if !wt.Detached && wt.Branch != "" {
				var indicators []string

				// Get worktree status (modified, untracked, staged)
				status, err := getWorktreeStatus(wt.Path)
				if err == nil && status != nil {
					if status.Staged > 0 {
						indicators = append(indicators, color.GreenString("+%d", status.Staged))
					}
					if status.Modified > 0 {
						indicators = append(indicators, color.YellowString("*%d", status.Modified))
					}
					if status.Untracked > 0 {
						indicators = append(indicators, color.YellowString("?%d", status.Untracked))
					}
				}

				// Get ahead/behind count
				aheadBehind, err := getAheadBehindCount(wt.Path, wt.Branch)
				if err == nil && aheadBehind != nil {
					if aheadBehind.Ahead > 0 {
						indicators = append(indicators, color.CyanString("↑%d", aheadBehind.Ahead))
					}
					if aheadBehind.Behind > 0 {
						indicators = append(indicators, color.RedString("↓%d", aheadBehind.Behind))
					}
				}

				// Print indicators
				if len(indicators) > 0 {
					fmt.Printf(" %s", strings.Join(indicators, " "))
				}
			}

			fmt.Println()

			if detailed {
				fmt.Printf("    Path: %s\n", wt.Path)
				fmt.Printf("    HEAD: %s\n", wt.Head[:8])

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
