package cmd

import (
	"bytes"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestAddWithPrompts(t *testing.T) {
	path := writeTestVault(t)
	p := &fakePrompter{
		passwords: [][]byte{[]byte(testMasterPassword), nil, []byte("s3cret")},
		lines:     []string{"", "YouTube", "alice@gmail.com", "https://youtube.com", "main account"},
	}

	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"add"})
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Added YouTube (alice@gmail.com)\n"; out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}

	wantPrompts := []string{
		"Master password: ",
		"Title: ",
		"Title: ",
		"Username: ",
		"Password: ",
		"Password: ",
		"URL (optional): ",
		"Notes (optional): ",
	}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}
	for _, msg := range []string{"Title cannot be empty.", "Password cannot be empty."} {
		if !strings.Contains(errOut.String(), msg) {
			t.Fatalf("stderr = %q, want %q", errOut.String(), msg)
		}
	}

	entries := readTestEntries(t, path)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	got := entries[0]
	if got.Title != "YouTube" || got.Username != "alice@gmail.com" || got.Password != "s3cret" ||
		got.URL != "https://youtube.com" || got.Notes != "main account" {
		t.Fatalf("entry = %+v", got)
	}
}

func TestAddUsesFlags(t *testing.T) {
	path := writeTestVault(t)
	p := &fakePrompter{passwords: [][]byte{[]byte(testMasterPassword), []byte("s3cret")}}

	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{
		"add", "YouTube",
		"--username", "alice@gmail.com",
		"--url", "https://youtube.com",
		"--notes", "main account",
	})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(io.Discard)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	wantPrompts := []string{"Master password: ", "Password: "}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}

	entries := readTestEntries(t, path)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	got := entries[0]
	if got.Title != "YouTube" || got.Username != "alice@gmail.com" || got.Password != "s3cret" ||
		got.URL != "https://youtube.com" || got.Notes != "main account" {
		t.Fatalf("entry = %+v", got)
	}
}

func TestAddOptionalFieldsSkipped(t *testing.T) {
	path := writeTestVault(t)
	p := &fakePrompter{
		passwords: [][]byte{[]byte(testMasterPassword), []byte("s3cret")},
		lines:     []string{"", ""},
	}

	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"add", "YouTube", "--username", "alice@gmail.com"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	wantPrompts := []string{"Master password: ", "Password: ", "URL (optional): ", "Notes (optional): "}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}

	entries := readTestEntries(t, path)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].URL != "" || entries[0].Notes != "" {
		t.Fatalf("optional fields = %q, %q, want empty", entries[0].URL, entries[0].Notes)
	}
}

func TestAddMissingVault(t *testing.T) {
	p := &fakePrompter{}
	root := newRoot(&app{vault: filepath.Join(t.TempDir(), "missing.gk"), prompter: p})
	root.SetArgs([]string{"add"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "vault not found") {
		t.Fatalf("Execute() error = %v, want a vault-not-found error", err)
	}
	if len(p.prompts) != 0 {
		t.Fatalf("prompted before finding the vault: %q", p.prompts)
	}
}

func TestAddUsernameRequired(t *testing.T) {
	path := writeTestVault(t)
	p := &fakePrompter{passwords: [][]byte{[]byte(testMasterPassword)}}

	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"add", "YouTube", "--username="})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "username is required") {
		t.Fatalf("Execute() error = %v, want a username-required error", err)
	}
	if want := []string{"Master password: "}; !slices.Equal(p.prompts, want) {
		t.Fatalf("prompts = %q, want %q", p.prompts, want)
	}
}
