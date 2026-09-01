package dstask

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// lockFileName sits in the repository directory rather than inside .git,
// because it must be takeable before git init has run. It is untracked and the
// commit pathspec excludes it, so it never reaches history. It is never
// deleted either: unlinking a lock file another process still holds open would
// let a third process lock the new inode and believe it holds the same lock.
const lockFileName = ".nib.lock"

const (
	lockPollInterval = 20 * time.Millisecond
	lockTimeout      = 30 * time.Second
)

// repositoryLock is an exclusive lock over one nib repository, held for the
// whole of allocate -> write -> commit. The lock is kernel held, so a process
// that dies releases it and a lock file left behind by a crash is not a stale
// lock at all; a lock that is genuinely held is waited for, never bypassed.
type repositoryLock struct {
	file *os.File
}

func acquireRepositoryLock(repoPath string) (*repositoryLock, error) {
	lockPath := filepath.Join(repoPath, lockFileName)

	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %v", lockPath, err)
	}

	deadline := time.Now().Add(lockTimeout)

	for {
		held, err := tryLockFile(file)
		if err != nil {
			_ = file.Close()

			return nil, fmt.Errorf("cannot lock %s: %v", lockPath, err)
		}

		if held {
			return &repositoryLock{file: file}, nil
		}

		if time.Now().After(deadline) {
			_ = file.Close()

			return nil, fmt.Errorf(
				"another nib process has held %s for more than %s",
				lockPath,
				lockTimeout,
			)
		}

		time.Sleep(lockPollInterval)
	}
}

func (lock *repositoryLock) release() error {
	if err := unlockFile(lock.file); err != nil {
		_ = lock.file.Close()

		return fmt.Errorf("cannot release the repository lock: %v", err)
	}

	if err := lock.file.Close(); err != nil {
		return fmt.Errorf("cannot close the repository lock: %v", err)
	}

	return nil
}

// withLock runs work under the repository lock and releases it on every path
// out, including a panic, so that a failed command never leaves the repository
// locked against the next one.
// LockRepo takes the repository lock and returns the release. The lock is held
// by the kernel, so a process that exits without releasing it still frees it.
func LockRepo(repoPath string) (func(), error) {
	lock, err := acquireRepositoryLock(repoPath)
	if err != nil {
		return nil, err
	}

	return func() { _ = lock.release() }, nil
}
