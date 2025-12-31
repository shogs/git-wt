# git-wt Usage Guide

This guide provides detailed usage examples for all git-wt commands. For installation and basic setup, see [README.md](README.md). For a quick 5-minute introduction, see [QUICKSTART.md](QUICKSTART.md).

## Table of Contents

- [Command Overview](#command-overview)
- [Worktree Commands](#worktree-commands)
  - [new](#new---create-a-new-worktree)
  - [switch](#switch---switch-to-a-worktree)
  - [resume](#resume---resume-work-in-a-worktree)
  - [remove](#remove---remove-a-worktree)
  - [list](#list---list-all-worktrees)
  - [status](#status---show-repository-status)
  - [clean](#clean---cleanup-merged-branches)
  - [root](#root---return-to-repository-root)
  - [task](#task---create-task-based-worktree)
  - [preview](#preview---create-a-preview-branch)
- [Diff Commands](#diff-commands)
  - [diff](#diff---interactive-diff-viewer)
- [Configuration](#configuration)
  - [init](#init---initialize-configuration)
- [Workflows](#workflows)
  - [Feature Development](#feature-development-workflow)
  - [Code Review](#code-review-workflow)
  - [Bug Fixing](#bug-fixing-workflow)
  - [Parallel Development](#parallel-development-workflow)
  - [Integration Testing](#integration-testing-workflow)

---

## Command Overview

| Command | Description |
|:--------|:------------|
| `init` | Initialize `.git-wt.yaml` configuration |
| `new` | Create a new worktree with a branch |
| `switch` | Switch to an existing worktree |
| `resume` | Switch with session info display |
| `remove` | Remove a worktree |
| `list` | List all worktrees |
| `status` | Show repository and worktree overview |
| `clean` | Interactive cleanup of merged branches |
| `root` | Return to repository root |
| `task` | Create worktree with task description |
| `preview` | Merge multiple branches into a preview branch |
| `diff` | Interactive diff viewer |

---

## Worktree Commands

### `new` - Create a New Worktree

Creates a new branch and worktree, then runs configured new actions.

```bash
# Basic usage - creates branch from default branch (main/master)
git-wt new feature-auth

# Create from specific base branch
git-wt new feature-auth develop

# Create from a tag or commit
git-wt new hotfix-v1 v1.0.0
```

**What happens:**
1. Creates branch `feature-auth` from the base branch
2. Creates worktree at `.worktrees/feature-auth/`
3. Adds `.worktrees/` to `.gitignore` (if not already)
4. Runs new actions from `.git-wt.yaml`
5. Saves session metadata (branch, base, creation time)

**Examples:**

```bash
# Feature branch from main
git-wt new feature/user-profile

# Bugfix from production branch
git-wt new bugfix/login-error production

# Experiment branch
git-wt new experiment/new-algorithm main
```

---

### `switch` - Switch to a Worktree

Changes to an existing worktree directory.

```bash
# Interactive selection (if no branch specified)
git-wt switch

# Switch to specific worktree (spawns new shell)
git-wt switch feature-auth

# Print path only (for shell integration)
git-wt switch -p feature-auth
```

**Interactive mode:**

When called without arguments, presents a navigable list:

```
Select worktree:
> feature-auth
  bugfix-login
  experiment-ui

↑/k up  ↓/j down  enter select  q cancel
```

**Shell integration:**

Add to your `.bashrc` or `.zshrc`:

```bash
gwt() {
    local path=$(git-wt switch -p "$1")
    if [ $? -eq 0 ]; then
        cd "$path"
    fi
}
```

Then use:
```bash
gwt feature-auth
```

---

### `resume` - Resume Work in a Worktree

Like `switch`, but displays session information when entering.

```bash
# Resume with session info
git-wt resume feature-auth

# Print path only
git-wt resume -p feature-auth
```

**Output example:**

```
═══ Resuming: feature-auth ═══
Task: Implement OAuth2 authentication flow
Created: 2025-01-15 14:30
Base: main

Starting shell in worktree: feature-auth
Type 'exit' to return
```

---

### `remove` - Remove a Worktree

Safely removes a worktree and optionally its branch.

```bash
# Safe removal (checks for uncommitted changes)
git-wt remove feature-auth

# Force removal (ignores uncommitted changes)
git-wt remove -f feature-auth
```

**Safety checks:**
- Prevents removal if you're inside the worktree
- Warns about uncommitted changes (unless `--force`)
- Runs remove actions from `.git-wt.yaml` before deletion

**Examples:**

```bash
# Remove completed feature
git-wt remove feature-auth

# Force remove abandoned experiment
git-wt remove -f experiment-failed
```

---

### `list` - List All Worktrees

Shows all worktrees in the repository.

```bash
# Simple list
git-wt list

# Detailed view with session info
git-wt list -d
```

**Simple output:**

```
main (current)
feature-auth [M:2 U:1]
bugfix-login [+3]
```

**Detailed output (`-d`):**

```
main
  Path: /home/user/project
  HEAD: abc1234

feature-auth [M:2 U:1]
  Path: /home/user/project/.worktrees/feature-auth
  HEAD: def5678
  Task: Implement OAuth authentication
  Created: 2025-01-15 14:30
  Base: main
```

**Status indicators:**
- `[M:n]` - Modified files
- `[U:n]` - Untracked files
- `[S:n]` - Staged files
- `[+n]` - Commits ahead of remote
- `[-n]` - Commits behind remote

---

### `status` - Show Repository Status

Displays an overview of the repository and all worktrees.

```bash
git-wt status
```

**Output example:**

```
═══ Repository Status ═══

Repository: /home/user/project
Default branch: main
Current branch: feature-auth

═══ Worktrees ═══

  main
  * feature-auth [M:2]
    bugfix-login

═══ Current Worktree ═══

On branch feature-auth
Changes not staged for commit:
  modified:   src/auth.go
  modified:   src/config.go
```

---

### `clean` - Cleanup Merged Branches

Interactively removes worktrees for branches that have been merged.

```bash
git-wt clean
```

**Interactive selection:**

```
Select branches to remove:
> [x] feature-completed
  [ ] bugfix-fixed
  [ ] old-experiment

↑/k up  ↓/j down  space select  enter confirm  q cancel
```

**What it does:**
1. Finds branches merged into the default branch
2. Filters to only those with worktrees
3. Lets you select which to remove
4. Runs remove actions and cleans up

---

### `root` - Return to Repository Root

Prints or navigates to the repository root directory.

```bash
# Print root path
git-wt root

# Use with cd
cd $(git-wt root)
```

**Shell alias:**

```bash
alias gwt-root='cd $(git-wt root)'
```

---

### `task` - Create Task-Based Worktree

Creates a worktree with a task description, useful for tracking what you're working on.

```bash
# Auto-generate branch name from task
git-wt task "Implement user authentication"

# Specify branch name
git-wt task "Fix login validation bug" bugfix-login

# With base branch
git-wt task "Add payment processing" feature-payments develop
```

**What happens:**
1. Creates branch (auto-named or specified)
2. Creates worktree
3. Saves task description to session
4. Opens in Claude Code (if installed)

---

### `preview` - Create a Preview Branch

Creates a preview branch by merging multiple worktree branches together, then continuously watches for changes and auto-rebuilds.

```bash
# Interactive selection of branches to merge
git-wt preview

# Include uncommitted changes (WIP) from all branches
git-wt preview --wip

# Custom preview branch name
git-wt preview --name staging

# Force overwrite existing preview branch
git-wt preview --force
```

**Requirements:**
- Git 2.38 or later (for conflict detection via `git merge-tree --write-tree`)

**What happens:**
1. Lists all worktree branches with commit counts
2. Lets you multi-select which branches to include
3. Checks for merge conflicts before creating preview
4. Creates/resets preview branch from main
5. Merges all selected branches sequentially
6. Enters watch mode to monitor for changes
7. Auto-rebuilds when commits are detected

**Interactive selection:**

```
Select branches to include in preview (base: main)

> [ ] feature-auth (5 commits)
  [ ] feature-api (12 commits)
  [ ] bugfix-login (2 commits) *

↑/k up  ↓/j down  space select  enter confirm  q cancel
```

The `*` indicates branches with uncommitted changes.

**Watch mode output:**

```
[14:32:15] Watching for changes... (w: toggle WIP, q: quit)
[14:32:15] WIP: feature-auth [ON], feature-api [OFF]
[14:35:42] Commits changed in: feature-auth
[14:35:42] Rebuilding preview branch...
[14:35:42]   Merged feature-auth
[14:35:42]   Merged feature-api
[14:35:42] Applying WIP from feature-auth...
[14:35:43] Preview branch updated
```

**Watch mode controls:**
- `w` - Toggle WIP inclusion per-branch
- `q` - Quit watch mode
- `Ctrl+C` - Exit gracefully

**Conflict handling:**

If branches have merge conflicts, the preview is aborted:

```
Checking for merge conflicts... conflicts detected!

Conflicting files:
  - src/auth/handler.go
  - src/api/routes.go

Preview aborted. Resolve conflicts between branches before creating preview.
```

During watch mode, conflicts pause auto-rebuild:

```
Merge conflict detected!
After changes in: feature-auth
Conflicting files:
  - src/auth/handler.go

Options:
  [r] Retry - check again for conflicts
  [q] Quit watch mode
```

**WIP (Work-in-Progress) support:**

Include uncommitted changes from worktrees:

```bash
# Enable WIP for all branches at start
git-wt preview --wip
```

Or toggle during watch mode:

```
[w] pressed - Toggle WIP for which branch?

  1. feature-auth [ON]
  2. feature-api [OFF]
  3. bugfix-login [OFF]
  a. Toggle ALL

> 2

WIP enabled for feature-api
Rebuilding preview with updated WIP settings...
```

WIP changes are applied as patches after merging commits, without modifying the original worktrees.

**Flags:**

| Flag | Short | Default | Description |
|:-----|:------|:--------|:------------|
| `--name` | `-n` | `preview` | Name for the preview branch |
| `--force` | `-f` | `false` | Force overwrite preview branch |
| `--wip` | `-w` | `false` | Include uncommitted changes |

**Use cases:**

```bash
# Test how multiple features integrate
git-wt preview

# Live preview with work-in-progress changes
git-wt preview --wip

# Create a staging branch for QA
git-wt preview --name staging

# Refresh preview after resolving conflicts
git-wt preview --force
```

---

## Diff Commands

### `diff` - Interactive Diff Viewer

Shows changed files with an interactive side-by-side diff viewer.

```bash
# Compare against base branch (from session or main)
git-wt diff

# Compare against specific branch
git-wt diff -b develop

# Show only staged changes
git-wt diff --staged

# Show only unstaged changes
git-wt diff --unstaged

# View diff for specific file (non-interactive)
git-wt diff cmd/auth.go
```

**Interactive file list:**

```
Changed files (compared to main)

> M src/auth.go
  M src/config.go
  A src/oauth.go
  D src/legacy.go

↑/k up  ↓/j down  enter view  q quit
```

**Status indicators:**
- `M` - Modified (yellow)
- `A` - Added (green)
- `D` - Deleted (red)
- `R` - Renamed (blue)

**Side-by-side diff view:**

```
 src/auth.go
     Original                    |      Modified
────────────────────────────────────────────────────────
  10 func authenticate() {       |   10 func authenticate() {
  11     return false            |   11     return true
  12 }                           |   12 }
  13                             |   13
                                 |   14 func validate() bool {
                                 |   15     return true
                                 |   16 }

↑/k up  ↓/j down  esc back  q quit                1/25
```

**File list navigation:**
- `↑` / `k` - Move up
- `↓` / `j` - Move down
- `Enter` - Open diff view
- `Esc` / `q` - Exit

**Diff view navigation:**
- `↑` / `↓` - Scroll line by line
- `j` / `k` - Jump to next/previous hunk (block)
- `Ctrl+d` - Half page down
- `Ctrl+u` - Half page up
- `g` / `G` - Jump to top/bottom
- Mouse scroll - Scroll up/down
- `Esc` - Back to file list
- `q` - Quit

**Color coding:**
- Red text - Deleted lines (left side)
- Green text - Added lines (right side)
- Normal text - Unchanged context lines
- Cyan - Headers and file names

**Use cases:**

```bash
# Review all changes before committing
git-wt diff --unstaged

# Review what will be committed
git-wt diff --staged

# Compare feature branch to develop
git-wt diff -b develop

# Quick look at specific file changes
git-wt diff src/main.go
```

---

## Configuration

### `init` - Initialize Configuration

Creates a `.git-wt.yaml` configuration file with auto-detected actions.

```bash
# Auto-detect project type and create config
git-wt init

# Create minimal empty config
git-wt init --minimal
```

**Auto-detection:**

| Project Type | Detection | Action |
|:-------------|:----------|:-------|
| Node.js | `package.json` | `npm install` |
| Python (pip) | `requirements.txt` | `pip install -r requirements.txt` |
| Python (pipenv) | `Pipfile` | `pipenv install` |
| Go | `go.mod` | `go mod download` |
| Ruby | `Gemfile` | `bundle install` |

**Example generated config:**

```yaml
new:
  - name: npm-install
    description: Install npm dependencies
    script: npm install
```

---

## Workflows

### Feature Development Workflow

```bash
# 1. Start new feature
git-wt new feature/user-dashboard

# 2. Work in isolated environment
gwt feature/user-dashboard
# ... make changes, commit ...

# 3. Review your changes
git-wt diff -b main

# 4. When done, merge and clean up
git checkout main
git merge feature/user-dashboard
git-wt remove feature/user-dashboard
```

### Code Review Workflow

```bash
# 1. Create worktree for the PR branch
git-wt new review/pr-123 origin/pr-123-feature

# 2. Switch to it
gwt review/pr-123

# 3. Review changes compared to main
git-wt diff -b main

# 4. Test, add review comments
npm test

# 5. Clean up when done
git-wt remove review/pr-123
```

### Bug Fixing Workflow

```bash
# 1. Create hotfix from production
git-wt new hotfix/critical-bug production

# 2. Switch and fix
gwt hotfix/critical-bug

# 3. Check what changed
git-wt diff -b production

# 4. Test and merge
npm test
git checkout production
git merge hotfix/critical-bug
git-wt remove hotfix/critical-bug
```

### Parallel Development Workflow

```bash
# Create multiple worktrees for parallel work
git-wt new feature/auth
git-wt new feature/payments
git-wt new bugfix/validation

# List all active work
git-wt list -d

# Switch between them as needed
gwt feature/auth
# ... work ...
gwt feature/payments
# ... work ...

# Check status across all
git-wt status

# Clean up merged branches periodically
git-wt clean
```

### Integration Testing Workflow

```bash
# 1. Create worktrees for features being developed
git-wt new feature/auth
git-wt new feature/payments
git-wt new feature/notifications

# 2. Work on each feature in parallel
gwt feature/auth
# ... implement auth ...

gwt feature/payments
# ... implement payments ...

# 3. Test how features integrate together
git-wt preview

# Select all branches you want to test together
# The preview branch is created with all features merged

# 4. Run tests on the preview branch
npm test

# 5. Watch mode auto-rebuilds as you make changes
# In one terminal: git-wt preview (keeps running)
# In other terminals: work on feature branches

# 6. Include work-in-progress for real-time testing
git-wt preview --wip
# Now even uncommitted changes are included in preview

# 7. When ready, merge features individually to main
git checkout main
git merge feature/auth
git merge feature/payments
```

---

## Tips and Best Practices

### Branch Naming

Use descriptive, hierarchical branch names:

```bash
git-wt new feature/user-auth
git-wt new bugfix/login-validation
git-wt new experiment/new-algorithm
git-wt new refactor/database-layer
```

### Shell Integration

Set up aliases for faster workflow:

```bash
# In .bashrc or .zshrc
gwt() {
    local path=$(git-wt switch -p "$1")
    [ $? -eq 0 ] && cd "$path"
}

alias gwt-root='cd $(git-wt root)'
alias gwt-list='git-wt list -d'
alias gwt-status='git-wt status'
alias gwt-diff='git-wt diff'
```

### Regular Cleanup

Keep your workspace clean:

```bash
# Check for merged branches weekly
git-wt clean

# Review all worktrees
git-wt list -d
```

### Session Tracking

Use tasks to track what you're working on:

```bash
git-wt task "JIRA-123: Implement OAuth2 flow"
```

Later, resume with context:

```bash
git-wt resume feature-oauth
# Shows: Task: JIRA-123: Implement OAuth2 flow
```

---

## Getting Help

```bash
# List all commands
git-wt --help

# Help for specific command
git-wt diff --help
git-wt new --help
git-wt remove --help
```

---

## See Also

- [README.md](README.md) - Full documentation and installation
- [QUICKSTART.md](QUICKSTART.md) - 5-minute getting started guide
- [.git-wt.yaml.example](.git-wt.yaml.example) - Example configuration
