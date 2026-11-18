package cmd

import (
	"bytes"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nxkh4ng/gatekeep/core"
)

func runShowCommand(t *testing.T, path string, p *fakePrompter, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs(append([]string{"show"}, args...))
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
	for _, want := range []string{"Gmail", "alice@gmail.com", maskedPassword, "https://mail.google.com"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output = %q, want %q", out, want)
		}
	}
	if strings.Contains(out, "mail-secret") {
		t.Fatalf("output leaked the password: %q", out)
	}
	if !strings.Contains(errOut, "--reveal") {
		t.Fatalf("stderr = %q, want a --reveal hint", errOut)
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
	if !strings.Contains(out, "bob@gmail.com") || !strings.Contains(out, maskedPassword) {
		t.Fatalf("output = %q, want the masked bob entry", out)
	}
	if strings.Contains(out, "alice-secret") || strings.Contains(out, "bob-secret") {
		t.Fatalf("output leaked a password: %q", out)
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
	if !strings.Contains(out, "Bank") || !strings.Contains(out, maskedPassword) {
		t.Fatalf("output = %q, want the masked Bank entry", out)
	}
	if strings.Contains(out, "bank-secret") {
		t.Fatalf("output leaked the password: %q", out)
	}
	if want := []string{"Master password: "}; !slices.Equal(p.prompts, want) {
		t.Fatalf("prompts = %q, want %q", p.prompts, want)
	}
}

func TestShowReveal(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	p := &fakePrompter{passwords: [][]byte{[]byte(testMasterPassword)}}

	out, errOut, err := runShowCommand(t, path, p, "gmail", "--reveal")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "mail-secret") {
		t.Fatalf("output = %q, want the revealed password", out)
	}
	if strings.Contains(out, maskedPassword) {
		t.Fatalf("output masked a revealed password: %q", out)
	}
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty", errOut)
	}
}

func TestPrintEntryEmptyPassword(t *testing.T) {
	root := newRoot(&app{vault: "unused"})
	var out bytes.Buffer
	root.SetOut(&out)

	if err := printEntry(root, core.Entry{Title: "Blank", Username: "alice"}, false); err != nil {
		t.Fatalf("printEntry: %v", err)
	}
	if strings.Contains(out.String(), maskedPassword) {
		t.Fatalf("output masked an empty password: %q", out.String())
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
