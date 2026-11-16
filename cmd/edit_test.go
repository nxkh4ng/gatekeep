package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nxkh4ng/gatekeep/core"
)

func runEditCommand(t *testing.T, path string, p *fakePrompter, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs(append([]string{"edit"}, args...))
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	err = root.Execute()
	return out.String(), errOut.String(), err
}

func findTestEntry(t *testing.T, entries []core.Entry, title, username string) core.Entry {
	t.Helper()
	for _, e := range entries {
		if e.Title == title && e.Username == username {
			return e
		}
	}
	t.Fatalf("entry %q (%q) not found", title, username)
	return core.Entry{}
}

func TestEditWizard(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{
		passwords: [][]byte{[]byte(testMasterPassword), nil},
		lines:     []string{"", "alice2@gmail.com", "", ""},
	}

	out, errOut, err := runEditCommand(t, path, p, "Gmail")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty", errOut)
	}
	if want := "Updated Gmail\n"; out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
	wantPrompts := []string{
		"Master password: ",
		"Title [Gmail]: ",
		"Username [alice@gmail.com]: ",
		"Password (Enter to keep): ",
		"URL [https://mail.google.com]: ",
		"Notes: ",
	}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}

	got := findTestEntry(t, readTestEntries(t, path), "Gmail", "alice2@gmail.com")
	if got.Password != "mail-secret" || got.URL != "https://mail.google.com" {
		t.Fatalf("entry = %+v", got)
	}
}

func TestEditFlags(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{passwords: [][]byte{[]byte(testMasterPassword), []byte("new-secret")}}

	out, errOut, err := runEditCommand(t, path, p, "Gmail",
		"--username", "alice2@gmail.com", "--url=", "--password")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty", errOut)
	}
	if want := "Updated Gmail\n"; out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
	wantPrompts := []string{"Master password: ", "New password (Enter to keep): "}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}

	got := findTestEntry(t, readTestEntries(t, path), "Gmail", "alice2@gmail.com")
	if got.Password != "new-secret" || got.URL != "" {
		t.Fatalf("entry = %+v", got)
	}
}

func TestEditWizardNoChanges(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	p := &fakePrompter{
		passwords: [][]byte{[]byte(testMasterPassword), nil},
		lines:     []string{"", "", "", ""},
	}

	out, _, err := runEditCommand(t, path, p, "Gmail")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "No changes.\n"; out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("edit rewrote the vault without any change")
	}
}

func TestEditPicker(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{
		passwords: [][]byte{[]byte(testMasterPassword)},
		lines:     []string{"1"},
	}

	out, errOut, err := runEditCommand(t, path, p, "YouTube", "--username", "alice2@gmail.com")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Updated YouTube\n"; out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
	if !strings.Contains(errOut, "1)") || !strings.Contains(errOut, "2)") {
		t.Fatalf("stderr = %q, want the picker list", errOut)
	}
	wantPrompts := []string{"Master password: ", "Select [1-2, q]: "}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}

	entries := readTestEntries(t, path)
	if got := findTestEntry(t, entries, "YouTube", "alice2@gmail.com"); got.Password != "alice-secret" {
		t.Fatalf("edited entry = %+v", got)
	}
	if got := findTestEntry(t, entries, "YouTube", "bob@gmail.com"); got.Password != "bob-secret" {
		t.Fatalf("other entry = %+v", got)
	}
}

func TestEditTitleRequired(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	p := &fakePrompter{passwords: [][]byte{[]byte(testMasterPassword)}}

	_, _, err = runEditCommand(t, path, p, "Gmail", "--title=")
	if err == nil || !strings.Contains(err.Error(), "title is required") {
		t.Fatalf("Execute() error = %v, want a title-required error", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("failed edit rewrote the vault")
	}
}

func TestEditEmptyQuery(t *testing.T) {
	p := &fakePrompter{}
	path := filepath.Join(t.TempDir(), "vault.gk")

	_, _, err := runEditCommand(t, path, p, "  ")
	if err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("Execute() error = %v, want a query-required error", err)
	}
	if len(p.prompts) != 0 {
		t.Fatalf("prompted before validating the query: %q", p.prompts)
	}
}
