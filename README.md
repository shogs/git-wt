# git-wt

A comprehensive Git worktree management CLI tool written in Go. Streamline your workflow by creating, switching between, and managing isolated work environments within a single repository.

## Features

- **Easy Worktree Management**: Create, switch, list, and remove worktrees with simple commands
- **Interactive Diff Viewer**: Side-by-side diff view with vim-style navigation
- **Auto-Configuration**: Automatically detects project type (Node.js, Python, Go, Ruby) and sets up dependencies
- **Session Tracking**: Remembers task descriptions and creation time for each worktree
- **New/Remove Actions**: Run custom scripts when creating or removing worktrees
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

Create a `.git-wt.yaml` configuration file with auto-detected actions:

```bash
git-wt init
```

This will detect your project type and create appropriate actions. You can also use:

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
2. Run new actions from `.git-wt.yaml`
3. Add `.worktrees/` to `.gitignore` automatically

### Switch to a Worktree

```bash
git-wt switch feature-branch       # Spawn new shell in worktree (default)
git-wt switch -p feature-branch    # Print path (for shell integration)
```

For seamless shell integration, add this to your `.bashrc` or `.zshrc`:

```bash
gwt() {
    local path=$(git-wt switch -p "$1")
    if [ $? -eq 0 ]; then
        cd "$path"
    fi
}
```

Now you can use `gwt feature-branch` to switch directories.

### Resume Work

Display session information when switching:

```bash
git-wt resume feature-branch       # Spawn new shell with session info (default)
git-wt resume -p feature-branch    # Print path (for shell integration)
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

### Interactive Diff Viewer

View changed files with an interactive side-by-side diff:

```bash
git-wt diff                    # Compare against base branch
git-wt diff -b develop         # Compare against specific branch
git-wt diff --staged           # Show staged changes only
git-wt diff --unstaged         # Show unstaged changes only
git-wt diff src/file.go        # View specific file (non-interactive)
```

Navigation:
- Arrow keys - Scroll line by line in diff view
- `j` / `k` - Navigate file list, or jump between hunks in diff view
- `Ctrl+d` / `Ctrl+u` - Half page scroll
- Mouse scroll - Scroll in diff view
- `Enter` - Open diff view for selected file
- `Esc` - Close diff view / exit
- `g` / `G` - Jump to top / bottom
- `q` - Quit

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

The `.git-wt.yaml` file defines new and remove actions:

```yaml
new:
  - name: npm-install
    description: Install npm dependencies
    script: npm install

  - name: copy-env
    description: Copy environment file
    script: cp $GIT_ROOT/.env $WORKTREE_PATH/.env

remove:
  - name: cleanup-cache
    description: Clean up cache files
    script: rm -rf node_modules/.cache
```

### Conditional Actions

Actions can have a `condition` that determines if they run. The condition is a shell command - if it exits with code 0, the action runs:

```yaml
new:
  - name: npm-install
    description: Install npm dependencies
    condition: "test -f package.json"  # Only run if package.json exists
    script: npm install

  - name: docker-setup
    description: Start Docker containers
    condition: "command -v docker"     # Only run if docker is installed
    script: docker-compose up -d
```

### Interactive Input

Actions can prompt for user input before running. Three input types are supported:

```yaml
new:
  # Boolean: y/n prompt (single keypress, same line)
  # Display: "Install dependencies? [y/n]:"
  - name: install-deps
    description: Install dependencies
    input:
      type: boolean
      message: Install dependencies?
    script: npm install

  # Option: key-based selection (single keypress, case-sensitive)
  # With descriptions - displays multi-line list
  # Display:
  #   Which environment?
  #     d) Local dev environment      (bold+green if default)
  #     s) Pre-production testing
  #     p) Live production
  #   Select:
  - name: select-env
    description: Configure environment
    input:
      type: option
      message: Which environment?
      options:
        - option: d
          description: Local dev environment
          default: true              # Displayed in bold+green
        - option: s
          description: Pre-production testing
        - option: p
          description: Live production
    script: ./setup-env.sh  # receives selected key as argument

  # Option without descriptions - displays on same line as message
  # Display: "Which database? [p, m, s]:"
  - name: select-db
    description: Select database
    input:
      type: option
      message: Which database?
      options:
        - option: p                  # Press 'p' for postgres
          default: true
        - option: m                  # Press 'm' for mysql
        - option: s                  # Press 's' for sqlite
    script: ./setup-db.sh

  # Option with per-option scripts
  - name: setup-cloud
    description: Configure cloud provider
    input:
      type: option
      message: Which cloud provider?
      options:
        - option: a
          description: Amazon Web Services
          script: ./setup-aws.sh      # Runs first for this option
        - option: g
          description: Google Cloud Platform
          script: ./setup-gcp.sh      # Runs first for this option
        - option: z
          description: Microsoft Azure
          script: ./setup-azure.sh    # Runs first for this option
    script: ./finalize-cloud.sh       # Then runs with option as argument

  # Text: free-form input (requires Enter)
  - name: set-port
    description: Configure port
    input:
      type: text
      message: Enter the port number
    script: ./configure-port.sh  # receives entered text as argument
