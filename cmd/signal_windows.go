//go:build windows

package cmd

import "os"

// sigwinch is nil on Windows since SIGWINCH doesn't exist
var sigwinch os.Signal = nil
