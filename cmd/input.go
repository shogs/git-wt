package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// promptBoolean displays a message and waits for a single y/n keypress (no Enter needed)
// Returns true if user pressed y/Y, false if n/N
func promptBoolean(message string) (bool, error) {
	// Print message and options on same line
	fmt.Printf("%s %s ", message, color.CyanString("[y/n]:"))

	// Get single character input without requiring Enter
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// Fallback to standard input if raw mode fails
		fmt.Println() // newline for fallback
		return promptBooleanFallback()
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Read single byte
	b := make([]byte, 1)
	for {
		_, err := os.Stdin.Read(b)
		if err != nil {
			return false, fmt.Errorf("failed to read input: %w", err)
		}

		char := strings.ToLower(string(b[0]))
		if char == "y" {
			fmt.Println("y")
			return true, nil
		}
		if char == "n" {
			fmt.Println("n")
			return false, nil
		}
		// Ignore other keys, wait for y or n
	}
}

// promptBooleanFallback uses standard input with Enter key
func promptBooleanFallback() (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes", nil
}

// hasAnyDescription checks if any option has a description
func hasAnyDescription(options []Option) bool {
	for _, opt := range options {
		if opt.Description != "" {
			return true
		}
	}
	return false
}

// buildOptionKeyMap creates a map of display key -> Option for case-sensitive lookup
// Default options are mapped to uppercase, others to lowercase
func buildOptionKeyMap(options []Option) map[string]*Option {
	keyMap := make(map[string]*Option)
	for i := range options {
		if options[i].Default {
			keyMap[strings.ToUpper(options[i].Option)] = &options[i]
		} else {
			keyMap[strings.ToLower(options[i].Option)] = &options[i]
		}
	}
	return keyMap
}

// findDefaultOption returns the default option if one exists
func findDefaultOption(options []Option) *Option {
	for i := range options {
		if options[i].Default {
			return &options[i]
		}
	}
	return nil
}

// formatOptionKey formats an option key with styling
// Default: uppercase + bold + green
// Non-default: lowercase
func formatOptionKey(key string, isDefault bool) string {
	if isDefault {
		return color.New(color.Bold, color.FgGreen).Sprint(strings.ToUpper(key))
	}
	return strings.ToLower(key)
}

// promptOption displays options and waits for a single keypress to select
// Returns: selected value, pointer to selected Option, error
func promptOption(message string, options []Option) (string, *Option, error) {
	if len(options) == 0 {
		return "", nil, fmt.Errorf("no options provided")
	}

	// Build key map for lookup (case-sensitive)
	keyMap := buildOptionKeyMap(options)

	// Build option keys list for display
	var optionKeys []string
	for _, opt := range options {
		optionKeys = append(optionKeys, formatOptionKey(opt.Option, opt.Default))
	}

	// Always show options in the question line
	if hasAnyDescription(options) {
		// Show question with options, then descriptions below
		fmt.Printf("%s %s\n", message, color.CyanString("[%s]:", strings.Join(optionKeys, ", ")))
		for _, opt := range options {
			keyDisplay := formatOptionKey(opt.Option, opt.Default)
			if opt.Description != "" {
				fmt.Printf("  %s) %s\n", keyDisplay, opt.Description)
			} else {
				fmt.Printf("  %s)\n", keyDisplay)
			}
		}
	} else {
		// Same line format without descriptions
		fmt.Printf("%s %s ", message, color.CyanString("[%s]:", strings.Join(optionKeys, ", ")))
	}

	// Get single character input without requiring Enter
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// Fallback to standard input if raw mode fails
		if hasAnyDescription(options) {
			// Already on new line
		} else {
			fmt.Println() // newline for fallback
		}
		return promptOptionFallback(options, keyMap)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Find default option for Enter key handling
	defaultOpt := findDefaultOption(options)

	// Read single byte
	b := make([]byte, 1)
	for {
		_, err := os.Stdin.Read(b)
		if err != nil {
			return "", nil, fmt.Errorf("failed to read input: %w", err)
		}

		// Check for Enter key - select default if available
		if b[0] == '\r' || b[0] == '\n' {
			if defaultOpt != nil {
				fmt.Println(strings.ToUpper(defaultOpt.Option))
				return defaultOpt.Option, defaultOpt, nil
			}
			// No default, ignore Enter
			continue
		}

		// Check if it's a valid option key (case-sensitive)
		key := string(b[0])
		if opt, ok := keyMap[key]; ok {
			fmt.Println(key)
			return opt.Option, opt, nil
		}
		// Ignore invalid keys
	}
}

// promptOptionFallback uses standard input with Enter key
func promptOptionFallback(options []Option, keyMap map[string]*Option) (string, *Option, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", nil, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)

	// Empty input (just Enter) selects default if available
	if input == "" {
		if defaultOpt := findDefaultOption(options); defaultOpt != nil {
			return defaultOpt.Option, defaultOpt, nil
		}
		return "", nil, fmt.Errorf("no selection made and no default option")
	}

	// Case-sensitive lookup
	if opt, ok := keyMap[input]; ok {
		return opt.Option, opt, nil
	}

	return "", nil, fmt.Errorf("invalid selection: %s", input)
}

