//go:build unix

package core

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// lockFile takes a non-blocking exclusive flock on f.
func lockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}

// unlockFile releases the flock held on f.
func unlockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}

// isLockBusy reports whether err means another process holds the lock.
func isLockBusy(err error) bool {
	return errors.Is(err, unix.EWOULDBLOCK)
}
