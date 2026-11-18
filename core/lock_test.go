package core

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLockFileExclusive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.gk")

	first, err := LockFile(path)
	if err != nil {
		t.Fatalf("LockFile: %v", err)
	}

	if _, err := LockFile(path); !errors.Is(err, ErrVaultInUse) {
		t.Fatalf("second LockFile error = %v, want ErrVaultInUse", err)
	} else if !strings.Contains(err.Error(), path+".lock") {
		t.Fatalf("second LockFile error = %q, want the lock path", err)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	second, err := LockFile(path)
	if err != nil {
		t.Fatalf("LockFile after release: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("Close second: %v", err)
	}
}

func TestLockFileCloseTwice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.gk")
	lock, err := LockFile(path)
	if err != nil {
		t.Fatalf("LockFile: %v", err)
	}
	if err := lock.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := lock.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestLockFileCreatesLock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested")
	path := filepath.Join(dir, "vault.gk")

	lock, err := LockFile(path)
	if err != nil {
		t.Fatalf("LockFile: %v", err)
	}

	lockPath := path + ".lock"
	fi, err := os.Stat(lockPath)
	if err != nil {
		t.Fatalf("stat lock file: %v", err)
	}

	// Windows ignores Unix file modes and reports 0666/0777 instead.
	if runtime.GOOS != "windows" {
		di, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat lock directory: %v", err)
		}
		if perm := di.Mode().Perm(); perm != 0o700 {
			t.Fatalf("lock directory mode = %#o, want 0700", perm)
		}
		if perm := fi.Mode().Perm(); perm != 0o600 {
			t.Fatalf("lock file mode = %#o, want 0600", perm)
		}
	}

	if err := lock.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock file was removed on Close: %v", err)
	}
}
