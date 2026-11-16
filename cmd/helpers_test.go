package cmd

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/nxkh4ng/gatekeep/core"
)

const testMasterPassword = "correct horse battery staple"

var testEntries = []core.Entry{
	{Title: "YouTube", Username: "alice@gmail.com", Password: "alice-secret", URL: "https://youtube.com"},
	{Title: "YouTube", Username: "bob@gmail.com", Password: "bob-secret"},
	{Title: "Gmail", Username: "alice@gmail.com", Password: "mail-secret", URL: "https://mail.google.com"},
	{Title: "Bank", Username: "Bob", Password: "bank-secret", URL: "https://bank.example"},
}

var (
	seedOnce sync.Once
	seedData []byte
	seedErr  error
)

var (
	entriesSeedOnce sync.Once
	entriesSeedData []byte
	entriesSeedErr  error
)

// writeTestVault writes a fresh copy of a vault sealed with testMasterPassword.
// The sealed seed is created once per test binary to keep Argon2id costs down.
func writeTestVault(t *testing.T) string {
	t.Helper()
	seedOnce.Do(func() {
		v, err := core.New([]byte(testMasterPassword))
		if err != nil {
			seedErr = err
			return
		}
		seedData, seedErr = v.Seal()
		v.Lock()
	})
	if seedErr != nil {
		t.Fatalf("seeding test vault: %v", seedErr)
	}

	path := filepath.Join(t.TempDir(), "vault.gk")
	if err := os.WriteFile(path, seedData, 0o600); err != nil {
		t.Fatalf("writing test vault: %v", err)
	}
	return path
}

// writeTestVaultWithEntries writes a vault containing testEntries, reusing one
// sealed seed across tests.
func writeTestVaultWithEntries(t *testing.T) string {
	t.Helper()
	entriesSeedOnce.Do(func() {
		v, err := core.New([]byte(testMasterPassword))
		if err != nil {
			entriesSeedErr = err
			return
		}
		defer v.Lock()
		for _, e := range testEntries {
			if _, err := v.Add(e); err != nil {
				entriesSeedErr = err
				return
			}
		}
		entriesSeedData, entriesSeedErr = v.Seal()
	})
	if entriesSeedErr != nil {
		t.Fatalf("seeding test vault with entries: %v", entriesSeedErr)
	}

	path := filepath.Join(t.TempDir(), "vault.gk")
	if err := os.WriteFile(path, entriesSeedData, 0o600); err != nil {
		t.Fatalf("writing test vault: %v", err)
	}
	return path
}

func readTestEntries(t *testing.T, path string) []core.Entry {
	t.Helper()
	v, err := core.LoadFile(path, []byte(testMasterPassword))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	defer v.Lock()
	entries, err := v.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	return entries
}
