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
