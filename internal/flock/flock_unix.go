//go:build !windows

package flock

import "syscall"

// Lock acquires an exclusive advisory lock on the file descriptor.
func Lock(fd uintptr) error {
	return syscall.Flock(int(fd), syscall.LOCK_EX)
}

// Unlock releases the advisory lock on the file descriptor.
func Unlock(fd uintptr) error {
	return syscall.Flock(int(fd), syscall.LOCK_UN)
}
