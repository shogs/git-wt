//go:build linux

package cmd

import (
	"syscall"
	"time"
)

func pollStdinReady(fd int, timeout time.Duration) bool {
	var readfds syscall.FdSet
	readfds.Bits[fd/64] |= 1 << (uint(fd) % 64)
	tv := syscall.Timeval{Sec: 0, Usec: timeout.Microseconds()}

	_, err := syscall.Select(fd+1, &readfds, nil, nil, &tv)
	if err != nil {
		return false
	}
	return readfds.Bits[fd/64]&(1<<(uint(fd)%64)) != 0
}
