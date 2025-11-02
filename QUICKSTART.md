# Quick Start Guide

Get started with git-wt in 5 minutes!

## Installation

### macOS (Homebrew)
```bash
brew tap shogs/tap
brew install git-wt
```

### Quick Install Script
```bash
curl -fsSL https://raw.githubusercontent.com/shogs/git-wt/main/install.sh | bash
```

## Basic Workflow

### 1. Initialize in your repository

```bash
cd your-project
git-wt init
```

This creates a `.git-wt.yaml` file with auto-detected setup actions.

### 2. Create your first worktree

```bash
git-wt new feature-login
```

This creates a new branch and worktree in `.worktrees/feature-login/`.

### 3. Switch to the worktree

Add this helper to your shell config (`.bashrc`, `.zshrc`):

```bash
gwt() {
    local path=$(git-wt switch "$1")
    if [ $? -eq 0 ]; then
        cd "$path"
    fi
}
```

Now you can quickly switch:

```bash
gwt feature-login
```

### 4. Work on your feature

Your worktree is isolated from the main codebase:
- Make changes, commit as normal
- Run tests without affecting main worktree
- Setup actions (like `npm install`) already ran automatically

### 5. View all worktrees

```bash
git-wt list
```

For detailed info:

```bash
git-wt list -d
```

### 6. Return to main

```bash
gwt-root    # If you added the alias
# Or:
cd $(git-wt root)
```

### 7. Clean up when done

Remove a single worktree:

```bash
git-wt remove feature-login
```

Clean up all merged branches:

```bash
git-wt clean
```

## Common Use Cases

### Working on multiple features

```bash
git-wt new feature-auth
git-wt new feature-ui
git-wt new bugfix-validation

# Switch between them easily
gwt feature-auth
gwt feature-ui
```

### Code review workflow

```bash
# Create worktree for PR review
git-wt new review-pr-123

# Make changes, test
gwt review-pr-123

# When done
git-wt remove review-pr-123
```

### Task-based workflow

```bash
# Creates worktree with task description
git-wt task "Implement OAuth authentication"

# Creates worktree with custom branch name
git-wt task "Fix login bug" bugfix-login-issue
```

## Configuration Example

Edit `.git-wt.yaml` to customize setup actions:

```yaml
setup:
  - name: install-deps
    description: Install dependencies
    script: npm install

  - name: setup-db
    description: Setup test database
    script: npm run db:setup

teardown:
  - name: cleanup
    description: Clean up test data
    script: npm run db:clean
```

## Shell Integration

### Bash/Zsh

Add to `.bashrc` or `.zshrc`:

```bash
# Quick switch to worktree
gwt() {
    local path=$(git-wt switch "$1")
    if [ $? -eq 0 ]; then
        cd "$path"
    fi
}

# Return to repository root
alias gwt-root='cd $(git-wt root)'

# List worktrees
alias gwt-list='git-wt list'
```

### Fish

Add to `~/.config/fish/config.fish`:

```fish
function gwt
    set path (git-wt switch $argv[1])
    if test $status -eq 0
        cd $path
    end
end

alias gwt-root='cd (git-wt root)'
alias gwt-list='git-wt list'
```

## Pro Tips

1. **Use descriptive branch names**: Makes it easier to identify worktrees
   ```bash
   git-wt new feature/user-auth
   git-wt new bugfix/login-validation
   ```

2. **Leverage setup actions**: Automate environment setup
   - Copy `.env` files
   - Install dependencies
   - Run database migrations
   - Generate mock data

3. **Check status often**: See what you have in progress
   ```bash
   git-wt status
   ```

4. **Clean regularly**: Remove merged branches to keep things tidy
   ```bash
   git-wt clean
   ```

5. **Use session tracking**: The `resume` command shows what you were working on
   ```bash
   git-wt resume feature-login
   ```

## Next Steps

- Read the full [README.md](README.md) for all commands
- Check out [.git-wt.yaml.example](.git-wt.yaml.example) for configuration ideas
- Set up shell integration for seamless workflow
- Join the discussion and contribute!

## Getting Help

```bash
# Show all commands
git-wt --help

# Help for specific command
git-wt new --help
git-wt switch --help
```

## Troubleshooting

### "not in a git repository"

Make sure you're inside a git repository:
```bash
git status  # Should show git info
```

### Worktree directory issues

The worktrees are created in `.worktrees/` by default. This is automatically added to `.gitignore`.

### Setup actions failing

Check your `.git-wt.yaml` syntax and make sure scripts are executable. You can test actions by running them manually in a worktree.

---

**Happy coding with git-wt!** 🚀
