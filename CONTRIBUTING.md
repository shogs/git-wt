# Contributing to git-wt

Thank you for your interest in contributing to git-wt! This document provides guidelines and instructions for contributing.

## Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/git-wt.git
   cd git-wt
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/shogs/git-wt.git
   ```

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- Make (optional, but recommended)

### Building

```bash
# Build for current platform
go build -o git-wt .

# Or use Make
make build
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -cover ./...
```

### Code Quality

Before submitting a PR, ensure your code passes all checks:

```bash
# Format code
go fmt ./...

# Lint code
go vet ./...

# Build for all platforms
make build-all
```

## Making Changes

1. Create a new branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the coding guidelines below

3. Add tests for new functionality

4. Commit your changes:
   ```bash
   git commit -m "Add feature: description"
   ```

5. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

6. Create a Pull Request

## Coding Guidelines

### Go Code Style

- Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `go fmt` to format your code
- Write clear, descriptive variable and function names
- Add comments for exported functions and complex logic
- Keep functions small and focused

### Commit Messages

- Use present tense ("Add feature" not "Added feature")
- Use imperative mood ("Move cursor to..." not "Moves cursor to...")
- Start with a capital letter
- Keep the first line under 72 characters
- Reference issues and PRs when relevant

Examples:
```
Add support for custom worktree directories
Fix crash when removing non-existent worktree
Update README with installation instructions
```

### Code Organization

- Place new commands in `cmd/` directory
- Follow existing patterns for command structure
- Keep utility functions in appropriate helper files:
  - `cmd/git.go` - Git operations
  - `cmd/config.go` - Configuration handling
  - `cmd/session.go` - Session management
  - `cmd/actions.go` - New/remove actions

## Adding New Commands

When adding a new command:

1. Create a new file in `cmd/` (e.g., `cmd/newcommand.go`)
2. Follow this structure:

```go
package cmd

import (
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new-command [args]",
	Short: "Short description",
	Long:  `Longer description of what the command does.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Command implementation
		return nil
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
	// Add flags if needed
}
```

3. Add tests in `cmd/newcommand_test.go`
4. Update README.md with usage examples
5. Update CHANGELOG.md

## Testing

### Unit Tests

Write unit tests for new functionality:

```go
func TestNewFeature(t *testing.T) {
	// Test implementation
}
```

### Integration Tests

For commands that interact with git, consider adding integration tests that:
- Create a temporary git repository
- Test the command in a realistic scenario
- Clean up after themselves

## Documentation

- Update README.md with new features
- Add examples for new commands
- Update configuration documentation if adding new config options
- Add comments to exported functions

## Pull Request Process

1. Ensure all tests pass
2. Update documentation as needed
3. Add a description of your changes
4. Link any related issues
5. Wait for review from maintainers

### PR Description Template

```markdown
## Description
Brief description of what this PR does

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
Describe how you tested your changes

## Checklist
- [ ] Tests pass locally
- [ ] Code follows style guidelines
- [ ] Documentation updated
- [ ] No new warnings
```

## Release Process

Releases are automated using GitHub Actions and GoReleaser:

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create and push a tag:
   ```bash
   git tag -a v1.1.0 -m "Release version 1.1.0"
   git push origin v1.1.0
   ```
4. GitHub Actions will automatically build and create a release

## Questions?

If you have questions, feel free to:
- Open an issue
- Start a discussion
- Reach out to maintainers

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Focus on constructive feedback
- Assume good intentions

Thank you for contributing to git-wt!
