//go:build windows

package flock

// Lock is a no-op on Windows. File locking is advisory and best-effort;
// on Windows the lock file's existence provides basic mutual exclusion.
func Lock(fd uintptr) error {
	return nil
}

// Unlock is a no-op on Windows.
func Unlock(fd uintptr) error {
	return nil
}
