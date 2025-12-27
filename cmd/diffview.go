package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// LineType categorizes diff lines
type LineType int

const (
	LineContext LineType = iota
	LineAddition
	LineDeletion
	LineHeader
)

// DiffLine represents a single line in a side-by-side diff
type DiffLine struct {
	LeftNum   int      // Line number on left (0 = no line)
	LeftText  string   // Content on left side
	RightNum  int      // Line number on right (0 = no line)
	RightText string   // Content on right side
	Type      LineType // Type of line
}

// DiffHunk represents a block/hunk in the diff
type DiffHunk struct {
	StartLine int // Starting line index in Lines
	EndLine   int // Ending line index (exclusive)
}

// DiffView holds the state for the diff viewer
type DiffView struct {
	File          *ChangedFile
	Lines         []DiffLine
	Hunks         []DiffHunk // List of hunks for block navigation
	CurrentHunk   int        // Currently highlighted hunk index
	ScrollOffset  int
	TermWidth     int
	TermHeight    int
	LeftWidth     int
	RightWidth    int
	ShowFullFile  bool       // Toggle: false = changes only (default), true = full file
	FullFileLines []DiffLine // Full file side-by-side view
	FullFileHunks []DiffHunk // Hunk positions in full file view
	// For hunk staging/unstaging
	Mode         DiffMode
	WorktreePath string
	BaseBranch   string
	RawDiff      string // Original unified diff text for hunk extraction
}

// FileListView holds the state for the file list
type FileListView struct {
	Files        []ChangedFile
	Cursor       int
	BaseBranch   string
	Title        string
	Mode         DiffMode // Track which mode we're in for stage/unstage
	WorktreePath string   // Path for git operations
}

// DiffMode represents what type of diff to show
type DiffMode int

const (
	DiffModeBase     DiffMode = iota // Compare against base branch
	DiffModeStaged                   // Show staged changes
	DiffModeUnstaged                 // Show unstaged changes
)

// DiffModeView holds the state for the diff mode selection
type DiffModeView struct {
	Cursor     int
	BaseBranch string
}

// Key represents a parsed keypress
type Key int

const (
	KeyUp Key = iota
	KeyDown
	KeyEnter
	KeyEscape
	KeyCtrlC
	KeyCtrlD
	KeyCtrlU
	KeyQ
	KeyJ
	KeyK
	KeyG
	KeyShiftG
	KeyA
	KeyShiftA
	KeyPageUp
	KeyPageDown
	KeyMouseScrollUp
	KeyMouseScrollDown
	KeySpace
	KeyOther
)

// ViewAction represents what the view should do
type ViewAction int

const (
	ViewActionNone ViewAction = iota
	ViewActionRedraw
	ViewActionSelect
	ViewActionClose
	ViewActionQuit
	ViewActionRefresh // Refresh file list after stage/unstage
)

// parseKeypress parses raw bytes into a Key
func parseKeypress(b []byte, n int) Key {
	if n == 1 {
		switch b[0] {
		case 'j':
			return KeyJ
		case 'k':
			return KeyK
		case 'q':
			return KeyQ
		case 'G':
			return KeyShiftG
		case 'g':
			return KeyG
		case 'a':
			return KeyA
		case 'A':
			return KeyShiftA
		case '\r', '\n':
			return KeyEnter
		case 27:
			return KeyEscape
		case 3:
			return KeyCtrlC
		case 4:
			return KeyCtrlD
		case 21:
			return KeyCtrlU
		case ' ':
			return KeySpace
		}
	} else if n >= 3 && b[0] == 27 && b[1] == 91 {
		// ESC [ sequences
		switch b[2] {
		case 65: // ESC [ A - Up arrow
			return KeyUp
		case 66: // ESC [ B - Down arrow
			return KeyDown
		case 53: // ESC [ 5 ~ - Page Up
			return KeyPageUp
		case 54: // ESC [ 6 ~ - Page Down
			return KeyPageDown
		case 60: // '<' - SGR mouse encoding: ESC [ < Cb ; Cx ; Cy M/m
			if n >= 9 { // Minimum: ESC [ < D ; D ; D M
				// Parse the button code (number after '<' until ';')
				buttonEnd := 3
				for buttonEnd < n && b[buttonEnd] >= '0' && b[buttonEnd] <= '9' {
					buttonEnd++
				}
				if buttonEnd > 3 && buttonEnd < n && b[buttonEnd] == ';' {
					button := 0
					for i := 3; i < buttonEnd; i++ {
						button = button*10 + int(b[i]-'0')
					}
					// Scroll wheel events in SGR mode:
					// 64 = scroll up, 65 = scroll down
					if button == 64 {
						return KeyMouseScrollUp
					} else if button == 65 {
						return KeyMouseScrollDown
					}
				}
			}
		case 77: // 'M' - Legacy X10 mouse encoding: ESC [ M Cb Cx Cy
			if n >= 6 {
				// In X10 mode, button byte has 32 added
				// Scroll up: 64 + 32 = 96 (0x60)
				// Scroll down: 65 + 32 = 97 (0x61)
				button := int(b[3])
				if button == 96 {
					return KeyMouseScrollUp
				} else if button == 97 {
					return KeyMouseScrollDown
				}
			}
		}
	}
	return KeyOther
}

