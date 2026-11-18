package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrVaultInUse reports that another process holds the vault lock.
var ErrVaultInUse = errors.New("vault is in use")

// FileLock is an exclusive advisory lock for one vault.
// The lock is released by Close or when the process exits.
type FileLock struct {
	f *os.File
}

// LockFile acquires the exclusive lock that serializes writes to the vault at
// vaultPath. The lock uses a separate <vault>.lock file, so renaming the vault
// does not invalidate it. It fails with ErrVaultInUse instead of waiting when
// another process holds the lock.
func LockFile(vaultPath string) (*FileLock, error) {
	lockPath := vaultPath + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return nil, fmt.Errorf("creating vault lock directory: %w", err)
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening vault lock %s: %w", lockPath, err)
	}

	if err := lockFile(f); err != nil {
		_ = f.Close()
		if isLockBusy(err) {
			return nil, fmt.Errorf("%w (lock file: %s)", ErrVaultInUse, lockPath)
		}
		return nil, fmt.Errorf("locking %s: %w", lockPath, err)
	}
	return &FileLock{f: f}, nil
}

// Close releases the lock. It is safe to call more than once.
func (l *FileLock) Close() error {
	if l == nil || l.f == nil {
		return nil
	}
	f := l.f
	l.f = nil
	return errors.Join(unlockFile(f), f.Close())
}
