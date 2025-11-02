package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the .git-wt.yaml configuration
type Config struct {
	Setup    []Action `yaml:"setup,omitempty"`
	Teardown []Action `yaml:"teardown,omitempty"`
}

// Action represents a setup or teardown action
type Action struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Script      string `yaml:"script"`
}

// loadConfig loads the .git-wt.yaml configuration
func loadConfig() (*Config, error) {
	gitRoot, err := getGitRoot()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(gitRoot, ".git-wt.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil // Return empty config if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// saveConfig saves the configuration to .git-wt.yaml
func saveConfig(config *Config) error {
	gitRoot, err := getGitRoot()
	if err != nil {
		return err
	}

	configPath := filepath.Join(gitRoot, ".git-wt.yaml")
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// detectProjectType detects the project type and returns a default config
func detectProjectType() *Config {
	gitRoot, err := getGitRoot()
	if err != nil {
		return &Config{}
	}

	config := &Config{}

	// Check for Node.js project
	if fileExists(filepath.Join(gitRoot, "package.json")) {
		config.Setup = append(config.Setup, Action{
			Name:        "npm-install",
			Description: "Install npm dependencies",
			Script:      "npm install",
		})
	}

	// Check for Python project
	if fileExists(filepath.Join(gitRoot, "requirements.txt")) {
		config.Setup = append(config.Setup, Action{
			Name:        "pip-install",
			Description: "Install Python dependencies",
			Script:      "pip install -r requirements.txt",
		})
	} else if fileExists(filepath.Join(gitRoot, "Pipfile")) {
		config.Setup = append(config.Setup, Action{
			Name:        "pipenv-install",
			Description: "Install Pipenv dependencies",
			Script:      "pipenv install",
		})
	}

	// Check for Go project
	if fileExists(filepath.Join(gitRoot, "go.mod")) {
		config.Setup = append(config.Setup, Action{
			Name:        "go-mod-download",
			Description: "Download Go dependencies",
			Script:      "go mod download",
		})
	}

	// Check for Ruby project
	if fileExists(filepath.Join(gitRoot, "Gemfile")) {
		config.Setup = append(config.Setup, Action{
			Name:        "bundle-install",
			Description: "Install Ruby gems",
			Script:      "bundle install",
		})
	}

	return config
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