// parseUnifiedDiff parses unified diff output into DiffLines
func parseUnifiedDiff(diffText string) []DiffLine {
	lines := strings.Split(diffText, "\n")
	var result []DiffLine

	leftLineNum := 0
	rightLineNum := 0

	// Regex to parse hunk header: @@ -start,count +start,count @@
	hunkRegex := regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			// Parse hunk header
			matches := hunkRegex.FindStringSubmatch(line)
			if len(matches) >= 3 {
				leftLineNum, _ = strconv.Atoi(matches[1])
				rightLineNum, _ = strconv.Atoi(matches[2])
			}
			result = append(result, DiffLine{
				LeftText: line,
				Type:     LineHeader,
			})
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			// Deletion - goes on left only
			result = append(result, DiffLine{
				LeftNum:  leftLineNum,
				LeftText: line[1:],
				Type:     LineDeletion,
			})
			leftLineNum++
		} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			// Addition - goes on right only
			result = append(result, DiffLine{
				RightNum:  rightLineNum,
				RightText: line[1:],
				Type:      LineAddition,
			})
			rightLineNum++
		} else if strings.HasPrefix(line, " ") {
			// Context line - both sides
			result = append(result, DiffLine{
				LeftNum:   leftLineNum,
				LeftText:  line[1:],
				RightNum:  rightLineNum,
				RightText: line[1:],
				Type:      LineContext,
			})
			leftLineNum++
			rightLineNum++
		}
		// Skip diff headers (diff --git, index, ---, +++)
	}

	return alignDiffLines(result)
}

// alignDiffLines pairs consecutive deletions and additions side-by-side
func alignDiffLines(lines []DiffLine) []DiffLine {
	var result []DiffLine
	i := 0

	for i < len(lines) {
		if lines[i].Type == LineDeletion {
			// Collect consecutive deletions
			var deletions []DiffLine
			for i < len(lines) && lines[i].Type == LineDeletion {
				deletions = append(deletions, lines[i])
				i++
			}

			// Collect consecutive additions
			var additions []DiffLine
			for i < len(lines) && lines[i].Type == LineAddition {
				additions = append(additions, lines[i])
				i++
			}

			// Pair them side by side
			maxLen := len(deletions)
			if len(additions) > maxLen {
				maxLen = len(additions)
			}

			for j := 0; j < maxLen; j++ {
				line := DiffLine{Type: LineContext}

				if j < len(deletions) {
					line.LeftNum = deletions[j].LeftNum
					line.LeftText = deletions[j].LeftText
					line.Type = LineDeletion
				}
				if j < len(additions) {
					line.RightNum = additions[j].RightNum
					line.RightText = additions[j].RightText
					if line.Type != LineDeletion {
						line.Type = LineAddition
					}
				}
				result = append(result, line)
			}
		} else {
			result = append(result, lines[i])
			i++
		}
	}

	return result
}

// buildAddedFileDiff creates diff lines for a newly added file
func buildAddedFileDiff(content string) []DiffLine {
	lines := strings.Split(content, "\n")
	var result []DiffLine

	for i, line := range lines {
		result = append(result, DiffLine{
			RightNum:  i + 1,
			RightText: line,
			Type:      LineAddition,
		})
	}

	return result
}

// buildDeletedFileDiff creates diff lines for a deleted file
func buildDeletedFileDiff(content string) []DiffLine {
	lines := strings.Split(content, "\n")
	var result []DiffLine

	for i, line := range lines {
		result = append(result, DiffLine{
			LeftNum:  i + 1,
			LeftText: line,
			Type:     LineDeletion,
		})
	}

	return result
}

// DiffChange represents a change identified from unified diff
type DiffChange struct {
	OldStart int // Starting line in old file (1-based)
	OldCount int // Number of lines affected in old file
	NewStart int // Starting line in new file (1-based)
	NewCount int // Number of lines affected in new file
}

