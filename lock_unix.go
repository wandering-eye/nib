//go:build !windows

package nib

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// tryLockFile takes an exclusive lock without blocking. It reports false when
// another process holds the lock, and returns an error only when locking
// itself failed, so a live lock can never be mistaken for an absent one.
func tryLockFile(file *os.File) (bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)

	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, unix.EWOULDBLOCK):
		return false, nil
	default:
		return false, err
	}
}

func unlockFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
