package cmd

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/nxkh4ng/gatekeep/core"
)

const newMasterPassword = "a much better master password"

func runPasswdCommand(t *testing.T, path string, p *fakePrompter) (stdout, stderr string, err error) {
	t.Helper()
	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"passwd"})
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	err = root.Execute()
	return out.String(), errOut.String(), err
}

func TestPasswdChangesMasterPassword(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{passwords: [][]byte{
		[]byte(testMasterPassword),
		[]byte(newMasterPassword),
		[]byte(newMasterPassword),
	}}

	out, errOut, err := runPasswdCommand(t, path, p)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty", errOut)
	}
	if want := "Master password changed.\n"; out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
	wantPrompts := []string{
		"Master password: ",
		"New master password: ",
		"Repeat new master password: ",
	}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}

	if _, err := core.LoadFile(path, []byte(testMasterPassword)); !errors.Is(err, core.ErrInvalidPassword) {
		t.Fatalf("LoadFile with the old password: error = %v, want ErrInvalidPassword", err)
	}

	v, err := core.LoadFile(path, []byte(newMasterPassword))
	if err != nil {
		t.Fatalf("LoadFile with the new password: %v", err)
	}
	defer v.Lock()
	entries, err := v.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("entries = %d, want 4", len(entries))
	}
}

func TestPasswdMismatch(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	p := &fakePrompter{passwords: [][]byte{
		[]byte(testMasterPassword),
		[]byte("first"),
		[]byte("second"),
	}}

	_, _, err = runPasswdCommand(t, path, p)
	if err == nil || !strings.Contains(err.Error(), "do not match") {
		t.Fatalf("Execute() error = %v, want a mismatch error", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("mismatched passwd rewrote the vault")
	}
}

func TestPasswdRetriesEmptyNewPassword(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	p := &fakePrompter{passwords: [][]byte{
		[]byte(testMasterPassword),
		nil,
		[]byte(newMasterPassword),
		[]byte(newMasterPassword),
	}}

	out, errOut, err := runPasswdCommand(t, path, p)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Master password changed.\n"; out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
	if !strings.Contains(errOut, "Password cannot be empty.") {
		t.Fatalf("stderr = %q, want an empty-password message", errOut)
	}
	wantPrompts := []string{
		"Master password: ",
		"New master password: ",
		"New master password: ",
		"Repeat new master password: ",
	}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if bytes.Equal(before, after) {
		t.Fatal("passwd did not rewrite the vault")
	}
}
