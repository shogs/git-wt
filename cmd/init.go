package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	templateName string
	minimal      bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize git-wt configuration",
	Long:  `Creates a .git-wt.yaml configuration file with auto-detected project setup actions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureGitRepo(); err != nil {
			return err
		}

		gitRoot, _ := getGitRoot()
		configPath := fmt.Sprintf("%s/.git-wt.yaml", gitRoot)

		// Check if config already exists
		if fileExists(configPath) {
			if !confirm(color.YellowString("Config file already exists. Overwrite?")) {
				color.Yellow("Initialization cancelled")
				return nil
			}
		}

		var config *Config

		if minimal {
			// Create minimal config
			config = &Config{}
		} else if templateName != "" {
			// Load template (for future extension)
			config = &Config{}
			color.Yellow("Template support coming soon, creating auto-detected config")
			config = detectProjectType()
		} else {
			// Auto-detect project type
			config = detectProjectType()
		}

		if err := saveConfig(config); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		color.Green("✓ Created .git-wt.yaml")

		if len(config.Setup) > 0 {
			fmt.Println("\nDetected setup actions:")
			for _, action := range config.Setup {
				fmt.Printf("  • %s\n", action.Description)
			}
		} else {
			fmt.Println("\nNo setup actions detected. Edit .git-wt.yaml to add custom actions.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&templateName, "template", "", "Use a predefined template")
	initCmd.Flags().BoolVar(&minimal, "minimal", false, "Create minimal config without auto-detection")
}
