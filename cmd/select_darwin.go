//go:build darwin

package cmd

import "syscall"

func sysSelect(nfd int, r *syscall.FdSet, w *syscall.FdSet, e *syscall.FdSet, timeout *syscall.Timeval) error {
	return syscall.Select(nfd, r, w, e, timeout)
}
