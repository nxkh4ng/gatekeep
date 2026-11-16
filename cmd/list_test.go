package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runListCommand(t *testing.T, path string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	p := &fakePrompter{passwords: [][]byte{[]byte(testMasterPassword)}}
	root := newRoot(&app{vault: path, prompter: p})
	root.SetArgs(append([]string{"list"}, args...))
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	err = root.Execute()
	return out.String(), errOut.String(), err
}

func listOutput(t *testing.T, path string, args ...string) (stdout, stderr string) {
	t.Helper()
	stdout, stderr, err := runListCommand(t, path, args...)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return stdout, stderr
}

func TestListTable(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	out, errOut := listOutput(t, path)
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty", errOut)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("lines = %d (%q), want 5", len(lines), out)
	}
	header := strings.Fields(lines[0])
	if len(header) != 3 || header[0] != "TITLE" || header[1] != "USERNAME" || header[2] != "URL" {
		t.Fatalf("header = %q", lines[0])
	}

	want := []struct{ title, username string }{
		{"Bank", "Bob"},
		{"Gmail", "alice@gmail.com"},
		{"YouTube", "alice@gmail.com"},
		{"YouTube", "bob@gmail.com"},
	}
	for i, w := range want {
		fields := strings.Fields(lines[i+1])
		if len(fields) < 2 || fields[0] != w.title || fields[1] != w.username {
			t.Fatalf("row %d = %q, want %s %s", i, lines[i+1], w.title, w.username)
		}
	}
	for _, secret := range []string{"alice-secret", "bob-secret", "mail-secret", "bank-secret"} {
		if strings.Contains(out, secret) {
			t.Fatalf("table leaked password %q", secret)
		}
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("list modified the vault file")
	}
}

func TestListQuery(t *testing.T) {
	path := writeTestVaultWithEntries(t)

	out, errOut := listOutput(t, path, "YOUTUBE")
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty", errOut)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines = %d (%q), want header + 2 rows", len(lines), out)
	}
	if !strings.Contains(lines[1], "alice@gmail.com") || !strings.Contains(lines[2], "bob@gmail.com") {
		t.Fatalf("query rows = %q", lines)
	}
	if strings.Contains(out, "Gmail") || strings.Contains(out, "Bank") {
		t.Fatalf("query returned non-matching entries: %q", out)
	}

	t.Run("username", func(t *testing.T) {
		out, _ := listOutput(t, path, "alice@")
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("lines = %d (%q), want header + 2 rows", len(lines), out)
		}
		if !strings.Contains(lines[1], "Gmail") || !strings.Contains(lines[2], "YouTube") {
			t.Fatalf("username query rows = %q", lines)
		}
	})
}

func TestListJSON(t *testing.T) {
	path := writeTestVaultWithEntries(t)

	out, errOut := listOutput(t, path, "--json")
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty", errOut)
	}
	var got []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshalling JSON: %v", err)
	}

	wantTitles := []string{"Bank", "Gmail", "YouTube", "YouTube"}
	if len(got) != len(wantTitles) {
		t.Fatalf("entries = %d, want %d", len(got), len(wantTitles))
	}
	for i := range wantTitles {
		var title string
		if err := json.Unmarshal(got[i]["title"], &title); err != nil {
			t.Fatalf("entry %d title: %v", i, err)
		}
		if title != wantTitles[i] {
			t.Fatalf("entry %d title = %q, want %q", i, title, wantTitles[i])
		}
		if _, ok := got[i]["password"]; ok {
			t.Fatalf("entry %d unexpectedly includes a password", i)
		}
	}
}

func TestListJSONShowPasswords(t *testing.T) {
	path := writeTestVaultWithEntries(t)

	out, errOut := listOutput(t, path, "--json", "--show-passwords")
	if errOut != "" {
		t.Fatalf("stderr = %q, want empty", errOut)
	}
	var got []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshalling JSON: %v", err)
	}

	wantTitles := []string{"Bank", "Gmail", "YouTube", "YouTube"}
	wantPasswords := []string{"bank-secret", "mail-secret", "alice-secret", "bob-secret"}
	if len(got) != len(wantTitles) {
		t.Fatalf("entries = %d, want %d", len(got), len(wantTitles))
	}
	for i := range wantTitles {
		var title, password string
		if err := json.Unmarshal(got[i]["title"], &title); err != nil {
			t.Fatalf("entry %d title: %v", i, err)
		}
		if err := json.Unmarshal(got[i]["password"], &password); err != nil {
			t.Fatalf("entry %d password: %v", i, err)
		}
		if title != wantTitles[i] || password != wantPasswords[i] {
			t.Fatalf("entry %d = %q/%q, want %q/%q", i, title, password, wantTitles[i], wantPasswords[i])
		}
	}
}

func TestListShowPasswordsRequiresJSON(t *testing.T) {
	out, _, err := runListCommand(t, filepath.Join(t.TempDir(), "missing.gk"), "--show-passwords")
	if err == nil {
		t.Fatal("Execute: want error")
	}
	if !strings.Contains(err.Error(), "--show-passwords requires --json") {
		t.Fatalf("error = %q, want requires --json", err)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty", out)
	}
}

func TestListEmptyVault(t *testing.T) {
	path := writeTestVault(t)
	out, errOut := listOutput(t, path)
	if out != "" {
		t.Fatalf("stdout = %q, want empty", out)
	}
	if !strings.Contains(errOut, "Vault is empty.") {
		t.Fatalf("stderr = %q, want an empty-vault message", errOut)
	}
}

func TestListNoMatches(t *testing.T) {
	path := writeTestVaultWithEntries(t)
	out, errOut := listOutput(t, path, "does-not-exist")
	if out != "" {
		t.Fatalf("stdout = %q, want empty", out)
	}
	if !strings.Contains(errOut, "No matching entries.") {
		t.Fatalf("stderr = %q, want a no-match message", errOut)
	}
}
