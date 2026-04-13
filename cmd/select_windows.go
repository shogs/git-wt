//go:build windows

package cmd

import "time"

// pollStdinReady on Windows uses a simple sleep-based poll since
// syscall.Select and FdSet are not available.
func pollStdinReady(fd int, timeout time.Duration) bool {
	time.Sleep(timeout)
	return true
}
