# git-wt

A comprehensive Git worktree management CLI tool written in Go. Streamline your workflow by creating, switching between, and managing isolated work environments within a single repository.

## Features

- **Easy Worktree Management**: Create, switch, list, and remove worktrees with simple commands
- **Auto-Configuration**: Automatically detects project type (Node.js, Python, Go, Ruby) and sets up dependencies
- **Session Tracking**: Remembers task descriptions and creation time for each worktree
- **Setup/Teardown Actions**: Run custom scripts when creating or removing worktrees
- **Interactive Cleanup**: Safely remove merged branches with interactive prompts
- **Cross-Platform**: Works on macOS, Linux, and Windows
- **No External Dependencies**: Unlike the bash version, no need for `yq` or other tools

## Installation

### Quick Install (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/shogs/git-wt/main/install.sh | bash
```

### Homebrew (macOS/Linux)

```bash
brew tap shogs/tap
brew install git-wt
```

### From Source

Requires Go 1.21 or later:

```bash
git clone https://github.com/shogs/git-wt.git
cd git-wt
make install
```

### Manual Installation

1. Download the latest release for your platform from [releases](https://github.com/shogs/git-wt/releases)
2. Extract the archive
3. Move the binary to a directory in your PATH:
   ```bash
   sudo mv git-wt /usr/local/bin/
   # Or for user install:
   mkdir -p ~/bin && mv git-wt ~/bin/
   ```

## Usage

### Initialize Configuration

Create a `.git-wt.yaml` configuration file with auto-detected setup actions:

```bash
git-wt init
```

This will detect your project type and create appropriate setup actions. You can also use:

```bash
git-wt init --minimal    # Create empty config
git-wt init --template=<name>  # Use predefined template (coming soon)
```

### Create a New Worktree

```bash
git-wt new feature-branch          # Create from default branch (main/master)
git-wt new feature-branch develop  # Create from specific base branch
```

This will:
1. Create a new branch and worktree in `.worktrees/feature-branch`
2. Run setup actions from `.git-wt.yaml`
3. Add `.worktrees/` to `.gitignore` automatically

### Switch to a Worktree

```bash
git-wt switch feature-branch       # Print path (for shell integration)
git-wt switch -s feature-branch    # Spawn new shell in worktree
```

For seamless shell integration, add this to your `.bashrc` or `.zshrc`:

```bash
gwt() {
    local path=$(git-wt switch "$1")
    if [ $? -eq 0 ]; then
        cd "$path"
    fi
}
```

Now you can use `gwt feature-branch` to switch directories.

### Resume Work

Display session information when switching:

```bash
git-wt resume feature-branch
git-wt resume -s feature-branch    # With new shell
```

### List Worktrees

```bash
git-wt list           # Simple list
git-wt list -d        # Detailed with session info
```

### View Status

```bash
git-wt status         # Repository and worktree overview
```

### Remove a Worktree

```bash
git-wt remove feature-branch       # Safe removal with checks
git-wt remove -f feature-branch    # Force removal
```

Prevents removal if:
- You're currently inside the worktree
- There are uncommitted changes (unless using `--force`)

### Clean Up Merged Branches

Interactively remove worktrees for branches that have been merged:

```bash
git-wt clean
```

### Create Task-Based Worktree

Create a worktree with a task description and optionally launch Claude Code:

```bash
git-wt task "Implement user authentication"
git-wt task "Fix login bug" bugfix-login
```

### Return to Repository Root

```bash
git-wt root    # or: git-wt main
```

## Configuration

The `.git-wt.yaml` file defines setup and teardown actions:

```yaml
setup:
  - name: npm-install
    description: Install npm dependencies
    script: npm install

  - name: copy-env
    description: Copy environment file
    script: cp $GIT_ROOT/.env $WORKTREE_PATH/.env

teardown:
  - name: cleanup-cache
    description: Clean up cache files
    script: rm -rf node_modules/.cache
```

### Environment Variables

Actions have access to these environment variables:

- `GIT_ROOT`: Repository root path
- `WORKTREE_PATH`: Full path to the worktree
- `WORKTREE_NAME`: Branch name
- `BASE_BRANCH`: Parent branch used to create the worktree

### Auto-Detection

When you run `git-wt init`, it automatically detects:

- **Node.js** (package.json) → `npm install`
- **Python** (requirements.txt) → `pip install -r requirements.txt`
- **Python** (Pipfile) → `pipenv install`
- **Go** (go.mod) → `go mod download`
- **Ruby** (Gemfile) → `bundle install`

## Shell Integration

### Bash/Zsh

Add to your `.bashrc` or `.zshrc`:

```bash
# Quick switch function
gwt() {
    local path=$(git-wt switch "$1")
    if [ $? -eq 0 ]; then
        cd "$path"
    fi
}

# Return to root
alias gwt-root='cd $(git-wt root)'
```

### Fish

Add to your `~/.config/fish/config.fish`:

```fish
function gwt
    set path (git-wt switch $argv[1])
    if test $status -eq 0
        cd $path
    end
end

alias gwt-root='cd (git-wt root)'
```

## Building from Source

### Requirements

- Go 1.21 or later
- Make (optional, for convenient building)

### Build for Current Platform

```bash
make build
```

### Build for All Platforms

```bash
make build-all
```

This creates binaries in `dist/` for:
- macOS (Intel & Apple Silicon)
- Linux (x86_64 & ARM64)
- Windows (x86_64)

### Install Locally

```bash
make install
```

This builds and installs to `~/bin/git-wt`.

## Comparison with Bash Version

This Go port offers several advantages over the [original bash script](https://github.com/shogs/conffiles/blob/main/git-wt):

### Advantages

- **No External Dependencies**: No need for `yq`, `jq`, or other tools
- **Cross-Platform**: Native support for Windows, macOS, and Linux
- **Better Error Handling**: More robust error messages and validation
- **Faster Performance**: Compiled binary with no shell overhead
- **Easier Installation**: Single binary with multiple install methods
- **Better Testing**: Go's testing framework for reliable code

### Feature Parity

All features from the bash version are included:
- All commands (init, new, switch, resume, remove, list, status, clean, task, root)
- YAML configuration with setup/teardown actions
- Session tracking
- Shell spawning
- Interactive cleanup
- Environment variable injection
- Color output

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Development

```bash
# Clone repository
git clone https://github.com/shogs/git-wt.git
cd git-wt

# Install dependencies
go mod download

# Build
go build

# Run tests
go test ./...

# Format code
go fmt ./...
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

Based on the original bash script by [shogs](https://github.com/shogs/conffiles/blob/main/git-wt).