// buildFullFileDiff creates a full file side-by-side view with changes highlighted
func buildFullFileDiff(worktreePath, baseBranch string, file *ChangedFile, staged, unstaged bool) ([]DiffLine, []DiffHunk, error) {
	var baseContent, currentContent string
	var err error

	if unstaged {
		// Unstaged: compare staged (index) vs working tree
		baseContent, err = getStagedFileContent(worktreePath, file.Path)
		if err != nil {
			// File might not be staged, try HEAD
			baseContent, err = getFileContent(worktreePath, "HEAD", file.Path)
			if err != nil {
				baseContent = ""
			}
		}
		currentContent, err = getWorkingTreeFileContent(worktreePath, file.Path)
		if err != nil {
			currentContent = ""
		}
	} else if staged {
		// Staged: compare HEAD vs staged (index)
		baseContent, err = getFileContent(worktreePath, "HEAD", file.Path)
		if err != nil {
			baseContent = ""
		}
		currentContent, err = getStagedFileContent(worktreePath, file.Path)
		if err != nil {
			currentContent = ""
		}
	} else {
		// Base branch comparison: compare baseBranch vs HEAD
		baseContent, err = getFileContent(worktreePath, baseBranch, file.Path)
		if err != nil {
			baseContent = ""
		}
		currentContent, err = getFileContent(worktreePath, "HEAD", file.Path)
		if err != nil {
			currentContent = ""
		}
	}

	// Get unified diff to identify changes
	diffText, _ := getFileDiff(worktreePath, baseBranch, file.Path, staged, unstaged)

	// Parse the unified diff to extract change ranges
	changes := parseDiffChanges(diffText)

	// Build full file side-by-side view
	baseLines := strings.Split(baseContent, "\n")
	currentLines := strings.Split(currentContent, "\n")

	// Remove empty last line if content ends with newline
	if len(baseLines) > 0 && baseLines[len(baseLines)-1] == "" {
		baseLines = baseLines[:len(baseLines)-1]
	}
	if len(currentLines) > 0 && currentLines[len(currentLines)-1] == "" {
		currentLines = currentLines[:len(currentLines)-1]
	}

	var result []DiffLine
	var hunks []DiffHunk

	baseIdx := 0  // 0-based index into baseLines
	currIdx := 0  // 0-based index into currentLines
	changeIdx := 0

	for baseIdx < len(baseLines) || currIdx < len(currentLines) {
		// Check if we're at a change boundary
		if changeIdx < len(changes) {
			change := changes[changeIdx]
			// Convert to 0-based
			changeOldStart := change.OldStart - 1
			changeNewStart := change.NewStart - 1

			// Are we at this change?
			if baseIdx == changeOldStart || currIdx == changeNewStart {
				// Record hunk start
				hunkStart := len(result)

				// Process deletions from old file
				deletions := []DiffLine{}
				for i := 0; i < change.OldCount && baseIdx < len(baseLines); i++ {
					deletions = append(deletions, DiffLine{
						LeftNum:  baseIdx + 1,
						LeftText: baseLines[baseIdx],
						Type:     LineDeletion,
					})
					baseIdx++
				}

				// Process additions from new file
				additions := []DiffLine{}
				for i := 0; i < change.NewCount && currIdx < len(currentLines); i++ {
					additions = append(additions, DiffLine{
						RightNum:  currIdx + 1,
						RightText: currentLines[currIdx],
						Type:      LineAddition,
					})
					currIdx++
				}

				// Pair deletions and additions side-by-side
				maxLen := len(deletions)
				if len(additions) > maxLen {
					maxLen = len(additions)
				}

				for i := 0; i < maxLen; i++ {
					line := DiffLine{Type: LineContext}
					if i < len(deletions) {
						line.LeftNum = deletions[i].LeftNum
						line.LeftText = deletions[i].LeftText
						line.Type = LineDeletion
					}
					if i < len(additions) {
						line.RightNum = additions[i].RightNum
						line.RightText = additions[i].RightText
						if line.Type != LineDeletion {
							line.Type = LineAddition
						}
					}
					result = append(result, line)
				}

				// Record hunk end
				if len(result) > hunkStart {
					hunks = append(hunks, DiffHunk{
						StartLine: hunkStart,
						EndLine:   len(result),
					})
				}

				changeIdx++
				continue
			}
		}

		// Context line - same in both files
		if baseIdx < len(baseLines) && currIdx < len(currentLines) {
			result = append(result, DiffLine{
				LeftNum:   baseIdx + 1,
				LeftText:  baseLines[baseIdx],
				RightNum:  currIdx + 1,
				RightText: currentLines[currIdx],
				Type:      LineContext,
			})
			baseIdx++
			currIdx++
		} else if baseIdx < len(baseLines) {
			// Extra lines in base (shouldn't happen with proper diff, but handle it)
			result = append(result, DiffLine{
				LeftNum:  baseIdx + 1,
				LeftText: baseLines[baseIdx],
				Type:     LineDeletion,
			})
			baseIdx++
		} else if currIdx < len(currentLines) {
			// Extra lines in current
			result = append(result, DiffLine{
				RightNum:  currIdx + 1,
				RightText: currentLines[currIdx],
				Type:      LineAddition,
			})
			currIdx++
		}
	}

	// If no hunks were found, create one spanning all changed content
	if len(hunks) == 0 && len(result) > 0 {
		hunks = append(hunks, DiffHunk{StartLine: 0, EndLine: len(result)})
	}

	return result, hunks, nil
}

// parseDiffChanges extracts change ranges from unified diff
func parseDiffChanges(diffText string) []DiffChange {
	var changes []DiffChange
	lines := strings.Split(diffText, "\n")

	// Regex to parse hunk header: @@ -oldStart,oldCount +newStart,newCount @@
	hunkRegex := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			matches := hunkRegex.FindStringSubmatch(line)
			if len(matches) >= 4 {
				oldStart, _ := strconv.Atoi(matches[1])
				oldCount := 1
				if matches[2] != "" {
					oldCount, _ = strconv.Atoi(matches[2])
				}
				newStart, _ := strconv.Atoi(matches[3])
				newCount := 1
				if len(matches) >= 5 && matches[4] != "" {
					newCount, _ = strconv.Atoi(matches[4])
				}

				changes = append(changes, DiffChange{
					OldStart: oldStart,
					OldCount: oldCount,
					NewStart: newStart,
					NewCount: newCount,
				})
			}
		}
	}

	return changes
}

// calculatePaneWidths calculates widths for side-by-side display
func (dv *DiffView) calculatePaneWidths() {
	width, height, _ := getTerminalSize()
	dv.TermWidth = width
	dv.TermHeight = height

	// Layout: [lineNum] left | [lineNum] right
	// Line numbers: 5 chars ("1234 ")
	// Separator: 3 chars " | "
	lineNumWidth := 5
	separator := 3

	availableWidth := width - (lineNumWidth * 2) - separator
	dv.LeftWidth = availableWidth / 2
	dv.RightWidth = availableWidth - dv.LeftWidth
}

// truncateOrPad ensures text fits in given width
func truncateOrPad(text string, width int) string {
	// Handle tabs
	text = strings.ReplaceAll(text, "\t", "    ")

	runes := []rune(text)
	if len(runes) > width {
		if width > 3 {
			return string(runes[:width-3]) + "..."
		}
		return string(runes[:width])
	}
	return text + strings.Repeat(" ", width-len(runes))
}

// printFileList prints the initial file list display (inline, not full-screen)
func printFileList(fl *FileListView) {
	fmt.Println(fl.Title)
	for i, file := range fl.Files {
		prefix := "  "
		if i == fl.Cursor {
			prefix = color.CyanString("> ")
		}

		// Status indicator with color
		var statusStr string
		switch file.Status {
		case "M":
			statusStr = color.YellowString("M")
		case "A":
			statusStr = color.GreenString("A")
		case "D":
			statusStr = color.RedString("D")
		case "R":
			statusStr = color.BlueString("R")
		case "?":
			statusStr = color.MagentaString("?")
		default:
			statusStr = file.Status
		}

		path := file.Path
		if file.OldPath != "" {
			path = fmt.Sprintf("%s -> %s", file.OldPath, file.Path)
		}

		fmt.Printf("%s%s %s\n", prefix, statusStr, path)
	}
	fmt.Print(color.New(color.Faint).Sprint("↑/k up  ↓/j down  enter view  space stage/unstage  esc/q back"))
}

