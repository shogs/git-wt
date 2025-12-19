package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the .git-wt.yaml configuration
type Config struct {
	New    []Action `yaml:"new,omitempty"`
	Remove []Action `yaml:"remove,omitempty"`
}

// Option represents a selectable option with optional description and script
type Option struct {
	Option      string `yaml:"option"`                // Required: the key to press
	Description string `yaml:"description,omitempty"` // Optional: help text
	Script      string `yaml:"script,omitempty"`      // Optional: script for this option
	Default     bool   `yaml:"default,omitempty"`     // Optional: mark as default option
}

// InputConfig represents user input configuration for an action
type InputConfig struct {
	Type    string   `yaml:"type"`              // "boolean", "option", "text"
	Message string   `yaml:"message"`           // Prompt message (can be multiline)
	Options []Option `yaml:"options,omitempty"` // For "option" type only
}

// Action represents a new or remove action
type Action struct {
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Script      string       `yaml:"script"`
	Condition   string       `yaml:"condition,omitempty"` // Shell command, runs if exit 0
	Input       *InputConfig `yaml:"input,omitempty"`
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
		config.New = append(config.New, Action{
			Name:        "npm-install",
			Description: "Install npm dependencies",
			Script:      "npm install",
		})
	}

	// Check for Python project
	if fileExists(filepath.Join(gitRoot, "requirements.txt")) {
		config.New = append(config.New, Action{
			Name:        "pip-install",
			Description: "Install Python dependencies",
			Script:      "pip install -r requirements.txt",
		})
	} else if fileExists(filepath.Join(gitRoot, "Pipfile")) {
		config.New = append(config.New, Action{
			Name:        "pipenv-install",
			Description: "Install Pipenv dependencies",
			Script:      "pipenv install",
		})
	}

	// Check for Go project
	if fileExists(filepath.Join(gitRoot, "go.mod")) {
		config.New = append(config.New, Action{
			Name:        "go-mod-download",
			Description: "Download Go dependencies",
			Script:      "go mod download",
		})
	}

	// Check for Ruby project
	if fileExists(filepath.Join(gitRoot, "Gemfile")) {
		config.New = append(config.New, Action{
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
