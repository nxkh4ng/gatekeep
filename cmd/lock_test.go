package cmd

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nxkh4ng/gatekeep/core"
)

func holdVaultLock(t *testing.T, path string) {
	t.Helper()
	lock, err := core.LockFile(path)
	if err != nil {
		t.Fatalf("LockFile: %v", err)
	}
	t.Cleanup(func() { _ = lock.Close() })
}

func TestMutatingCommandsFailWhenLocked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.gk")
	holdVaultLock(t, path)

	tests := []struct {
		name string
		args []string
	}{
		{"init", []string{"init"}},
		{"add", []string{"add", "X"}},
		{"edit", []string{"edit", "X"}},
		{"rm", []string{"rm", "X"}},
		{"passwd", []string{"passwd"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &fakePrompter{}
			root := newRoot(&app{vault: path, prompter: p})
			root.SetArgs(tt.args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)

			err := root.Execute()
			if !errors.Is(err, core.ErrVaultInUse) {
				t.Fatalf("Execute() error = %v, want ErrVaultInUse", err)
			}
			if len(p.prompts) != 0 {
				t.Fatalf("prompted before reporting the lock: %q", p.prompts)
			}
			if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("vault file exists after a locked command: %v", err)
			}
		})
	}
}

func TestListWorksWhileVaultLocked(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	holdVaultLock(t, path)

	p := &fakePrompter{passwords: [][]byte{[]byte(testMasterPassword)}}
	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"list"})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(io.Discard)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "Gmail") {
		t.Fatalf("output = %q, want the seeded entries", out.String())
	}
}

func TestLockReleasedAfterCommandError(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{passwords: [][]byte{[]byte(testMasterPassword)}}
	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"add", "X", "--username="})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "username is required") {
		t.Fatalf("Execute() error = %v, want a username-required error", err)
	}

	lock, err := core.LockFile(path)
	if err != nil {
		t.Fatalf("LockFile after a command error: %v", err)
	}
	if err := lock.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