// redrawFileList redraws the file list by moving cursor up and reprinting
func redrawFileList(fl *FileListView, totalLines int) {
	// Move cursor up to the title line and clear to end of screen
	fmt.Printf("\033[%dF", totalLines-1)
	fmt.Print("\033[J")

	// Redraw
	fmt.Println(fl.Title)
	for i, file := range fl.Files {
		prefix := "  "
		if i == fl.Cursor {
			prefix = color.CyanString("> ")
		}

		var statusStr string
		switch file.Status {
		case "M":
			statusStr = color.YellowString("M")
		case "A":
			statusStr = color.GreenString("A")
		case "D":
			statusStr = color.RedString("D")
		case "R":
			statusStr = color.BlueString("R")
		case "?":
			statusStr = color.MagentaString("?")
		default:
			statusStr = file.Status
		}

		path := file.Path
		if file.OldPath != "" {
			path = fmt.Sprintf("%s -> %s", file.OldPath, file.Path)
		}

		fmt.Printf("%s%s %s\n", prefix, statusStr, path)
	}
	fmt.Print(color.New(color.Faint).Sprint("↑/k up  ↓/j down  enter view  space stage/unstage  esc/q back"))
}

// handleFileListInput handles input for the file list view
func handleFileListInput(fl *FileListView, key Key) ViewAction {
	switch key {
	case KeyUp, KeyK:
		if fl.Cursor > 0 {
			fl.Cursor--
			return ViewActionRedraw
		}
	case KeyDown, KeyJ:
		if fl.Cursor < len(fl.Files)-1 {
			fl.Cursor++
			return ViewActionRedraw
		}
	case KeyEnter:
		return ViewActionSelect
	case KeyEscape, KeyQ:
		return ViewActionClose // Go back to mode selection
	case KeyCtrlC:
		return ViewActionQuit // Exit completely
	case KeyShiftG:
		if fl.Cursor != len(fl.Files)-1 {
			fl.Cursor = len(fl.Files) - 1
			return ViewActionRedraw
		}
	case KeyG:
		if fl.Cursor != 0 {
			fl.Cursor = 0
			return ViewActionRedraw
		}
	case KeySpace:
		// Stage/unstage the current file
		if len(fl.Files) > 0 && fl.Cursor < len(fl.Files) {
			file := fl.Files[fl.Cursor]
			switch fl.Mode {
			case DiffModeUnstaged:
				// Stage the file
				_ = stageFile(fl.WorktreePath, file.Path)
				return ViewActionRefresh
			case DiffModeStaged:
				// Unstage the file
				_ = unstageFile(fl.WorktreePath, file.Path)
				return ViewActionRefresh
			}
		}
	}
	return ViewActionNone
}

// printModeSelection prints the diff mode selection view
func printModeSelection(mv *DiffModeView) {
	fmt.Println("Select what to diff:")
	options := []string{
		fmt.Sprintf("Compare against %s", mv.BaseBranch),
		"Staged changes",
		"Unstaged changes",
	}

	for i, opt := range options {
		prefix := "  "
		if i == mv.Cursor {
			prefix = color.CyanString("> ")
		}
		fmt.Printf("%s%s\n", prefix, opt)
	}
	fmt.Print(color.New(color.Faint).Sprint("↑/k up  ↓/j down  enter select  q quit"))
}

// redrawModeSelection redraws the mode selection view
func redrawModeSelection(mv *DiffModeView) {
	// Move cursor up to the title line and clear to end of screen
	// 1 title + 3 options + 1 help = 5 lines, move up 4
	fmt.Printf("\033[%dF", 4)
	fmt.Print("\033[J")

	// Redraw
	fmt.Println("Select what to diff:")
	options := []string{
		fmt.Sprintf("Compare against %s", mv.BaseBranch),
		"Staged changes",
		"Unstaged changes",
	}

	for i, opt := range options {
		prefix := "  "
		if i == mv.Cursor {
			prefix = color.CyanString("> ")
		}
		fmt.Printf("%s%s\n", prefix, opt)
	}
	fmt.Print(color.New(color.Faint).Sprint("↑/k up  ↓/j down  enter select  q quit"))
}

// handleModeSelectionInput handles input for the mode selection view
func handleModeSelectionInput(mv *DiffModeView, key Key) ViewAction {
	switch key {
	case KeyUp, KeyK:
		if mv.Cursor > 0 {
			mv.Cursor--
			return ViewActionRedraw
		}
	case KeyDown, KeyJ:
		if mv.Cursor < 2 { // 3 options (0, 1, 2)
			mv.Cursor++
			return ViewActionRedraw
		}
	case KeyEnter:
		return ViewActionSelect
	case KeyEscape, KeyQ, KeyCtrlC:
		return ViewActionQuit
	}
	return ViewActionNone
}

// enableMouseTracking enables mouse scroll tracking
func enableMouseTracking() {
	// Mode 1003: Report all events (including scroll wheel)
	// Mode 1006: SGR extended mouse mode for better coordinate handling
	fmt.Print("\x1b[?1003h\x1b[?1006h")
	os.Stdout.Sync()
}

// disableMouseTracking disables mouse scroll tracking
func disableMouseTracking() {
	fmt.Print("\x1b[?1006l\x1b[?1003l")
	os.Stdout.Sync()
}

// isLineInCurrentHunk checks if a line index is in the current hunk
func (dv *DiffView) isLineInCurrentHunk(lineIdx int) bool {
	hunks := dv.getActiveHunks()
	if len(hunks) == 0 || dv.CurrentHunk >= len(hunks) {
		return false
	}
	hunk := hunks[dv.CurrentHunk]
	return lineIdx >= hunk.StartLine && lineIdx < hunk.EndLine
}

