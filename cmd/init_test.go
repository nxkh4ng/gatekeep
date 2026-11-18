package cmd

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nxkh4ng/gatekeep/core"
)

func TestInitCreatesVault(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested")
	path := filepath.Join(dir, "vault.gk")
	const password = "correct horse battery staple"
	p := &fakePrompter{passwords: [][]byte{[]byte(password), []byte(password)}}

	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"init"})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(io.Discard)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Vault created at " + path + "\n"; out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
	if len(p.prompts) != 2 || p.prompts[0] != "Master password: " || p.prompts[1] != "Repeat master password: " {
		t.Fatalf("prompts = %q", p.prompts)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat vault: %v", err)
	}

	// Windows ignores Unix file modes and reports 0666/0777 instead.
	if runtime.GOOS != "windows" {
		if perm := fi.Mode().Perm(); perm != 0o600 {
			t.Fatalf("vault mode = %#o, want 0600", perm)
		}
		di, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat vault directory: %v", err)
		}
		if perm := di.Mode().Perm(); perm != 0o700 {
			t.Fatalf("vault directory mode = %#o, want 0700", perm)
		}
	}

	v, err := core.LoadFile(path, []byte(password))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	entries, err := v.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("vault has %d entries, want 0", len(entries))
	}
}

func TestInitRefusesExistingVault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.gk")
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	p := &fakePrompter{}

	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"init"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Execute() error = %v, want an already-exists error", err)
	}
	if len(p.prompts) != 0 {
		t.Fatalf("prompted for a password: %q", p.prompts)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(raw) != "existing" {
		t.Fatalf("existing vault was modified: %q", raw)
	}
}

func TestInitRejectsMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.gk")
	p := &fakePrompter{passwords: [][]byte{[]byte("first"), []byte("second")}}

	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"init"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "do not match") {
		t.Fatalf("Execute() error = %v, want a mismatch error", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("vault file exists after a mismatch: %v", err)
	}
}

func TestInitRetriesEmptyPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.gk")
	const password = "correct horse battery staple"
	p := &fakePrompter{passwords: [][]byte{nil, []byte(password), []byte(password)}}

	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"init"})
	root.SetOut(io.Discard)
	var errOut bytes.Buffer
	root.SetErr(&errOut)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(p.prompts) != 3 {
		t.Fatalf("prompts = %q, want 3 entries", p.prompts)
	}
	if !strings.Contains(errOut.String(), "cannot be empty") {
		t.Fatalf("stderr = %q, want an empty-password message", errOut.String())
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat vault: %v", err)
	}
}

func TestInitPrompterError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.gk")
	p := &fakePrompter{err: errors.New("no terminal")}

	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"init"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "no terminal") {
		t.Fatalf("Execute() error = %v, want the prompter error", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("vault file exists after a prompter error: %v", err)
	}
}
