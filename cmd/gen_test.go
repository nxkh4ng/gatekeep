package cmd

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nxkh4ng/gatekeep/core"
)

// executeGen runs a gen command against a vault that does not exist. It fails
// the test if the command prompts for anything, because gen never reads the
// vault.
func executeGen(t *testing.T, args ...string) (string, error) {
	t.Helper()
	p := &fakePrompter{}
	root := newRoot(&app{vault: filepath.Join(t.TempDir(), "vault.gk"), prompter: p})
	root.SetArgs(args)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(io.Discard)
	err := root.Execute()
	if len(p.prompts) != 0 {
		t.Fatalf("gen prompted for input: %q", p.prompts)
	}
	return out.String(), err
}

func countDigits(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n++
		}
	}
	return n
}

func isLetter(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}

func isAlnum(r rune) bool {
	return isLetter(r) || r >= '0' && r <= '9'
}

func TestGenDefaults(t *testing.T) {
	out, err := executeGen(t, "gen")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	passwords := strings.Fields(out)
	if len(passwords) != 1 {
		t.Fatalf("gen printed %d passwords, want 1", len(passwords))
	}
	if got := len(passwords[0]); got != 20 {
		t.Fatalf("password has length %d, want 20", got)
	}
	if countDigits(passwords[0]) < 1 {
		t.Fatalf("password %q has no digit", passwords[0])
	}
	if !strings.ContainsAny(passwords[0], core.DefaultSymbols) {
		t.Fatalf("password %q has no symbol", passwords[0])
	}
}

func TestGenFlags(t *testing.T) {
	out, err := executeGen(t,
		"gen",
		"-l", "12",
		"--charset", "lower,upper,digits",
		"--min-digits", "2",
		"--count", "3",
		"--start-letter",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	passwords := strings.Fields(out)
	if len(passwords) != 3 {
		t.Fatalf("gen printed %d passwords, want 3", len(passwords))
	}
	for _, password := range passwords {
		if len(password) != 12 {
			t.Fatalf("password %q has length %d, want 12", password, len(password))
		}
		if countDigits(password) < 2 {
			t.Fatalf("password %q has fewer than 2 digits", password)
		}
		if strings.ContainsAny(password, core.DefaultSymbols) {
			t.Fatalf("password %q contains a symbol", password)
		}
		for _, r := range password {
			alnum := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
			if !alnum {
				t.Fatalf("password %q contains %q outside the charset", password, r)
			}
		}
		if r := rune(password[0]); !isLetter(r) {
			t.Fatalf("password %q does not start with a letter", password)
		}
	}
}

func TestGenCustomSymbols(t *testing.T) {
	out, err := executeGen(t, "gen", "--charset", "symbols", "--symbols", "!@#", "-l", "16")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	passwords := strings.Fields(out)
	if len(passwords) != 1 {
		t.Fatalf("gen printed %d passwords, want 1", len(passwords))
	}
	for _, r := range passwords[0] {
		if !strings.ContainsRune("!@#", r) {
			t.Fatalf("password %q contains %q outside the symbol set", passwords[0], r)
		}
	}
}

func TestGenErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"unknown class", []string{"gen", "--charset", "lower,bogus"}, "unknown character class"},
		{"empty charset", []string{"gen", "--charset", ""}, "unknown character class"},
		{"bad count", []string{"gen", "--count", "0"}, "count must be between"},
		{
			"symbols without class",
			[]string{"gen", "--charset", "digits", "--symbols", "!@#"},
			"symbol class is not enabled",
		},
		{"unsatisfiable", []string{"gen", "-l", "4", "--min-digits", "5"}, "invalid generation policy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeGen(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Execute() error = %v, want %q", err, tt.want)
			}
		})
	}
}
