package cmd

import (
	"bytes"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultVaultPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME-based path resolution differs on Windows")
	}

	t.Run("XDG_DATA_HOME", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "/custom/data")
		got, err := defaultVaultPath()
		if err != nil {
			t.Fatalf("defaultVaultPath: %v", err)
		}
		if want := filepath.Join("/custom/data", appName, "vault.gk"); got != want {
			t.Fatalf("defaultVaultPath = %q, want %q", got, want)
		}
	})

	t.Run("relative XDG ignored", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "relative/path")
		t.Setenv("HOME", "/home/test")
		got, err := defaultVaultPath()
		if err != nil {
			t.Fatalf("defaultVaultPath: %v", err)
		}
		if want := filepath.Join("/home/test", ".local", "share", appName, "vault.gk"); got != want {
			t.Fatalf("defaultVaultPath = %q, want %q", got, want)
		}
	})

	t.Run("home fallback", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("HOME", "/home/test")
		got, err := defaultVaultPath()
		if err != nil {
			t.Fatalf("defaultVaultPath: %v", err)
		}
		if want := filepath.Join("/home/test", ".local", "share", appName, "vault.gk"); got != want {
			t.Fatalf("defaultVaultPath = %q, want %q", got, want)
		}
	})
}

func TestRootCommands(t *testing.T) {
	root := newRoot(&app{vault: "/tmp/vault.gk"})

	names := make(map[string]bool)
	for _, c := range root.Commands() {
		names[c.Name()] = true
	}
	for _, want := range []string{"init", "add", "list", "show", "edit", "rm", "passwd"} {
		if !names[want] {
			t.Errorf("root is missing %q command", want)
		}
	}
	if root.Version != version {
		t.Errorf("root version = %q, want %q", root.Version, version)
	}
	if root.PersistentFlags().Lookup("vault") == nil {
		t.Error("root is missing the --vault flag")
	}
}

func TestRootUsageLine(t *testing.T) {
	root := newRoot(&app{vault: "/tmp/vault.gk", prompter: &fakePrompter{}})
	root.SetArgs([]string{})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(io.Discard)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Usage:\n  gatekeep [command] [flags]\n"; !strings.Contains(out.String(), want) {
		t.Fatalf("help output = %q, want usage line %q", out.String(), want)
	}
	if strings.Contains(out.String(), "\n  gatekeep [command]\n") {
		t.Fatalf("help output still has the default usage lines: %q", out.String())
	}
}

func TestSubcommandUsageUnchanged(t *testing.T) {
	root := newRoot(&app{vault: "/tmp/vault.gk", prompter: &fakePrompter{}})
	root.SetArgs([]string{"add", "--help"})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(io.Discard)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Usage:\n  gatekeep add [title] [flags]\n"; !strings.Contains(out.String(), want) {
		t.Fatalf("help output = %q, want usage line %q", out.String(), want)
	}
}

func TestVaultFlagOverridesDefault(t *testing.T) {
	a := &app{vault: "/tmp/default.gk"}
	root := newRoot(a)
	root.SetArgs([]string{"--vault", "/tmp/other.gk"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if a.vault != "/tmp/other.gk" {
		t.Fatalf("vault = %q, want /tmp/other.gk", a.vault)
	}
}
