package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	previewBranchName string
	previewForce      bool
	previewStaged     bool
	previewUnstaged   bool
)

var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Create and maintain a preview branch merging multiple worktree branches",
	Long: `Interactively select worktree branches to merge into a preview branch.

This command allows you to test how multiple feature branches will work together
before they are individually merged to main. It uses git merge-tree to detect
conflicts before creating the preview branch.

After creating the preview, the command enters watch mode to continuously
monitor for new commits and automatically rebuild the preview branch.

Requires Git 2.38 or later for conflict detection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPreview()
	},
}

func init() {
	rootCmd.AddCommand(previewCmd)
	previewCmd.Flags().StringVarP(&previewBranchName, "name", "n", "preview",
		"Name for the preview branch")
	previewCmd.Flags().BoolVarP(&previewForce, "force", "f", false,
		"Force overwrite if preview branch has uncommitted changes")
	previewCmd.Flags().BoolVarP(&previewStaged, "staged", "s", true,
		"Include staged changes from all selected branches (default: enabled)")
	previewCmd.Flags().BoolVarP(&previewUnstaged, "unstaged", "u", false,
		"Include unstaged changes from all selected branches (default: disabled)")
}

// BranchInfo holds metadata about a branch for display and selection
type BranchInfo struct {
	Branch       string
	CommitsAhead int
	WorktreePath string
	HasWIP       bool
}

func runPreview() error {
	// 1. Validate environment
	if err := ensureGitRepo(); err != nil {
		return err
	}

	// 2. Check Git version (need 2.38+ for merge-tree --write-tree)
	major, minor, _, err := getGitVersion()
	if err != nil {
		return fmt.Errorf("failed to get Git version: %w", err)
	}
	if major < 2 || (major == 2 && minor < 38) {
		return fmt.Errorf("git version 2.38 or later required for conflict detection (have %d.%d)", major, minor)
	}

	// 3. Get default branch
	defaultBranch, err := getDefaultBranch()
	if err != nil {
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	// 4. Build list of selectable branches
	selectItems, branchInfo, err := buildBranchSelectionList(defaultBranch)
	if err != nil {
		return err
	}

	if len(selectItems) == 0 {
		color.Yellow("No worktree branches available for preview")
		return nil
	}

	// 5. Show multi-select UI
	fmt.Printf("Select branches to include in preview (base: %s)\n\n", color.CyanString(defaultBranch))
	selected, confirmed, err := promptMultiSelect("", selectItems)
	if err != nil {
		return fmt.Errorf("selection failed: %w", err)
	}

	if !confirmed {
		color.Yellow("Cancelled")
		return nil
	}

	// Filter to only selected items
	var selectedItems []MultiSelectItem
	for _, item := range selected {
		if item.Selected {
			selectedItems = append(selectedItems, item)
		}
	}

	if len(selectedItems) == 0 {
		color.Yellow("No branches selected")
		return nil
	}

	// 6. Extract selected branch names
	var selectedBranches []string
	for _, item := range selectedItems {
		selectedBranches = append(selectedBranches, item.Value)
	}

	// 7. Show summary with commit counts
	fmt.Println()
	color.Cyan("Selected branches:")
	for _, branch := range selectedBranches {
		info := branchInfo[branch]
		wipStatus := ""
		if info.HasWIP {
			wipStatus = color.YellowString(" [has uncommitted changes]")
		}
		fmt.Printf("  - %s (%d commits ahead of %s)%s\n", branch, info.CommitsAhead, defaultBranch, wipStatus)
	}
	fmt.Println()

	// 8. Check for merge conflicts
	fmt.Print("Checking for merge conflicts... ")
	conflictResult, err := checkMergeConflicts(defaultBranch, selectedBranches)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("conflict check failed: %w", err)
	}

	if conflictResult.HasConflicts {
		color.Red("conflicts detected!")
		fmt.Println()
		color.Red("Conflicting files:")
		for _, file := range conflictResult.ConflictingFiles {
			fmt.Printf("  - %s\n", file)
		}
		fmt.Println()
		color.Yellow("Preview aborted. Resolve conflicts between branches before creating preview.")
		return nil
	}
	color.Green("no conflicts")
	fmt.Println()

	// 9. Get git root
	gitRoot, _ := getGitRoot()

	// 10. Initialize branch enabled, staged, and unstaged settings
	branchEnabled := make(map[string]bool)
	stagedEnabled := make(map[string]bool)
	unstagedEnabled := make(map[string]bool)
	for _, branch := range selectedBranches {
		branchEnabled[branch] = true
		stagedEnabled[branch] = previewStaged
		unstagedEnabled[branch] = previewUnstaged
	}

	// 11. Create preview worktree and merge
	previewWorktreePath, err := createPreviewBranch(gitRoot, defaultBranch, selectedBranches, stagedEnabled, unstagedEnabled, branchInfo)
	if err != nil {
		return err
	}

	// 12. Enter watch mode
	fmt.Println()
	return watchAndMerge(previewWorktreePath, defaultBranch, branchEnabled, stagedEnabled, unstagedEnabled, branchInfo)
}

func buildBranchSelectionList(defaultBranch string) ([]MultiSelectItem, map[string]*BranchInfo, error) {
	worktrees, err := getWorktrees()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get worktrees: %w", err)
	}

	gitRoot, _ := getGitRoot()

	var items []MultiSelectItem
	branchInfo := make(map[string]*BranchInfo)

	for _, wt := range worktrees {
		// Skip main worktree (the git root)
		if wt.Path == gitRoot {
			continue
		}

		// Skip default branch
		if wt.Branch == defaultBranch || wt.Branch == "main" || wt.Branch == "master" {
			continue
		}

		// Skip the preview branch itself
		if wt.Branch == previewBranchName {
			continue
		}

		// Skip detached worktrees
		if wt.Detached || wt.Branch == "" {
			continue
		}

		// Get commit count ahead of default branch
		commitsAhead, err := getCommitCountBetween(defaultBranch, wt.Branch)
		if err != nil {
			commitsAhead = 0
		}

		// Check for uncommitted changes
		hasWIP := false
		if hasChanges, err := hasUncommittedChanges(wt.Path); err == nil && hasChanges {
			hasWIP = true
		}

		// Build info struct
		info := &BranchInfo{
			Branch:       wt.Branch,
			CommitsAhead: commitsAhead,
			WorktreePath: wt.Path,
			HasWIP:       hasWIP,
		}
		branchInfo[wt.Branch] = info

		// Build label with commit count
		label := fmt.Sprintf("%s (%d commits)", wt.Branch, commitsAhead)
		if hasWIP {
			label += color.YellowString(" *")
		}

		items = append(items, MultiSelectItem{
			Label:    label,
			Value:    wt.Branch,
			Selected: false,
		})
	}

	return items, branchInfo, nil
}

func createPreviewBranch(gitRoot, baseBranch string, branches []string, stagedEnabled, unstagedEnabled map[string]bool, branchInfo map[string]*BranchInfo) (string, error) {
	timestamp := time.Now().Format("15:04:05")

	// Create/reset preview worktree
	fmt.Printf("[%s] Creating preview worktree from %s...\n", timestamp, baseBranch)
	worktreePath, wasCreated, err := createPreviewWorktree(gitRoot, previewBranchName, baseBranch)
	if err != nil {
		return "", fmt.Errorf("failed to create preview worktree: %w", err)
	}

	// Run new actions if worktree was just created (like git-wt new does)
	if wasCreated {
		config, err := loadConfig()
		if err == nil && len(config.New) > 0 {
			fmt.Printf("[%s] Running setup actions...\n", timestamp)
			if err := runActions(config.New, worktreePath, previewBranchName, baseBranch); err != nil {
				color.Yellow("Warning: Some setup actions failed: %v", err)
			}
		}
	}

	// Merge each branch's commits
	for i, branch := range branches {
		fmt.Printf("[%s] [%d/%d] Merging %s...\n", timestamp, i+1, len(branches), branch)
		if err := mergeBranchInWorktree(worktreePath, branch, true); err != nil {
			return "", fmt.Errorf("merge failed for %s: %w", branch, err)
		}
	}

	// Apply staged/unstaged changes
	if err := applyChangesInWorktree(worktreePath, branches, stagedEnabled, unstagedEnabled, branchInfo); err != nil {
		return "", err
	}

	fmt.Println()
	color.Green("Preview worktree created at: %s", worktreePath)
	color.Green("Preview branch '%s' ready!", previewBranchName)
	return worktreePath, nil
}

func applyChangesInWorktree(worktreePath string, branches []string, stagedEnabled, unstagedEnabled map[string]bool, branchInfo map[string]*BranchInfo) error {
	timestamp := time.Now().Format("15:04:05")

	for _, branch := range branches {
		info := branchInfo[branch]
		if info == nil || !info.HasWIP {
			continue
		}

		applyStaged := stagedEnabled[branch]
		applyUnstaged := unstagedEnabled[branch]

		if !applyStaged && !applyUnstaged {
			continue
		}

		var appliedParts []string

		// Apply staged changes (and stage them in preview)
		if applyStaged {
			stagedDiff, err := getStagedDiff(info.WorktreePath)
			if err == nil && stagedDiff != "" {
				if err := applyPatchAndStageInWorktree(worktreePath, stagedDiff); err != nil {
					color.Yellow("  Warning: Staged patch from %s could not be applied: %v", branch, err)
				} else {
					appliedParts = append(appliedParts, "staged")
				}
			}
		}

		// Apply unstaged changes
		if applyUnstaged {
			unstagedDiff, err := getUnstagedDiff(info.WorktreePath)
			if err == nil && unstagedDiff != "" {
				if err := applyPatchInWorktree(worktreePath, unstagedDiff); err != nil {
					color.Yellow("  Warning: Unstaged patch from %s could not be applied: %v", branch, err)
				} else {
					appliedParts = append(appliedParts, "unstaged")
				}
			}

			// Add untracked files (only when unstaged is enabled)
			untracked, err := getUntrackedFilesWithContent(info.WorktreePath)
			if err == nil && len(untracked) > 0 {
				for _, f := range untracked {
					if err := addFileContent(worktreePath, f.Path, f.Content); err != nil {
						color.Yellow("  Warning: Could not add untracked file %s: %v", f.Path, err)
					}
				}
				appliedParts = append(appliedParts, fmt.Sprintf("%d untracked", len(untracked)))
			}
		}

		if len(appliedParts) > 0 {
			fmt.Printf("[%s] Applied from %s: %s\n", timestamp, branch, strings.Join(appliedParts, ", "))
		}
	}

	return nil
}

func applyWIPChanges(gitRoot string, branches []string, wipEnabled map[string]bool, branchInfo map[string]*BranchInfo) error {
	timestamp := time.Now().Format("15:04:05")

	for _, branch := range branches {
		if !wipEnabled[branch] {
			continue
		}

		info := branchInfo[branch]
		if info == nil || !info.HasWIP {
			continue
		}

		wip, err := getWIPChanges(info.WorktreePath, branch)
		if err != nil || !wip.HasChanges {
			continue
		}

		fmt.Printf("[%s] Applying WIP from %s...\n", timestamp, branch)

		// Apply diff patch
		if wip.Diff != "" {
			if err := applyPatch(wip.Diff); err != nil {
				color.Yellow("  Warning: WIP patch from %s could not be applied cleanly: %v", branch, err)
				// Continue anyway - don't fail the whole preview
			}
		}

		// Add untracked files
		for _, f := range wip.Untracked {
			if err := addFileContent(gitRoot, f.Path, f.Content); err != nil {
				color.Yellow("  Warning: Could not add untracked file %s: %v", f.Path, err)
			}
		}
	}

	return nil
}

func getWIPChanges(worktreePath, branch string) (*WIPChanges, error) {
	wip := &WIPChanges{
		WorktreePath: worktreePath,
		Branch:       branch,
	}

	// Get combined diff (staged + unstaged vs HEAD)
	diff, err := getWorktreeDiff(worktreePath)
	if err == nil && diff != "" {
		wip.Diff = diff
		wip.HasChanges = true
	}

	// Get untracked files with content
	untracked, err := getUntrackedFilesWithContent(worktreePath)
	if err == nil && len(untracked) > 0 {
		wip.Untracked = untracked
		wip.HasChanges = true
	}

	return wip, nil
}

func watchAndMerge(previewWorktreePath, baseBranch string, branchEnabled map[string]bool, stagedEnabled, unstagedEnabled map[string]bool, branchInfo map[string]*BranchInfo) error {
	// Helper to get list of enabled branches
	getEnabledBranches := func() []string {
		var branches []string
		for branch, enabled := range branchEnabled {
			if enabled {
				branches = append(branches, branch)
			}
		}
		return branches
	}

	// Helper to get sorted list of all branches
	getSortedAllBranches := func() []string {
		var allBranches []string
		for branch := range branchInfo {
			allBranches = append(allBranches, branch)
		}
		for i := 0; i < len(allBranches); i++ {
			for j := i + 1; j < len(allBranches); j++ {
				if allBranches[i] > allBranches[j] {
					allBranches[i], allBranches[j] = allBranches[j], allBranches[i]
				}
			}
		}
		return allBranches
	}

	// Track last known HEAD for each branch (all branches, not just enabled)
	lastHeads := make(map[string]string)
	for branch := range branchInfo {
		head, _ := getBranchHead(branch)
		lastHeads[branch] = head
	}

	// Track last known WIP state
	lastWIPHash := make(map[string]string)
	for branch, info := range branchInfo {
		if info != nil {
			lastWIPHash[branch] = getWIPHash(info.WorktreePath)
		}
	}

	// Refresh branch info from current worktrees
	// Returns: new branches added, branches removed
	refreshBranchInfo := func() (added []string, removed []string) {
		gitRoot, _ := getGitRoot()
		worktrees, err := getWorktrees()
		if err != nil {
			return nil, nil
		}

		// Track current branch names
		currentBranches := make(map[string]bool)

		for _, wt := range worktrees {
			// Skip main worktree, default branch, preview branch, detached
			if wt.Path == gitRoot || wt.Branch == baseBranch ||
				wt.Branch == "main" || wt.Branch == "master" ||
				wt.Branch == previewBranchName || wt.Detached || wt.Branch == "" {
				continue
			}

			currentBranches[wt.Branch] = true

			// Add new branch if not in branchInfo
			if _, exists := branchInfo[wt.Branch]; !exists {
				commitsAhead, _ := getCommitCountBetween(baseBranch, wt.Branch)
				hasWIP := false
				if hasChanges, err := hasUncommittedChanges(wt.Path); err == nil {
					hasWIP = hasChanges
				}

				branchInfo[wt.Branch] = &BranchInfo{
					Branch:       wt.Branch,
					CommitsAhead: commitsAhead,
					WorktreePath: wt.Path,
					HasWIP:       hasWIP,
				}
				added = append(added, wt.Branch)

				// Initialize state maps for new branch
				branchEnabled[wt.Branch] = false // Off by default
				stagedEnabled[wt.Branch] = previewStaged
				unstagedEnabled[wt.Branch] = previewUnstaged
				lastHeads[wt.Branch], _ = getBranchHead(wt.Branch)
				lastWIPHash[wt.Branch] = getWIPHash(wt.Path)
			} else {
				// Update worktree path (might have changed)
				branchInfo[wt.Branch].WorktreePath = wt.Path
			}
		}

		// Find removed branches
		for branch := range branchInfo {
			if !currentBranches[branch] {
				removed = append(removed, branch)
			}
		}

		// Remove deleted branches from all maps
		for _, branch := range removed {
			delete(branchInfo, branch)
			delete(branchEnabled, branch)
			delete(stagedEnabled, branch)
			delete(unstagedEnabled, branch)
			delete(lastHeads, branch)
			delete(lastWIPHash, branch)
		}

		return added, removed
	}

	pollInterval := 5 * time.Second

	// Setup signal handling for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Setup keyboard input handling
	keyChan := make(chan byte, 1)
	var termState *term.State

	if term.IsTerminal(int(os.Stdin.Fd())) {
		var err error
		termState, err = term.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			// Ensure terminal is restored on exit
			defer func() {
				term.Restore(int(os.Stdin.Fd()), termState)
				showCursor()
				clearScreen()
			}()

			go func() {
				buf := make([]byte, 1)
				for {
					n, err := os.Stdin.Read(buf)
					if err != nil || n == 0 {
						return
					}
					keyChan <- buf[0]
				}
			}()
		}
	}

	// TUI state
	var logLines []string
	maxLogLines := 100

	// Helper to add log line
	addLog := func(format string, args ...interface{}) {
		timestamp := time.Now().Format("15:04:05")
		line := fmt.Sprintf("[%s] %s", timestamp, fmt.Sprintf(format, args...))
		logLines = append(logLines, line)
		if len(logLines) > maxLogLines {
			logLines = logLines[1:]
		}
	}

	// Calculate header height based on branches
	getHeaderHeight := func() int {
		branches := getEnabledBranches()
		// Title (1) + preview box (3: top + content + bottom) + source box (2 + branches: top + branches + bottom) + help (1)
		return 1 + 3 + 2 + len(branches) + 1
	}

	// Get preview branch status
	getPreviewStatus := func() string {
		var parts []string

		// Get commits ahead of base
		commits, err := getCommitCountBetween(baseBranch, previewBranchName)
		if err == nil {
			parts = append(parts, fmt.Sprintf("%d commits", commits))
		}

		// Get change counts in preview worktree
		changes, err := getWorktreeChangeCounts(previewWorktreePath)
		if err == nil {
			if changes.Staged > 0 {
				parts = append(parts, color.GreenString("%d staged", changes.Staged))
			}
			if changes.Unstaged > 0 {
				parts = append(parts, color.YellowString("%d unstaged", changes.Unstaged))
			}
			if changes.Untracked > 0 {
				parts = append(parts, color.CyanString("%d untracked", changes.Untracked))
			}
		}

		if len(parts) == 0 {
			return "empty"
		}
		return strings.Join(parts, " + ")
	}

	// Draw the full screen
	drawScreen := func() {
		termWidth, termHeight, _ := getTerminalSize()
		branches := getEnabledBranches()
		headerHeight := getHeaderHeight()

		clearScreen()
		hideCursor()

		// Row tracker
		row := 1

		// Draw header - title bar
		moveCursor(row, 1)
		title := fmt.Sprintf(" PREVIEW: %s ", previewBranchName)
		fmt.Print(color.New(color.BgBlue, color.FgWhite, color.Bold).Sprint(title))
		padding := termWidth - len(title)
		if padding > 0 {
			fmt.Print(color.New(color.BgBlue).Sprint(strings.Repeat(" ", padding)))
		}
		row++

		// Preview box - top border with title
		moveCursor(row, 1)
		previewBoxTitle := "─ Preview "
		previewBoxTitleLen := utf8.RuneCountInString(previewBoxTitle) // visual length
		remainingWidth := termWidth - previewBoxTitleLen - 2         // -2 for corners
		if remainingWidth < 0 {
			remainingWidth = 0
		}
		fmt.Print("┌" + previewBoxTitle + strings.Repeat("─", remainingWidth) + "┐")
		row++

		// Preview box - content
		moveCursor(row, 1)
		previewStatus := getPreviewStatus()
		previewContent := fmt.Sprintf(" %s → %s", color.New(color.Bold).Sprint(previewBranchName), previewStatus)
		// Calculate visible length (rune count without ANSI codes) for padding
		contentPadding := termWidth - visualLen(previewContent) - 2 // -2 for box sides
		if contentPadding < 0 {
			contentPadding = 0
		}
		fmt.Print("│" + previewContent + strings.Repeat(" ", contentPadding) + "│")
		row++

		// Preview box - bottom border
		moveCursor(row, 1)
		fmt.Print("└" + strings.Repeat("─", termWidth-2) + "┘")
		row++

		// Source branches box - top border with title
		moveCursor(row, 1)
		sourceBoxTitle := "─ Source Branches "
		sourceBoxTitleLen := utf8.RuneCountInString(sourceBoxTitle) // visual length
		remainingWidth = termWidth - sourceBoxTitleLen - 2
		if remainingWidth < 0 {
			remainingWidth = 0
		}
		fmt.Print("┌" + sourceBoxTitle + strings.Repeat("─", remainingWidth) + "┐")
		row++

		// Source branches box - content (branch lines)
		for _, branch := range branches {
			moveCursor(row, 1)
			info := branchInfo[branch]
			line := formatBranchLineBoxed(branch, info, stagedEnabled, unstagedEnabled, termWidth)
			fmt.Print(line)
			row++
		}

		// Source branches box - bottom border
		moveCursor(row, 1)
		fmt.Print("└" + strings.Repeat("─", termWidth-2) + "┘")
		row++

		// Help line
		moveCursor(row, 1)
		helpText := " [b]ranches  [s]taged  [u]nstaged  [q]uit"
		fmt.Print(color.New(color.Faint).Sprint(helpText))
		row++

		// Draw log area
		logStartRow := headerHeight + 1
		logHeight := termHeight - headerHeight - 1
		if logHeight < 1 {
			logHeight = 1
		}

		// Show recent log lines
		startIdx := 0
		if len(logLines) > logHeight {
			startIdx = len(logLines) - logHeight
		}

		for i := 0; i < logHeight && startIdx+i < len(logLines); i++ {
			moveCursor(logStartRow+i, 1)
			line := logLines[startIdx+i]
			if len(line) > termWidth {
				line = line[:termWidth]
			}
			fmt.Print(line)
		}

		// Status bar at bottom
		moveCursor(termHeight, 1)
		statusText := fmt.Sprintf(" Watching %d branches | Base: %s | %s ",
			len(branches), baseBranch, time.Now().Format("15:04:05"))
		fmt.Print(color.New(color.BgBlue, color.FgWhite).Sprint(statusText))
		padding = termWidth - len(statusText)
		if padding > 0 {
			fmt.Print(color.New(color.BgBlue).Sprint(strings.Repeat(" ", padding)))
		}
	}

	// Update just the status bar (no screen clear - avoids flicker)
	updateStatusBar := func() {
		termWidth, termHeight, _ := getTerminalSize()
		branches := getEnabledBranches()

		moveCursor(termHeight, 1)
		statusText := fmt.Sprintf(" Watching %d branches | Base: %s | %s ",
			len(branches), baseBranch, time.Now().Format("15:04:05"))
		fmt.Print(color.New(color.BgBlue, color.FgWhite).Sprint(statusText))
		padding := termWidth - len(statusText)
		if padding > 0 {
			fmt.Print(color.New(color.BgBlue).Sprint(strings.Repeat(" ", padding)))
		}
	}

	// Draw modal menu overlay
	drawModal := func(title string, lines []string) {
		termWidth, termHeight, _ := getTerminalSize()

		// Calculate modal dimensions
		modalWidth := 60
		if modalWidth > termWidth-4 {
			modalWidth = termWidth - 4
		}
		modalHeight := len(lines) + 4
		startRow := (termHeight - modalHeight) / 2
		startCol := (termWidth - modalWidth) / 2

		// Draw modal box
		moveCursor(startRow, startCol)
		fmt.Print("┌" + strings.Repeat("─", modalWidth-2) + "┐")

		// Title
		moveCursor(startRow+1, startCol)
		titlePadded := fmt.Sprintf(" %-*s", modalWidth-3, title)
		fmt.Print("│" + color.New(color.Bold).Sprint(titlePadded) + "│")

		// Separator
		moveCursor(startRow+2, startCol)
		fmt.Print("├" + strings.Repeat("─", modalWidth-2) + "┤")

		// Content lines
		for i, line := range lines {
			moveCursor(startRow+3+i, startCol)
			displayLine := line
			if len(displayLine) > modalWidth-4 {
				displayLine = displayLine[:modalWidth-4]
			}
			fmt.Printf("│ %-*s │", modalWidth-4, displayLine)
		}

		// Bottom border
		moveCursor(startRow+3+len(lines), startCol)
		fmt.Print("└" + strings.Repeat("─", modalWidth-2) + "┘")
	}

	// Build branch modal lines
	buildBranchModalLines := func() []string {
		var lines []string
		allBranches := getSortedAllBranches()
		for i, branch := range allBranches {
			status := color.RedString("[OFF]")
			if branchEnabled[branch] {
				status = color.GreenString("[ON]")
			}
			info := branchInfo[branch]
			commits := ""
			if info != nil {
				commits = fmt.Sprintf("(%d commits)", info.CommitsAhead)
			}
			lines = append(lines, fmt.Sprintf("%d. %s %s %s", i+1, branch, status, commits))
		}
		lines = append(lines, "")
		lines = append(lines, "a. Enable ALL    n. Disable ALL    c. Cancel")
		return lines
	}

	// Build staged modal lines
	buildStagedModalLines := func() []string {
		var lines []string
		branches := getEnabledBranches()
		for i, branch := range branches {
			status := color.RedString("[OFF]")
			if stagedEnabled[branch] {
				status = color.GreenString("[ON]")
			}
			info := branchInfo[branch]
			staged := ""
			if info != nil {
				changes, err := getWorktreeChangeCounts(info.WorktreePath)
				if err == nil && changes.Staged > 0 {
					staged = fmt.Sprintf("(%d staged)", changes.Staged)
				}
			}
			lines = append(lines, fmt.Sprintf("%d. %s %s %s", i+1, branch, status, staged))
		}
		lines = append(lines, "")
		lines = append(lines, "a. Enable ALL    n. Disable ALL    c. Cancel")
		return lines
	}

	// Build unstaged modal lines
	buildUnstagedModalLines := func() []string {
		var lines []string
		branches := getEnabledBranches()
		for i, branch := range branches {
			status := color.RedString("[OFF]")
			if unstagedEnabled[branch] {
				status = color.GreenString("[ON]")
			}
			info := branchInfo[branch]
			details := ""
			if info != nil {
				changes, err := getWorktreeChangeCounts(info.WorktreePath)
				if err == nil && (changes.Unstaged > 0 || changes.Untracked > 0) {
					var parts []string
					if changes.Unstaged > 0 {
						parts = append(parts, fmt.Sprintf("%d mod", changes.Unstaged))
					}
					if changes.Untracked > 0 {
						parts = append(parts, fmt.Sprintf("%d new", changes.Untracked))
					}
					details = "(" + strings.Join(parts, ", ") + ")"
				}
			}
			lines = append(lines, fmt.Sprintf("%d. %s %s %s", i+1, branch, status, details))
		}
		lines = append(lines, "")
		lines = append(lines, "a. Enable ALL    n. Disable ALL    c. Cancel")
		return lines
	}

	// Rebuild and log to TUI
	rebuildWithTUI := func(reason string) error {
		branches := getEnabledBranches()
		addLog("─── Rebuild triggered: %s ───", reason)
		addLog("Resetting preview to %s...", baseBranch)
		if err := cleanWorktree(previewWorktreePath, baseBranch); err != nil {
			return err
		}

		addLog("Merging %d branches:", len(branches))
		for i, branch := range branches {
			info := branchInfo[branch]
			commits := 0
			if info != nil {
				commits = info.CommitsAhead
			}
			if err := mergeBranchInWorktree(previewWorktreePath, branch, true); err != nil {
				return err
			}
			addLog("  [%d/%d] %s (%d commits)", i+1, len(branches), branch, commits)
		}

		// Apply staged/unstaged changes
		appliedAny := false
		for _, branch := range branches {
			info := branchInfo[branch]
			if info == nil {
				continue
			}

			applyStaged := stagedEnabled[branch]
			applyUnstaged := unstagedEnabled[branch]

			if !applyStaged && !applyUnstaged {
				continue
			}

			// Get current change counts for logging
			changes, _ := getWorktreeChangeCounts(info.WorktreePath)

			var appliedParts []string

			if applyStaged && changes != nil && changes.Staged > 0 {
				stagedDiff, err := getStagedDiff(info.WorktreePath)
				if err == nil && stagedDiff != "" {
					if err := applyPatchAndStageInWorktree(previewWorktreePath, stagedDiff); err != nil {
						addLog("  ✗ %s: staged patch failed - %v", branch, err)
					} else {
						appliedParts = append(appliedParts, fmt.Sprintf("%d staged", changes.Staged))
						appliedAny = true
					}
				}
			}

			if applyUnstaged {
				if changes != nil && changes.Unstaged > 0 {
					unstagedDiff, err := getUnstagedDiff(info.WorktreePath)
					if err == nil && unstagedDiff != "" {
						if err := applyPatchInWorktree(previewWorktreePath, unstagedDiff); err != nil {
							addLog("  ✗ %s: unstaged patch failed - %v", branch, err)
						} else {
							appliedParts = append(appliedParts, fmt.Sprintf("%d unstaged", changes.Unstaged))
							appliedAny = true
						}
					}
				}

				untracked, err := getUntrackedFilesWithContent(info.WorktreePath)
				if err == nil && len(untracked) > 0 {
					for _, f := range untracked {
						addFileContent(previewWorktreePath, f.Path, f.Content)
					}
					appliedParts = append(appliedParts, fmt.Sprintf("%d untracked", len(untracked)))
					appliedAny = true
				}
			}

			if len(appliedParts) > 0 {
				addLog("  + %s: %s", branch, strings.Join(appliedParts, ", "))
			}
		}

		if !appliedAny {
			addLog("No uncommitted changes to apply")
		}
		addLog("─── Preview updated ───")
		return nil
	}

	// Initial draw
	addLog("Watching for changes...")
	drawScreen()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			clearScreen()
			showCursor()
			fmt.Println("Exiting watch mode...")
			fmt.Printf("Preview worktree remains at: %s\n", previewWorktreePath)
			return nil

		case key := <-keyChan:
			switch key {
			case 'q', 'Q', 3: // q, Q, or Ctrl+C
				clearScreen()
				showCursor()
				fmt.Println("Exiting watch mode...")
				fmt.Printf("Preview worktree remains at: %s\n", previewWorktreePath)
				return nil

			case 'b', 'B':
				// Refresh branch list before showing modal
				added, removed := refreshBranchInfo()
				if len(added) > 0 {
					addLog("New branches available: %s", strings.Join(added, ", "))
				}
				if len(removed) > 0 {
					addLog("Branches removed: %s", strings.Join(removed, ", "))
				}

				// Show branch toggle modal
				drawScreen()
				drawModal("Toggle Branches", buildBranchModalLines())

				// Wait for selection
				select {
				case menuKey := <-keyChan:
					allBranches := getSortedAllBranches()
					changed := false

					switch {
					case menuKey >= '1' && menuKey <= '9':
						idx := int(menuKey - '1')
						if idx < len(allBranches) {
							branch := allBranches[idx]
							branchEnabled[branch] = !branchEnabled[branch]
							if branchEnabled[branch] {
								addLog("Enabled %s", branch)
							} else {
								addLog("Disabled %s", branch)
							}
							changed = true
						}
					case menuKey == 'a' || menuKey == 'A':
						for branch := range branchInfo {
							branchEnabled[branch] = true
						}
						addLog("All branches enabled")
						changed = true
					case menuKey == 'n' || menuKey == 'N':
						for branch := range branchInfo {
							branchEnabled[branch] = false
						}
						addLog("All branches disabled")
						changed = true
					}

					if changed {
						branches := getEnabledBranches()
						if len(branches) == 0 {
							addLog("Warning: No branches selected")
						} else {
							// Check conflicts and rebuild
							addLog("Checking for conflicts...")
							conflictResult, err := checkMergeConflicts(baseBranch, branches)
							if err != nil {
								addLog("Conflict check failed: %v", err)
							} else if conflictResult.HasConflicts {
								addLog(color.RedString("CONFLICT detected!"))
								for _, file := range conflictResult.ConflictingFiles {
									addLog("  - %s", file)
								}
							} else {
								if err := rebuildWithTUI("branch selection changed"); err != nil {
									addLog("Rebuild failed: %v", err)
								}
							}
						}
					}
				case <-time.After(10 * time.Second):
					addLog("Timeout - cancelled")
				}
				drawScreen()

			case 's', 'S':
				// Show staged toggle modal
				drawScreen()
				drawModal("Toggle Staged Changes", buildStagedModalLines())

				branches := getEnabledBranches()
				select {
				case menuKey := <-keyChan:
					changed := false
					var toggledBranch string

					switch {
					case menuKey >= '1' && menuKey <= '9':
						idx := int(menuKey - '1')
						if idx < len(branches) {
							branch := branches[idx]
							stagedEnabled[branch] = !stagedEnabled[branch]
							toggledBranch = branch
							if stagedEnabled[branch] {
								addLog("Staged enabled for %s", branch)
							} else {
								addLog("Staged disabled for %s", branch)
							}
							changed = true
						}
					case menuKey == 'a' || menuKey == 'A':
						for _, branch := range branches {
							stagedEnabled[branch] = true
						}
						toggledBranch = "all branches"
						addLog("Staged enabled for all")
						changed = true
					case menuKey == 'n' || menuKey == 'N':
						for _, branch := range branches {
							stagedEnabled[branch] = false
						}
						toggledBranch = "all branches"
						addLog("Staged disabled for all")
						changed = true
					}

					if changed {
						if err := rebuildWithTUI(fmt.Sprintf("staged toggle: %s", toggledBranch)); err != nil {
							addLog("Rebuild failed: %v", err)
						}
					}
				case <-time.After(10 * time.Second):
					addLog("Timeout - cancelled")
				}
				drawScreen()

			case 'u', 'U':
				// Show unstaged toggle modal
				drawScreen()
				drawModal("Toggle Unstaged Changes", buildUnstagedModalLines())

				branches := getEnabledBranches()
				select {
				case menuKey := <-keyChan:
					changed := false
					var toggledBranch string

					switch {
					case menuKey >= '1' && menuKey <= '9':
						idx := int(menuKey - '1')
						if idx < len(branches) {
							branch := branches[idx]
							unstagedEnabled[branch] = !unstagedEnabled[branch]
							toggledBranch = branch
							if unstagedEnabled[branch] {
								addLog("Unstaged enabled for %s", branch)
							} else {
								addLog("Unstaged disabled for %s", branch)
							}
							changed = true
						}
					case menuKey == 'a' || menuKey == 'A':
						for _, branch := range branches {
							unstagedEnabled[branch] = true
						}
						toggledBranch = "all branches"
						addLog("Unstaged enabled for all")
						changed = true
					case menuKey == 'n' || menuKey == 'N':
						for _, branch := range branches {
							unstagedEnabled[branch] = false
						}
						toggledBranch = "all branches"
						addLog("Unstaged disabled for all")
						changed = true
					}

					if changed {
						if err := rebuildWithTUI(fmt.Sprintf("unstaged toggle: %s", toggledBranch)); err != nil {
							addLog("Rebuild failed: %v", err)
						}
					}
				case <-time.After(10 * time.Second):
					addLog("Timeout - cancelled")
				}
				drawScreen()

			case 'r', 'R':
				// Force refresh
				drawScreen()
			}

		case <-ticker.C:
			// Refresh branch list to detect new/removed worktrees
			added, removed := refreshBranchInfo()
			if len(added) > 0 {
				addLog("New branches available: %s", strings.Join(added, ", "))
				drawScreen()
			}
			if len(removed) > 0 {
				addLog("Branches removed: %s", strings.Join(removed, ", "))
				drawScreen()
			}

			// Get currently enabled branches
			branches := getEnabledBranches()
			if len(branches) == 0 {
				continue
			}

			// Check for commit changes (only enabled branches)
			var changedBranches []string
			for _, branch := range branches {
				currentHead, _ := getBranchHead(branch)
				if currentHead != lastHeads[branch] {
					changedBranches = append(changedBranches, branch)
					lastHeads[branch] = currentHead
				}
			}

			// Check for staged/unstaged changes (only when staged or unstaged is enabled)
			var changesDetectedBranches []string
			for _, branch := range branches {
				if !stagedEnabled[branch] && !unstagedEnabled[branch] {
					continue
				}
				info := branchInfo[branch]
				if info == nil {
					continue
				}
				currentHash := getWIPHash(info.WorktreePath)
				if currentHash != lastWIPHash[branch] {
					changesDetectedBranches = append(changesDetectedBranches, branch)
					lastWIPHash[branch] = currentHash
				}
				// Update HasWIP status
				if hasChanges, err := hasUncommittedChanges(info.WorktreePath); err == nil {
					info.HasWIP = hasChanges
				}
			}

			if len(changedBranches) == 0 && len(changesDetectedBranches) == 0 {
				// Just refresh status bar time (no flicker)
				updateStatusBar()
				continue
			}

			// Build reason string
			var reasons []string
			if len(changedBranches) > 0 {
				addLog("New commits detected in: %s", strings.Join(changedBranches, ", "))
				reasons = append(reasons, fmt.Sprintf("commits in %s", strings.Join(changedBranches, ", ")))
			}
			if len(changesDetectedBranches) > 0 {
				addLog("Uncommitted changes in: %s", strings.Join(changesDetectedBranches, ", "))
				reasons = append(reasons, fmt.Sprintf("changes in %s", strings.Join(changesDetectedBranches, ", ")))
			}

			// Check for conflicts with new commits
			conflictResult, err := checkMergeConflicts(baseBranch, branches)
			if err != nil {
				addLog("Conflict check failed: %v", err)
				drawScreen()
				continue
			}

			if conflictResult.HasConflicts {
				addLog(color.RedString("CONFLICT detected!"))
				for _, file := range conflictResult.ConflictingFiles {
					addLog("  - %s", file)
				}
				addLog("Press 'r' to retry after resolving")
				drawScreen()
				continue
			}

			// Rebuild preview branch
			reason := strings.Join(reasons, " + ")
			if err := rebuildWithTUI(reason); err != nil {
				addLog("Rebuild failed: %v", err)
			}
			drawScreen()
		}
	}
}

// getWIPHash returns a simple hash of the WIP state for change detection
func getWIPHash(worktreePath string) string {
	// Use git status output as a simple change indicator
	status, err := getGitStatus(worktreePath)
	if err != nil {
		return ""
	}
	return status
}

// formatBranchLine formats a branch line for the header display
func formatBranchLine(branch string, info *BranchInfo, stagedEnabled, unstagedEnabled map[string]bool, termWidth int) string {
	if info == nil {
		return fmt.Sprintf(" %s", branch)
	}

	// Build change counts
	var changeParts []string
	changeParts = append(changeParts, fmt.Sprintf("%d commits", info.CommitsAhead))

	if info.WorktreePath != "" {
		changes, err := getWorktreeChangeCounts(info.WorktreePath)
		if err == nil {
			if changes.Staged > 0 {
				if stagedEnabled[branch] {
					changeParts = append(changeParts, color.GreenString("%d staged", changes.Staged))
				} else {
					changeParts = append(changeParts, color.New(color.Faint).Sprintf("%d staged", changes.Staged))
				}
			}
			if changes.Unstaged > 0 {
				if unstagedEnabled[branch] {
					changeParts = append(changeParts, color.YellowString("%d unstaged", changes.Unstaged))
				} else {
					changeParts = append(changeParts, color.New(color.Faint).Sprintf("%d unstaged", changes.Unstaged))
				}
			}
			if changes.Untracked > 0 {
				if unstagedEnabled[branch] {
					changeParts = append(changeParts, color.CyanString("%d untracked", changes.Untracked))
				} else {
					changeParts = append(changeParts, color.New(color.Faint).Sprintf("%d untracked", changes.Untracked))
				}
			}
		}
	}

	// Build indicators
	var indicators []string
	if stagedEnabled[branch] {
		indicators = append(indicators, color.GreenString("S"))
	}
	if unstagedEnabled[branch] {
		indicators = append(indicators, color.YellowString("U"))
	}
	indicatorStr := ""
	if len(indicators) > 0 {
		indicatorStr = "[" + strings.Join(indicators, "") + "]"
	}

	return fmt.Sprintf(" %s %s: %s", indicatorStr, branch, strings.Join(changeParts, " + "))
}

// stripAnsi removes ANSI escape codes from a string to get visible length
func stripAnsi(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

// visualLen returns the visual width of a string (rune count after stripping ANSI)
func visualLen(s string) int {
	return utf8.RuneCountInString(stripAnsi(s))
}

// formatBranchLineBoxed formats a branch line with box borders for the header display
func formatBranchLineBoxed(branch string, info *BranchInfo, stagedEnabled, unstagedEnabled map[string]bool, termWidth int) string {
	if info == nil {
		content := fmt.Sprintf(" %s", branch)
		padding := termWidth - visualLen(content) - 2 // -2 for box sides
		if padding < 0 {
			padding = 0
		}
		return "│" + content + strings.Repeat(" ", padding) + "│"
	}

	// Build change counts
	var changeParts []string
	changeParts = append(changeParts, fmt.Sprintf("%d commits", info.CommitsAhead))

	if info.WorktreePath != "" {
		changes, err := getWorktreeChangeCounts(info.WorktreePath)
		if err == nil {
			if changes.Staged > 0 {
				if stagedEnabled[branch] {
					changeParts = append(changeParts, color.GreenString("%d staged", changes.Staged))
				} else {
					changeParts = append(changeParts, color.New(color.Faint).Sprintf("%d staged", changes.Staged))
				}
			}
			if changes.Unstaged > 0 {
				if unstagedEnabled[branch] {
					changeParts = append(changeParts, color.YellowString("%d unstaged", changes.Unstaged))
				} else {
					changeParts = append(changeParts, color.New(color.Faint).Sprintf("%d unstaged", changes.Unstaged))
				}
			}
			if changes.Untracked > 0 {
				if unstagedEnabled[branch] {
					changeParts = append(changeParts, color.CyanString("%d untracked", changes.Untracked))
				} else {
					changeParts = append(changeParts, color.New(color.Faint).Sprintf("%d untracked", changes.Untracked))
				}
			}
		}
	}

	// Build indicators
	var indicators []string
	if stagedEnabled[branch] {
		indicators = append(indicators, color.GreenString("S"))
	}
	if unstagedEnabled[branch] {
		indicators = append(indicators, color.YellowString("U"))
	}
	indicatorStr := ""
	if len(indicators) > 0 {
		indicatorStr = "[" + strings.Join(indicators, "") + "]"
	}

	content := fmt.Sprintf(" %s %s: %s", indicatorStr, branch, strings.Join(changeParts, " + "))

	// Calculate visible length (rune count without ANSI codes) for padding
	padding := termWidth - visualLen(content) - 2 // -2 for box sides
	if padding < 0 {
		padding = 0
	}

	return "│" + content + strings.Repeat(" ", padding) + "│"
}
