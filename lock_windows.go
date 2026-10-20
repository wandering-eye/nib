//go:build windows

package nib

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// wholeFile is the byte range the lock covers. The lock file carries no
// content: it is a token, so the range is the largest one there is.
const wholeFile = ^uint32(0)

// tryLockFile takes an exclusive lock without blocking. It reports false when
// another process holds the lock, and returns an error only when locking
// itself failed, so a live lock can never be mistaken for an absent one.
func tryLockFile(file *os.File) (bool, error) {
	err := windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		wholeFile,
		wholeFile,
		new(windows.Overlapped),
	)

	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, windows.ERROR_LOCK_VIOLATION):
		return false, nil
	default:
		return false, err
	}
}

func unlockFile(file *os.File) error {
	return windows.UnlockFileEx(
		windows.Handle(file.Fd()),
		0,
		wholeFile,
		wholeFile,
		new(windows.Overlapped),
	)
}