// renderDiffView renders the side-by-side diff view
func renderDiffView(dv *DiffView) {
	dv.calculatePaneWidths()

	// Get active lines and hunks based on current mode
	lines := dv.getActiveLines()
	hunks := dv.getActiveHunks()

	// Clear screen and position cursor at top-left
	fmt.Print("\x1b[2J\x1b[H")

	// Header with file path, mode indicator, and hunk info (row 1)
	header := fmt.Sprintf(" %s ", dv.File.Path)
	if dv.File.OldPath != "" {
		header = fmt.Sprintf(" %s -> %s ", dv.File.OldPath, dv.File.Path)
	}

	// Mode indicator
	modeIndicator := ""
	if dv.ShowFullFile {
		modeIndicator = " [full] "
	}

	// Hunk info
	hunkInfo := ""
	if len(hunks) > 1 {
		hunkInfo = fmt.Sprintf(" [%d/%d] ", dv.CurrentHunk+1, len(hunks))
	}

	color.New(color.BgBlue, color.FgWhite).Print(header)
	if modeIndicator != "" {
		color.New(color.BgMagenta, color.FgWhite).Print(modeIndicator)
	}
	if hunkInfo != "" {
		color.New(color.BgCyan, color.FgBlack).Print(hunkInfo)
	}
	padding := dv.TermWidth - len(header) - len(modeIndicator) - len(hunkInfo)
	if padding > 0 {
		fmt.Print(strings.Repeat(" ", padding))
	}
	fmt.Print("\r\n")

	// Content area (compact - no column headers or separator)
	viewHeight := dv.TermHeight - 2 // Header + footer only

	endIdx := dv.ScrollOffset + viewHeight
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	for i := dv.ScrollOffset; i < endIdx; i++ {
		isHighlighted := dv.isLineInCurrentHunk(i)
		renderDiffLine(dv, lines[i], isHighlighted)
	}

	// Pad remaining lines to push footer to bottom
	for i := endIdx - dv.ScrollOffset; i < viewHeight; i++ {
		fmt.Print("\r\n")
	}

	// Footer (last row)
	position := fmt.Sprintf(" %d/%d ", dv.ScrollOffset+1, max(1, len(lines)))
	help := "↑↓ scroll  j/k hunk  space stage/unstage  A/a view  esc back"
	fmt.Print(color.New(color.Faint).Sprint(help))
	padding = dv.TermWidth - len(help) - len(position)
	if padding > 0 {
		fmt.Print(strings.Repeat(" ", padding))
	}
	fmt.Print(position)

	// Enable mouse tracking after all rendering
	enableMouseTracking()
}

// renderDiffLine renders a single line of the side-by-side diff
func renderDiffLine(dv *DiffView, line DiffLine, isHighlighted bool) {
	// Format line numbers
	leftNum := "     "
	rightNum := "     "
	if line.LeftNum > 0 {
		leftNum = fmt.Sprintf("%4d ", line.LeftNum)
	}
	if line.RightNum > 0 {
		rightNum = fmt.Sprintf("%4d ", line.RightNum)
	}

	// Format content (account for highlight marker in width)
	effectiveLeftWidth := dv.LeftWidth
	effectiveRightWidth := dv.RightWidth
	leftContent := truncateOrPad(line.LeftText, effectiveLeftWidth)
	rightContent := truncateOrPad(line.RightText, effectiveRightWidth)

	// Apply colors based on line type
	switch line.Type {
	case LineDeletion:
		if isHighlighted {
			fmt.Print(color.New(color.FgYellow).Sprint(leftNum))
		} else {
			fmt.Print(color.New(color.Faint).Sprint(leftNum))
		}
		if line.LeftText != "" {
			color.New(color.FgRed).Print(leftContent)
		} else {
			fmt.Print(leftContent)
		}
		fmt.Print(" | ")
		if isHighlighted {
			fmt.Print(color.New(color.FgYellow).Sprint(rightNum))
		} else {
			fmt.Print(color.New(color.Faint).Sprint(rightNum))
		}
		if line.RightText != "" {
			color.New(color.FgGreen).Print(rightContent)
		} else {
			fmt.Print(rightContent)
		}

	case LineAddition:
		if isHighlighted {
			fmt.Print(color.New(color.FgYellow).Sprint(leftNum))
		} else {
			fmt.Print(color.New(color.Faint).Sprint(leftNum))
		}
		fmt.Print(leftContent)
		fmt.Print(" | ")
		if isHighlighted {
			fmt.Print(color.New(color.FgYellow).Sprint(rightNum))
		} else {
			fmt.Print(color.New(color.Faint).Sprint(rightNum))
		}
		color.New(color.FgGreen).Print(rightContent)

	case LineContext:
		if isHighlighted {
			fmt.Print(color.New(color.FgYellow).Sprint(leftNum))
		} else {
			fmt.Print(color.New(color.Faint).Sprint(leftNum))
		}
		fmt.Print(leftContent)
		fmt.Print(" | ")
		if isHighlighted {
			fmt.Print(color.New(color.FgYellow).Sprint(rightNum))
		} else {
			fmt.Print(color.New(color.Faint).Sprint(rightNum))
		}
		fmt.Print(rightContent)

	case LineHeader:
		if isHighlighted {
			color.New(color.FgCyan, color.Bold).Print(truncateOrPad(line.LeftText, dv.TermWidth))
		} else {
			color.New(color.FgCyan).Print(truncateOrPad(line.LeftText, dv.TermWidth))
		}
	}

	fmt.Print("\r\n")
}

