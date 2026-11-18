package cmd

import (
	"bytes"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nxkh4ng/gatekeep/core"
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

func TestAddDuplicateFlagRejected(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{passwords: [][]byte{[]byte(testMasterPassword)}}

	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"add", "gmail", "--username", "alice@gmail.com"})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(io.Discard)

	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Execute() error = %v, want a duplicate error", err)
	}
	if !strings.Contains(err.Error(), "gatekeep edit") {
		t.Fatalf("Execute() error = %v, want an edit hint", err)
	}
	if out.String() != "" {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
	if want := []string{"Master password: "}; !slices.Equal(p.prompts, want) {
		t.Fatalf("prompts = %q, want %q (a flag duplicate must not re-prompt)", p.prompts, want)
	}
	if entries := readTestEntries(t, path); len(entries) != len(testEntries) {
		t.Fatalf("entries = %d, want %d", len(entries), len(testEntries))
	}
}

func TestAddDuplicateReprompt(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{
		passwords: [][]byte{[]byte(testMasterPassword), []byte("new-secret")},
		lines:     []string{"alice@gmail.com", "alice2@gmail.com", "", ""},
	}

	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"add", "Gmail"})
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Added Gmail (alice2@gmail.com)\n"; out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
	if !strings.Contains(errOut.String(), "already exists") {
		t.Fatalf("stderr = %q, want a duplicate warning", errOut.String())
	}
	wantPrompts := []string{
		"Master password: ",
		"Username: ",
		"Username: ",
		"Password: ",
		"URL (optional): ",
		"Notes (optional): ",
	}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}

	entries := readTestEntries(t, path)
	if len(entries) != len(testEntries)+1 {
		t.Fatalf("entries = %d, want %d", len(entries), len(testEntries)+1)
	}
	if last := entries[len(entries)-1]; last.Username != "alice2@gmail.com" {
		t.Fatalf("added username = %q, want alice2@gmail.com", last.Username)
	}
}

func TestAddNotDuplicate(t *testing.T) {
	cases := []struct {
		name     string
		title    string
		username string
	}{
		{"same username", "GitHub", "alice@gmail.com"},
		{"same title", "Gmail", "bob@gmail.com"},
		{"username case differs", "Gmail", "ALICE@gmail.com"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTestVaultWithEntries(t)
			p := &fakePrompter{
				passwords: [][]byte{[]byte(testMasterPassword), []byte("new-secret")},
				lines:     []string{"", ""},
			}

			root := newRoot(&app{vault: path, prompter: p})
			root.SetArgs([]string{"add", tc.title, "--username", tc.username})
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)

			if err := root.Execute(); err != nil {
				t.Fatalf("Execute: %v", err)
			}
			wantPrompts := []string{
				"Master password: ",
				"Password: ",
				"URL (optional): ",
				"Notes (optional): ",
			}
			if !slices.Equal(p.prompts, wantPrompts) {
				t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
			}
			if entries := readTestEntries(t, path); len(entries) != len(testEntries)+1 {
				t.Fatalf("entries = %d, want %d", len(entries), len(testEntries)+1)
			}
		})
	}
}

func TestFindDuplicate(t *testing.T) {
	entries := []core.Entry{
		{Title: "Gmail", Username: "alice@gmail.com"},
		{Title: "YouTube", Username: "alice@gmail.com"},
	}
	cases := []struct {
		name     string
		title    string
		username string
		want     bool
	}{
		{"exact", "Gmail", "alice@gmail.com", true},
		{"title case-insensitive", "gmail", "alice@gmail.com", true},
		{"username case differs", "Gmail", "ALICE@gmail.com", false},
		{"different username", "Gmail", "bob@gmail.com", false},
		{"different title", "GitHub", "alice@gmail.com", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := findDuplicate(entries, tc.title, tc.username)
			if ok != tc.want {
				t.Fatalf("findDuplicate(%q, %q) = %+v, %v; want found %v", tc.title, tc.username, got, ok, tc.want)
			}
			if ok && got.Username != tc.username {
				t.Fatalf("findDuplicate returned %+v, want the matching entry", got)
			}
		})
	}

	if _, ok := findDuplicate(nil, "Gmail", "alice@gmail.com"); ok {
		t.Fatal("findDuplicate on an empty vault reported a match")
	}
}
