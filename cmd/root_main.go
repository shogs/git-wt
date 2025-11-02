package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootMainCmd = &cobra.Command{
	Use:     "root",
	Aliases: []string{"main"},
	Short:   "Return to repository root",
	Long:    `Prints the path to the repository root directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureGitRepo(); err != nil {
			return err
		}

		gitRoot, err := getGitRoot()
		if err != nil {
			return err
		}

		fmt.Println(gitRoot)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rootMainCmd)
}
