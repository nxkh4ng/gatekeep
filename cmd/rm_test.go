package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func runRmCommand(t *testing.T, path string, p *fakePrompter, query string) (stdout, stderr string, err error) {
	t.Helper()
	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"rm", query})
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	err = root.Execute()
	return out.String(), errOut.String(), err
}

func TestRemoveConfirmed(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{
		passwords: [][]byte{[]byte(testMasterPassword)},
		lines:     []string{"YES"},
	}

	out, errOut, err := runRmCommand(t, path, p, "Gmail")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty", errOut)
	}
	if want := "Deleted Gmail\n"; out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
	wantPrompts := []string{
		"Master password: ",
		`Delete "Gmail" (alice@gmail.com)? [y/N] `,
	}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}

	entries := readTestEntries(t, path)
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	for _, e := range entries {
		if e.Title == "Gmail" {
			t.Fatalf("Gmail still exists: %+v", e)
		}
	}
}

func TestRemoveDeclined(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	p := &fakePrompter{
		passwords: [][]byte{[]byte(testMasterPassword)},
		lines:     []string{""},
	}

	out, _, err := runRmCommand(t, path, p, "Gmail")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Aborted.\n"; out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("declined delete rewrote the vault")
	}
}

func TestRemoveAbortPicker(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	p := &fakePrompter{
		passwords: [][]byte{[]byte(testMasterPassword)},
		lines:     []string{"q"},
	}

	out, errOut, err := runRmCommand(t, path, p, "YouTube")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Aborted.\n"; out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
	if !strings.Contains(errOut, "1)") || !strings.Contains(errOut, "2)") {
		t.Fatalf("stderr = %q, want the picker list", errOut)
	}
	wantPrompts := []string{"Master password: ", "Select [1-2, q]: "}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("aborted delete rewrote the vault")
	}
}

func TestRemovePicker(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{
		passwords: [][]byte{[]byte(testMasterPassword)},
		lines:     []string{"2", "y"},
	}

	out, errOut, err := runRmCommand(t, path, p, "YouTube")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Deleted YouTube\n"; out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
	if !strings.Contains(errOut, "1)") || !strings.Contains(errOut, "2)") {
		t.Fatalf("stderr = %q, want the picker list", errOut)
	}
	wantPrompts := []string{
		"Master password: ",
		"Select [1-2, q]: ",
		`Delete "YouTube" (bob@gmail.com)? [y/N] `,
	}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}

	entries := readTestEntries(t, path)
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	if got := findTestEntry(t, entries, "YouTube", "alice@gmail.com"); got.Password != "alice-secret" {
		t.Fatalf("alice entry = %+v", got)
	}
	for _, e := range entries {
		if e.Username == "bob@gmail.com" {
			t.Fatalf("bob entry still exists: %+v", e)
		}
	}
}

func TestRemoveNoMatch(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{passwords: [][]byte{[]byte(testMasterPassword)}}

	out, _, err := runRmCommand(t, path, p, "nothing")
	if err == nil || !strings.Contains(err.Error(), "no entry matches") {
		t.Fatalf("Execute() error = %v, want a no-match error", err)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty", out)
	}
	if want := []string{"Master password: "}; !slices.Equal(p.prompts, want) {
		t.Fatalf("prompts = %q, want %q", p.prompts, want)
	}
}

func TestRemoveEmptyQuery(t *testing.T) {
	p := &fakePrompter{}
	path := filepath.Join(t.TempDir(), "vault.gk")

	_, _, err := runRmCommand(t, path, p, "  ")
	if err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("Execute() error = %v, want a query-required error", err)
	}
	if len(p.prompts) != 0 {
		t.Fatalf("prompted before validating the query: %q", p.prompts)
	}
}