// handleDiffViewInput handles input for the diff view
func handleDiffViewInput(dv *DiffView, key Key) ViewAction {
	_, height, _ := getTerminalSize()
	viewHeight := height - 2 // Header + footer only
	halfPage := viewHeight / 2

	// Get active lines and hunks based on current mode
	lines := dv.getActiveLines()
	hunks := dv.getActiveHunks()

	maxScroll := len(lines) - viewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}

	switch key {
	// Toggle full file view
	case KeyShiftA: // 'A' - activate full file view
		if !dv.ShowFullFile && len(dv.FullFileLines) > 0 {
			dv.ShowFullFile = true
			dv.ScrollOffset = 0 // Reset to top when switching
			dv.CurrentHunk = 0
			return ViewActionRedraw
		}
	case KeyA: // 'a' - deactivate full file view
		if dv.ShowFullFile {
			dv.ShowFullFile = false
			dv.ScrollOffset = 0 // Reset to top when switching
			dv.CurrentHunk = 0
			return ViewActionRedraw
		}

	// Arrow keys: line-by-line scrolling
	case KeyUp:
		if dv.ScrollOffset > 0 {
			dv.ScrollOffset--
			dv.updateCurrentHunk()
			return ViewActionRedraw
		}
	case KeyDown:
		if dv.ScrollOffset < maxScroll {
			dv.ScrollOffset++
			dv.updateCurrentHunk()
			return ViewActionRedraw
		}

	// j/k: block-by-block (hunk) navigation
	case KeyJ:
		if len(hunks) > 0 && dv.CurrentHunk < len(hunks)-1 {
			dv.CurrentHunk++
			dv.scrollToCurrentHunk(viewHeight, maxScroll)
			return ViewActionRedraw
		}
	case KeyK:
		if len(hunks) > 0 && dv.CurrentHunk > 0 {
			dv.CurrentHunk--
			dv.scrollToCurrentHunk(viewHeight, maxScroll)
			return ViewActionRedraw
		}

	// Ctrl+d: half page down
	case KeyCtrlD:
		dv.ScrollOffset += halfPage
		if dv.ScrollOffset > maxScroll {
			dv.ScrollOffset = maxScroll
		}
		dv.updateCurrentHunk()
		return ViewActionRedraw

	// Ctrl+u: half page up
	case KeyCtrlU:
		dv.ScrollOffset -= halfPage
		if dv.ScrollOffset < 0 {
			dv.ScrollOffset = 0
		}
		dv.updateCurrentHunk()
		return ViewActionRedraw

	// Page up/down
	case KeyPageUp:
		dv.ScrollOffset -= viewHeight
		if dv.ScrollOffset < 0 {
			dv.ScrollOffset = 0
		}
		dv.updateCurrentHunk()
		return ViewActionRedraw
	case KeyPageDown:
		dv.ScrollOffset += viewHeight
		if dv.ScrollOffset > maxScroll {
			dv.ScrollOffset = maxScroll
		}
		dv.updateCurrentHunk()
		return ViewActionRedraw

	// Mouse scroll
	case KeyMouseScrollUp:
		if dv.ScrollOffset > 0 {
			dv.ScrollOffset -= 3 // Scroll 3 lines at a time
			if dv.ScrollOffset < 0 {
				dv.ScrollOffset = 0
			}
			dv.updateCurrentHunk()
			return ViewActionRedraw
		}
	case KeyMouseScrollDown:
		if dv.ScrollOffset < maxScroll {
			dv.ScrollOffset += 3 // Scroll 3 lines at a time
			if dv.ScrollOffset > maxScroll {
				dv.ScrollOffset = maxScroll
			}
			dv.updateCurrentHunk()
			return ViewActionRedraw
		}

	case KeyEscape, KeyQ:
		return ViewActionClose
	case KeyCtrlC:
		return ViewActionQuit
	case KeyShiftG:
		if dv.ScrollOffset != maxScroll {
			dv.ScrollOffset = maxScroll
			if len(hunks) > 0 {
				dv.CurrentHunk = len(hunks) - 1
			}
			return ViewActionRedraw
		}
	case KeyG:
		if dv.ScrollOffset != 0 {
			dv.ScrollOffset = 0
			dv.CurrentHunk = 0
			return ViewActionRedraw
		}

	case KeySpace:
		// Stage/unstage current hunk
		if dv.RawDiff != "" && len(hunks) > 0 {
			patch, err := extractHunkPatch(dv.RawDiff, dv.CurrentHunk)
			if err == nil {
				switch dv.Mode {
				case DiffModeUnstaged:
					// Stage this hunk
					if stageHunk(dv.WorktreePath, patch) == nil {
						return ViewActionRefresh
					}
				case DiffModeStaged:
					// Unstage this hunk
					if unstageHunk(dv.WorktreePath, patch) == nil {
						return ViewActionRefresh
					}
				}
			}
		}
	}
	return ViewActionNone
}

// getActiveLines returns the lines to display based on current mode
func (dv *DiffView) getActiveLines() []DiffLine {
	if dv.ShowFullFile {
		return dv.FullFileLines
	}
	return dv.Lines
}

// getActiveHunks returns the hunks to use based on current mode
func (dv *DiffView) getActiveHunks() []DiffHunk {
	if dv.ShowFullFile {
		return dv.FullFileHunks
	}
	return dv.Hunks
}

// scrollToCurrentHunk scrolls the view to show the current hunk
func (dv *DiffView) scrollToCurrentHunk(viewHeight, maxScroll int) {
	hunks := dv.getActiveHunks()
	if len(hunks) == 0 {
		return
	}
	hunk := hunks[dv.CurrentHunk]
	// Scroll so the hunk starts near the top of the view
	dv.ScrollOffset = hunk.StartLine - 1
	if dv.ScrollOffset < 0 {
		dv.ScrollOffset = 0
	}
	if dv.ScrollOffset > maxScroll {
		dv.ScrollOffset = maxScroll
	}
}

// updateCurrentHunk updates the current hunk based on scroll position
func (dv *DiffView) updateCurrentHunk() {
	hunks := dv.getActiveHunks()
	for i, hunk := range hunks {
		if dv.ScrollOffset >= hunk.StartLine && dv.ScrollOffset < hunk.EndLine {
			dv.CurrentHunk = i
			return
		}
	}
	// If past all hunks, select the last one
	if len(hunks) > 0 && dv.ScrollOffset >= hunks[len(hunks)-1].StartLine {
		dv.CurrentHunk = len(hunks) - 1
	}
}

