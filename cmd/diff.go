package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	diffBaseBranch string
	diffStaged     bool
	diffUnstaged   bool
)

var diffCmd = &cobra.Command{
	Use:   "diff [file]",
	Short: "Interactive diff viewer for changed files",
	Long: `Shows an interactive list of files changed in the current worktree
compared to the base branch. Select a file to view a side-by-side diff.

Navigation:
  Arrow keys / j,k  - Navigate file list or scroll diff
  Enter             - Open diff view for selected file
  Escape            - Go back to previous screen
  q / Ctrl+C        - Quit completely
  A / a             - Toggle full file view
  g / G             - Jump to top / bottom`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureGitRepo(); err != nil {
			return err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Validate flag combinations
		if diffStaged && diffUnstaged {
			return fmt.Errorf("cannot use both --staged and --unstaged")
		}

		// Determine base branch
		baseBranch := diffBaseBranch
		if baseBranch == "" {
			// Try to load from session
			session, _ := loadSession(cwd)
			if session != nil && session.BaseBranch != "" {
				baseBranch = session.BaseBranch
			} else {
				// Fallback to default branch
				baseBranch, _ = getDefaultBranch()
			}
		}

		if baseBranch == "" {
			return fmt.Errorf("could not determine base branch; use -b to specify")
		}

		// If specific file provided, show diff directly (need to determine mode)
		if len(args) == 1 {
			// Get files in the appropriate mode
			files, err := getChangedFiles(cwd, baseBranch, diffStaged, diffUnstaged)
			if err != nil {
				return fmt.Errorf("failed to get changed files: %w", err)
			}

			// Find the file in the list
			var targetFile *ChangedFile
			for i := range files {
				if files[i].Path == args[0] {
					targetFile = &files[i]
					break
				}
			}
			if targetFile == nil {
				return fmt.Errorf("file '%s' not in changed files list", args[0])
			}

			// Show single file diff
			diff, err := loadFileDiff(cwd, baseBranch, targetFile, diffStaged, diffUnstaged)
			if err != nil {
				return fmt.Errorf("failed to load diff: %w", err)
			}

			// Just print the diff without interactive mode
			return printNonInteractiveDiff(diff)
		}

		// If flags are provided, skip mode selection and go directly to file list
		if diffStaged || diffUnstaged {
			files, err := getChangedFiles(cwd, baseBranch, diffStaged, diffUnstaged)
			if err != nil {
				return fmt.Errorf("failed to get changed files: %w", err)
			}

			if len(files) == 0 {
				if diffStaged {
					color.Green("No staged changes")
				} else {
					color.Green("No unstaged changes")
				}
				return nil
			}

			return runDiffViewer(files, cwd, baseBranch, diffStaged, diffUnstaged)
		}

		// No flags provided - show interactive mode selection
		return runInteractiveDiff(cwd, baseBranch)
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().StringVarP(&diffBaseBranch, "base", "b", "",
		"Base branch to compare against (default: from session or main)")
	diffCmd.Flags().BoolVar(&diffStaged, "staged", false,
		"Show only staged changes")
	diffCmd.Flags().BoolVar(&diffUnstaged, "unstaged", false,
		"Show only unstaged changes")
}

// printNonInteractiveDiff prints a diff without interactive mode
func printNonInteractiveDiff(dv *DiffView) error {
	// Print file header
	header := dv.File.Path
	if dv.File.OldPath != "" {
		header = fmt.Sprintf("%s -> %s", dv.File.OldPath, dv.File.Path)
	}
	color.Cyan("=== %s ===", header)
	fmt.Println()

	// Print diff lines with simple coloring
	for _, line := range dv.Lines {
		switch line.Type {
		case LineDeletion:
			if line.LeftText != "" {
				color.Red("- %s", line.LeftText)
			}
			if line.RightText != "" {
				color.Green("+ %s", line.RightText)
			}
		case LineAddition:
			color.Green("+ %s", line.RightText)
		case LineContext:
			fmt.Printf("  %s\n", line.LeftText)
		case LineHeader:
			color.Cyan("%s", line.LeftText)
		}
	}

	return nil
}
