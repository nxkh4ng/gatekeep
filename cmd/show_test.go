package cmd

import (
	"bytes"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func runShowCommand(t *testing.T, path string, p *fakePrompter, query string) (stdout, stderr string, err error) {
	t.Helper()
	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs([]string{"show", query})
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	err = root.Execute()
	return out.String(), errOut.String(), err
}

func TestShowUniqueTitle(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{passwords: [][]byte{[]byte(testMasterPassword)}}

	out, errOut, err := runShowCommand(t, path, p, "gmail")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty", errOut)
	}
	for _, want := range []string{"Gmail", "alice@gmail.com", "mail-secret", "https://mail.google.com"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output = %q, want %q", out, want)
		}
	}
	if strings.Contains(out, "Notes:") {
		t.Fatalf("output contains an empty notes field: %q", out)
	}
	if want := []string{"Master password: "}; !slices.Equal(p.prompts, want) {
		t.Fatalf("prompts = %q, want %q", p.prompts, want)
	}
}

func TestShowPicker(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{
		passwords: [][]byte{[]byte(testMasterPassword)},
		lines:     []string{"9", "2"},
	}

	out, errOut, err := runShowCommand(t, path, p, "YouTube")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "bob@gmail.com") || !strings.Contains(out, "bob-secret") {
		t.Fatalf("output = %q, want the selected bob entry", out)
	}
	if strings.Contains(out, "alice-secret") {
		t.Fatalf("output leaked the other account: %q", out)
	}
	if strings.Contains(out, "URL:") || strings.Contains(out, "Notes:") {
		t.Fatalf("output contains empty optional fields: %q", out)
	}
	for _, want := range []string{"1)", "2)", "alice@gmail.com", "bob@gmail.com", "Invalid selection."} {
		if !strings.Contains(errOut, want) {
			t.Fatalf("stderr = %q, want %q", errOut, want)
		}
	}
	wantPrompts := []string{"Master password: ", "Select [1-2, q]: ", "Select [1-2, q]: "}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}
}

func TestShowSubstringUnique(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{passwords: [][]byte{[]byte(testMasterPassword)}}

	out, _, err := runShowCommand(t, path, p, "ban")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "bank-secret") {
		t.Fatalf("output = %q, want the Bank entry", out)
	}
	if want := []string{"Master password: "}; !slices.Equal(p.prompts, want) {
		t.Fatalf("prompts = %q, want %q", p.prompts, want)
	}
}

func TestShowNoMatch(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{passwords: [][]byte{[]byte(testMasterPassword)}}

	out, _, err := runShowCommand(t, path, p, "nothing")
	if err == nil || !strings.Contains(err.Error(), "no entry matches") {
		t.Fatalf("Execute() error = %v, want a no-match error", err)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty", out)
	}
}

func TestShowAbortPicker(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{
		passwords: [][]byte{[]byte(testMasterPassword)},
		lines:     []string{"q"},
	}

	out, _, err := runShowCommand(t, path, p, "YouTube")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Aborted.\n"; out != want {
		t.Fatalf("output = %q, want %q", out, want)
	}
	wantPrompts := []string{"Master password: ", "Select [1-2, q]: "}
	if !slices.Equal(p.prompts, wantPrompts) {
		t.Fatalf("prompts = %q, want %q", p.prompts, wantPrompts)
	}
}

func TestShowEmptyQuery(t *testing.T) {
	p := &fakePrompter{}
	path := filepath.Join(t.TempDir(), "vault.gk")

	_, _, err := runShowCommand(t, path, p, "  ")
	if err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("Execute() error = %v, want a query-required error", err)
	}
	if len(p.prompts) != 0 {
		t.Fatalf("prompted before validating the query: %q", p.prompts)
	}
}