// identifyHunks finds all hunks (blocks) in the diff lines
func identifyHunks(lines []DiffLine) []DiffHunk {
	var hunks []DiffHunk
	currentStart := -1

	for i, line := range lines {
		if line.Type == LineHeader {
			// End previous hunk if exists
			if currentStart >= 0 {
				hunks = append(hunks, DiffHunk{StartLine: currentStart, EndLine: i})
			}
			currentStart = i
		}
	}

	// Add final hunk
	if currentStart >= 0 {
		hunks = append(hunks, DiffHunk{StartLine: currentStart, EndLine: len(lines)})
	}

	// If no headers found (added/deleted files), treat entire content as one hunk
	if len(hunks) == 0 && len(lines) > 0 {
		hunks = append(hunks, DiffHunk{StartLine: 0, EndLine: len(lines)})
	}

	return hunks
}

// loadFileDiff loads the diff for a file and creates a DiffView
func loadFileDiff(worktreePath, baseBranch string, file *ChangedFile, staged, unstaged bool) (*DiffView, error) {
	mode := DiffModeBase
	if staged {
		mode = DiffModeStaged
	} else if unstaged {
		mode = DiffModeUnstaged
	}

	dv := &DiffView{
		File:         file,
		ScrollOffset: 0,
		CurrentHunk:  0,
		ShowFullFile: false,
		Mode:         mode,
		WorktreePath: worktreePath,
		BaseBranch:   baseBranch,
	}

	switch file.Status {
	case "A":
		// Added file - show full content in green
		var content string
		var err error
		if staged {
			// For staged added files, get content from the staging area (index)
			content, err = getStagedFileContent(worktreePath, file.Path)
		} else if unstaged {
			content, err = getWorkingTreeFileContent(worktreePath, file.Path)
		} else {
			// For base branch comparison, get from HEAD
			content, err = getFileContent(worktreePath, "HEAD", file.Path)
		}
		if err != nil {
			return nil, err
		}
		dv.Lines = buildAddedFileDiff(content)
		// For added files, full file view is the same
		dv.FullFileLines = dv.Lines
		dv.FullFileHunks = nil // Will be set below

	case "?":
		// Untracked file - show full content in green (like added)
		content, err := getWorkingTreeFileContent(worktreePath, file.Path)
		if err != nil {
			return nil, err
		}
		dv.Lines = buildAddedFileDiff(content)
		// For untracked files, full file view is the same
		dv.FullFileLines = dv.Lines
		dv.FullFileHunks = nil // Will be set below

	case "D":
		// Deleted file - show full content in red
		content, err := getFileContent(worktreePath, baseBranch, file.Path)
		if err != nil {
			return nil, err
		}
		dv.Lines = buildDeletedFileDiff(content)
		// For deleted files, full file view is the same
		dv.FullFileLines = dv.Lines
		dv.FullFileHunks = nil // Will be set below

	default:
		// Modified or renamed - show unified diff
		diffText, err := getFileDiff(worktreePath, baseBranch, file.Path, staged, unstaged)
		if err != nil {
			return nil, err
		}
		if diffText == "" {
			// No diff content (possibly binary)
			dv.Lines = []DiffLine{{
				LeftText: "(Binary file or no changes)",
				Type:     LineHeader,
			}}
			dv.FullFileLines = dv.Lines
		} else {
			dv.RawDiff = diffText // Store for hunk staging/unstaging
			dv.Lines = parseUnifiedDiff(diffText)
			// Build full file view for modified files
			dv.FullFileLines, dv.FullFileHunks, _ = buildFullFileDiff(worktreePath, baseBranch, file, staged, unstaged)
		}
	}

	// Identify hunks for block navigation (changes-only view)
	dv.Hunks = identifyHunks(dv.Lines)

	// If full file hunks weren't set, use the same as regular hunks
	if dv.FullFileHunks == nil {
		dv.FullFileHunks = dv.Hunks
	}

	return dv, nil
}

// runInteractiveDiff runs the complete interactive diff flow with mode selection
func runInteractiveDiff(worktreePath, baseBranch string) error {
	modeView := &DiffModeView{
		Cursor:     0,
		BaseBranch: baseBranch,
	}

	b := make([]byte, 64)

	// Print initial mode selection
	printModeSelection(modeView)

	for {
		// Get terminal into raw mode for mode selection input
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("failed to set raw mode: %w", err)
		}

		// Read input
		n, err := os.Stdin.Read(b)
		term.Restore(int(os.Stdin.Fd()), oldState)

		if err != nil {
			return err
		}

		key := parseKeypress(b, n)
		action := handleModeSelectionInput(modeView, key)

		switch action {
		case ViewActionQuit:
			fmt.Println()
			return nil
		case ViewActionSelect:
			// Determine mode based on cursor
			var staged, unstaged bool
			switch modeView.Cursor {
			case 0: // Compare against base
				staged, unstaged = false, false
			case 1: // Staged
				staged, unstaged = true, false
			case 2: // Unstaged
				staged, unstaged = false, true
			}

			// Get changed files for selected mode
			files, err := getChangedFiles(worktreePath, baseBranch, staged, unstaged)
			if err != nil {
				fmt.Println()
				return fmt.Errorf("failed to get changed files: %w", err)
			}

			if len(files) == 0 {
				// Clear mode selection and show message
				fmt.Printf("\033[5F\033[J") // Move up 5 lines and clear
				if staged {
					color.Green("No staged changes")
				} else if unstaged {
					color.Green("No unstaged changes")
				} else {
					color.Green("No changes compared to %s", baseBranch)
				}
				fmt.Println()
				// Re-print mode selection
				printModeSelection(modeView)
				continue
			}

			// Clear mode selection before showing file list
			fmt.Printf("\033[5F\033[J") // Move up 5 lines and clear

			// Run file list viewer, returns true if quit, false if go back
			quit, err := runFileListViewer(files, worktreePath, baseBranch, staged, unstaged)
			if err != nil {
				return err
			}
			if quit {
				return nil
			}

			// User pressed Esc, re-print mode selection
			printModeSelection(modeView)

		case ViewActionRedraw:
			redrawModeSelection(modeView)
		}
	}
}