```

For `option` and `text` inputs, the value is passed to the script both as:
- A command argument (appended to the script)
- An environment variable `INPUT_VALUE`

For `option` inputs with per-option scripts: the option's script runs first, then the action's main script runs with the option value as argument.

Option selection is case-sensitive. The default option (marked with `default: true`) is displayed in uppercase, bold, and green - press the uppercase key (Shift+key) or Enter to select it. Other options are shown in lowercase and selected by pressing the lowercase key.

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
    local path=$(git-wt switch -p "$1")
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
    set path (git-wt switch -p $argv[1])
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

## Library Usage

The `worktree` package can be imported into other Go projects to programmatically manage Git worktrees.

### Installation

```bash
go get github.com/shogs/git-wt/worktree
```

### Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/shogs/git-wt/worktree"
)

func main() {
    // Query functions - no stdout output
    root, err := worktree.GetGitRoot()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Repository root: %s\n", root)

    // List all worktrees
    wts, err := worktree.GetWorktrees()
    if err != nil {
        log.Fatal(err)
    }
    for _, wt := range wts {
        fmt.Printf("  %s -> %s\n", wt.Branch, wt.Path)
    }

    // Check worktree status
    status, _ := worktree.GetWorktreeStatus(root)
    fmt.Printf("Modified: %d, Staged: %d, Untracked: %d\n",
        status.Modified, status.Staged, status.Untracked)

    // Create a new worktree (silent - no CLI output)
    err = worktree.CreateWorktree("feature-x", "main", ".worktrees/feature-x", true)
    if err != nil {
        log.Fatal(err)
    }

    // Remove a worktree
    err = worktree.RemoveWorktree(".worktrees/feature-x", false)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Available Functions

**Query Functions** (read-only, no stdout):

| Function | Description |
|----------|-------------|
| `GetGitRoot()` | Returns the repository root directory |
| `GetCurrentBranch()` | Returns the current branch name |
| `GetWorktrees()` | Returns a list of all worktrees |
| `GetDefaultBranch()` | Returns the default branch (main/master) |
| `GetWorktreeDir()` | Returns the `.worktrees` directory path |
| `BranchExists(branch)` | Checks if a local branch exists |
| `RemoteBranchExists(branch)` | Checks if a remote branch exists |
| `HasUncommittedChanges(path)` | Checks for uncommitted changes |
| `GetGitStatus(path)` | Returns git status output |
| `GetWorktreeStatus(path)` | Returns modified/staged/untracked counts |
| `GetAheadBehindCount(path, branch)` | Returns commits ahead/behind remote |
| `GetUnpushedCommitCount(path, branch, baseBranch)` | Returns unpushed commit count |

**Operations** (state-changing, silent):

| Function | Description |
|----------|-------------|
| `CreateWorktree(branch, baseBranch, path, newBranch)` | Creates a new worktree |
| `RemoveWorktree(path, force)` | Removes a worktree |
| `DeleteBranch(branch, force)` | Deletes a local branch |
| `EnsureWorktreeDir()` | Creates `.worktrees` directory and updates `.gitignore` |
| `StashChanges(path)` | Stashes changes including untracked files |
| `StashAll(path)` | Stages everything and stashes |
| `MixedResetToBase(path, branch, baseBranch)` | Resets to base with changes unstaged |
| `PushToRemote(path, branch)` | Pushes to remote tracking branch |
| `PushToNewRemote(path, localBranch, remoteBranch)` | Pushes and sets upstream |

**Types:**

```go
type WorktreeInfo struct {
    Path     string
    Branch   string
    Head     string
    Detached bool
}

type WorktreeStatus struct {
    Modified  int
    Untracked int
    Staged    int
}

type AheadBehindCount struct {
    Ahead  int
    Behind int
}
```

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

All features from the bash version are included, plus new additions:
- All commands (init, new, switch, resume, remove, list, status, clean, task, root, diff)
- Interactive diff viewer with side-by-side view (new in Go version)
- YAML configuration with new/remove actions
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

## Documentation

- [QUICKSTART.md](QUICKSTART.md) - Get started in 5 minutes
- [USAGE.md](USAGE.md) - Detailed usage examples for all commands
- [.git-wt.yaml.example](.git-wt.yaml.example) - Example configuration file

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

Based on the original bash script by [shogs](https://github.com/shogs/conffiles/blob/main/git-wt).