// promptText displays a message and reads text input (requires Enter)
// Returns the entered text
func promptText(message string) (string, error) {
	fmt.Println(message)
	fmt.Print(color.CyanString("> "))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return strings.TrimSpace(input), nil
}

// promptTextWithDefault displays a message with a grayed default, reads text input
// If user just presses Enter, returns the default value
func promptTextWithDefault(message, defaultValue string) (string, error) {
	grayDefault := color.New(color.Faint).Sprint(defaultValue)
	fmt.Printf("%s %s ", message, color.CyanString("[%s]:", grayDefault))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}
	return input, nil
}

// MultiSelectItem represents an item in a multi-select list
type MultiSelectItem struct {
	Label    string
	Value    string
	Selected bool
}

// promptMultiSelect displays a navigable multi-select list
// Navigation: arrow keys or hjkl, space to toggle, enter to confirm, q to cancel
// Returns selected items and whether user confirmed (false if cancelled)
func promptMultiSelect(title string, items []MultiSelectItem) ([]MultiSelectItem, bool, error) {
	if len(items) == 0 {
		return nil, false, fmt.Errorf("no items provided")
	}

	cursor := 0
	totalLines := len(items) + 2 // title + items + help line

	// Print initial display
	printMultiSelect(title, items, cursor)

	for {
		// Get terminal into raw mode for reading input
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return nil, false, fmt.Errorf("failed to set raw mode: %w", err)
		}

		// Read input
		b := make([]byte, 3) // Up to 3 bytes for arrow keys
		n, err := os.Stdin.Read(b)

		// Restore terminal before processing
		term.Restore(int(os.Stdin.Fd()), oldState)

		if err != nil {
			return nil, false, fmt.Errorf("failed to read input: %w", err)
		}

		needsRedraw := false

		// Handle input
		if n == 1 {
			switch b[0] {
			case 'q', 27, 3: // q, ESC, or Ctrl+C
				fmt.Println()
				return nil, false, nil
			case '\r', '\n': // Enter
				fmt.Println()
				var selected []MultiSelectItem
				for _, item := range items {
					if item.Selected {
						selected = append(selected, item)
					}
				}
				return selected, true, nil
			case ' ': // Space - toggle selection
				items[cursor].Selected = !items[cursor].Selected
				needsRedraw = true
			case 'k': // Vim up
				if cursor > 0 {
					cursor--
					needsRedraw = true
				}
			case 'j': // Vim down
				if cursor < len(items)-1 {
					cursor++
					needsRedraw = true
				}
			}
		} else if n == 3 && b[0] == 27 && b[1] == 91 {
			// Arrow keys: ESC [ A/B/C/D
			switch b[2] {
			case 65: // Up arrow
				if cursor > 0 {
					cursor--
					needsRedraw = true
				}
			case 66: // Down arrow
				if cursor < len(items)-1 {
					cursor++
					needsRedraw = true
				}
			}
		}

		if needsRedraw {
			redrawMultiSelect(title, items, cursor, totalLines)
		}
	}
}

// printMultiSelect prints the initial multi-select display
func printMultiSelect(title string, items []MultiSelectItem, cursor int) {
	fmt.Println(title)
	for i, item := range items {
		prefix := "  "
		if i == cursor {
			prefix = color.CyanString("> ")
		}
		checkbox := "[ ]"
		if item.Selected {
			checkbox = color.GreenString("[x]")
		}
		fmt.Printf("%s%s %s\n", prefix, checkbox, item.Label)
	}
	fmt.Print(color.New(color.Faint).Sprint("↑/k up  ↓/j down  space select  enter confirm  q cancel"))
}

// redrawMultiSelect redraws the multi-select display (called with terminal in normal mode)
func redrawMultiSelect(title string, items []MultiSelectItem, cursor int, totalLines int) {
	// Move cursor up to the title line and clear to end of screen
	// We're on the help line (no newline after it), so go up totalLines-1
	fmt.Printf("\033[%dF", totalLines-1) // Move to beginning of line N lines up
	fmt.Print("\033[J")                  // Clear from cursor to end of screen

	// Redraw everything
	fmt.Println(title)
	for i, item := range items {
		prefix := "  "
		if i == cursor {
			prefix = color.CyanString("> ")
		}
		checkbox := "[ ]"
		if item.Selected {
			checkbox = color.GreenString("[x]")
		}
		fmt.Printf("%s%s %s\n", prefix, checkbox, item.Label)
	}
	fmt.Print(color.New(color.Faint).Sprint("↑/k up  ↓/j down  space select  enter confirm  q cancel"))
}
