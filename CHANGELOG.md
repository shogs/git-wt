# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.9] - 2026-04-13

### Changed

- `new` now creates worktrees from within an existing worktree and creates path in root .worktrees path

## [0.2.0] - 2025-12-25

### Added

- **Interactive worktree selection**: `switch` and `remove` commands now show interactive selection when no branch is specified
  - Navigate with arrow keys or vim-style `j`/`k`
  - `remove` supports multi-select with space to toggle
  - `switch` uses single-select with enter to confirm
- **Smart removal prompts**: When removing worktrees with uncommitted changes or unpushed commits, users are prompted with contextual options:
  - `f` - Force remove (discard changes)
  - `s` - Stash changes before removing (when uncommitted changes exist)
  - `p` - Push to remote first (when unpushed commits exist)
  - `b` - Stash and push (when both issues exist)
  - `a` - Stash all (reset branch to base and stash everything including commits)
- **Push to new remote branch**: When pushing unpushed commits, prompts for remote branch name with current branch as default
- **Single keystroke confirmations**: Y/n prompts now respond immediately without requiring Enter
- **Ctrl+C support**: All interactive prompts now properly handle Ctrl+C to cancel

### Changed

- **Config renamed**: `setup`/`teardown` actions renamed to `new`/`remove` to match command names
- **Switch default behavior**: `switch` now spawns a shell by default; use `-p` flag to print path for shell integration
- **Stash includes untracked**: Stash operations now include untracked files with `--include-untracked`
- Improved terminal UI handling for better compatibility across different terminal emulators

### Fixed

- Branch deletion after stashing changes now works correctly
- Unpushed commit detection works for branches without remote tracking
- Interactive UI redraws properly in neovim terminal and other terminal emulators

## [0.1.0] - 2025-10-28

### Added

- Initial release of git-wt Go CLI
- Port of bash script functionality to Go
- Commands:
  - `init` - Initialize git-wt configuration
  - `new`/`add` - Create new worktree
  - `switch` - Switch to a worktree
  - `resume` - Resume work with session info
  - `remove` - Remove a worktree
  - `list` - List all worktrees
  - `status` - Show repository status
  - `clean` - Interactive cleanup of merged branches
  - `task` - Create worktree with task description
  - `root`/`main` - Return to repository root
- YAML configuration support without external dependencies
- Auto-detection of project types (Node.js, Python, Go, Ruby)
- Session tracking with task descriptions
- Setup and teardown action support
- Cross-platform builds (macOS, Linux, Windows)
- Installation script
- GoReleaser configuration
- GitHub Actions workflows
- Comprehensive documentation

### Changed

- No longer depends on `yq` for YAML parsing
- Improved error handling and validation
- Better cross-platform compatibility

### Security

- Prevents removal of worktree while inside it
- Checks for uncommitted changes before removal
- Safe handling of file operations

## [Unreleased]

### Planned

- Template support for common project types
- Git hooks integration
- Worktree archiving
- Batch operations
- Custom worktree locations
- Plugin system
