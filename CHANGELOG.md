# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-10-28

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
