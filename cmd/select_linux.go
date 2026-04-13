//go:build linux

package cmd

import "syscall"

func sysSelect(nfd int, r *syscall.FdSet, w *syscall.FdSet, e *syscall.FdSet, timeout *syscall.Timeval) error {
	_, err := syscall.Select(nfd, r, w, e, timeout)
	return err
}
