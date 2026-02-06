package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show repository and worktree status",
	Long:  `Displays the overall git repository status and information about all worktrees.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureGitRepo(); err != nil {
			return err
		}

		gitRoot, _ := getGitRoot()
		currentBranch, _ := getCurrentBranch()

		// Display header
		color.Cyan("═══ Git Worktree Status ═══\n")
		fmt.Printf("Repository: %s\n", gitRoot)
		fmt.Printf("Current branch: %s\n", currentBranch)
		fmt.Println()

		// Get worktrees
		worktrees, err := getWorktrees()
		if err != nil {
			return fmt.Errorf("failed to get worktrees: %w", err)
		}

		if len(worktrees) == 0 {
			color.Yellow("No worktrees found")
			return nil
		}

		// Display worktrees
		color.Cyan("Worktrees (%d):\n", len(worktrees))
		cwd, _ := os.Getwd()

		for _, wt := range worktrees {
			isMain := wt.Path == gitRoot
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
			fmt.Printf("    %s\n", wt.Path)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
