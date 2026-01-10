//go:build !windows

package cmd

import "syscall"

// sigwinch is the window resize signal (Unix-only)
var sigwinch = syscall.SIGWINCH