// runFileListViewer runs the file list and diff viewer loop
// Returns: quit (true = quit completely, false = go back to mode selection), error
func runFileListViewer(files []ChangedFile, worktreePath, baseBranch string, staged, unstaged bool) (bool, error) {
	// Initialize file list view
	title := fmt.Sprintf("Changed files (compared to %s):", baseBranch)
	mode := DiffModeBase
	if staged {
		title = "Staged changes:"
		mode = DiffModeStaged
	} else if unstaged {
		title = "Unstaged changes:"
		mode = DiffModeUnstaged
	}

	fileList := &FileListView{
		Files:        files,
		Cursor:       0,
		BaseBranch:   baseBranch,
		Title:        title,
		Mode:         mode,
		WorktreePath: worktreePath,
	}

	totalLines := len(files) + 2 // title + files + help line

	var currentDiff *DiffView
	inDiffView := false

	// Print initial file list (inline)
	printFileList(fileList)

	// Main event loop - larger buffer for mouse events
	b := make([]byte, 64)

	// Helper function to run diff view with persistent raw mode
	// Returns: (exitCompletely, needsRefresh)
	runDiffViewLoop := func() (exitCompletely bool, needsRefresh bool) {
		// Enter raw mode for the entire diff view session
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return true, false
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)

		hideCursor()
		renderDiffView(currentDiff)

		for {
			n, err := os.Stdin.Read(b)
			if err != nil {
				return true, false
			}

			key := parseKeypress(b, n)
			action := handleDiffViewInput(currentDiff, key)

			switch action {
			case ViewActionClose:
				disableMouseTracking()
				clearScreen()
				showCursor()
				return false, false // Go back to file list
			case ViewActionQuit:
				disableMouseTracking()
				clearScreen()
				showCursor()
				return true, false // Exit completely
			case ViewActionRedraw:
				renderDiffView(currentDiff)
			case ViewActionRefresh:
				// Reload the diff after staging/unstaging a hunk
				file := currentDiff.File
				newDiff, err := loadFileDiff(worktreePath, baseBranch, file, staged, unstaged)
				if err != nil || len(newDiff.Lines) == 0 || newDiff.RawDiff == "" {
					// No more changes in this file, go back to file list
					disableMouseTracking()
					clearScreen()
					showCursor()
					return false, true // Go back and refresh file list
				}
				currentDiff = newDiff
				renderDiffView(currentDiff)
			}
		}
	}

	for {
		// Get terminal into raw mode for file list input
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return true, fmt.Errorf("failed to set raw mode: %w", err)
		}

		// Read input
		n, err := os.Stdin.Read(b)

		// Restore terminal before processing
		term.Restore(int(os.Stdin.Fd()), oldState)

		if err != nil {
			return true, err
		}

		key := parseKeypress(b, n)

		if inDiffView {
			// This branch shouldn't be reached anymore since diff view
			// is now handled in its own loop, but keep for safety
			continue
		}

		action := handleFileListInput(fileList, key)
		switch action {
		case ViewActionQuit:
			fmt.Println()
			return true, nil // Quit completely
		case ViewActionClose:
			// Clear file list and go back to mode selection
			fmt.Printf("\033[%dF\033[J", totalLines)
			return false, nil // Go back
		case ViewActionSelect:
			// Load and show diff (full-screen)
			file := &fileList.Files[fileList.Cursor]
			diff, err := loadFileDiff(worktreePath, baseBranch, file, staged, unstaged)
			if err != nil {
				continue
			}
			currentDiff = diff
			inDiffView = true

			// Run the diff view loop (stays in raw mode the entire time)
			exitCompletely, needsRefresh := runDiffViewLoop()
			inDiffView = false
			currentDiff = nil

			if exitCompletely {
				fmt.Println()
				return true, nil // Quit completely
			}

			// If hunk was staged/unstaged, refresh the file list
			if needsRefresh {
				newFiles, err := getChangedFiles(worktreePath, baseBranch, staged, unstaged)
				if err == nil {
					fileList.Files = newFiles
					if fileList.Cursor >= len(newFiles) {
						fileList.Cursor = len(newFiles) - 1
					}
					if fileList.Cursor < 0 {
						fileList.Cursor = 0
					}
					totalLines = len(newFiles) + 2
					if len(newFiles) == 0 {
						return false, nil // No more files, go back to mode selection
					}
				}
			}

			// Re-print the file list inline
			printFileList(fileList)
		case ViewActionRedraw:
			redrawFileList(fileList, totalLines)
		case ViewActionRefresh:
			// Refresh file list after stage/unstage
			newFiles, err := getChangedFiles(worktreePath, baseBranch, staged, unstaged)
			if err == nil {
				// Clear old list
				fmt.Printf("\033[%dF\033[J", totalLines)
				fileList.Files = newFiles
				// Adjust cursor if needed
				if fileList.Cursor >= len(newFiles) {
					fileList.Cursor = len(newFiles) - 1
				}
				if fileList.Cursor < 0 {
					fileList.Cursor = 0
				}
				totalLines = len(newFiles) + 2
				if len(newFiles) == 0 {
					// No more files, go back to mode selection
					return false, nil
				}
				printFileList(fileList)
			}
		}
	}
}

// runDiffViewer runs the diff viewer (legacy entry point, used when flags are provided)
func runDiffViewer(files []ChangedFile, worktreePath, baseBranch string, staged, unstaged bool) error {
	quit, err := runFileListViewer(files, worktreePath, baseBranch, staged, unstaged)
	if err != nil {
		return err
	}
	_ = quit // When called directly, we don't go back to mode selection
	return nil
}

// max returns the larger of two ints
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
