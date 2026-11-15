package core

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadFile(t *testing.T) {
	password := []byte("correct horse battery staple")
	v := newTestVault(t, password)
	if _, err := v.Add(Entry{Title: "mail", Username: "alice", Password: "s3cret"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	path := filepath.Join(t.TempDir(), "vault.gk")
	if err := v.SaveFile(path); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading saved vault: %v", err)
	}
	if bytes.Contains(raw, []byte("s3cret")) {
		t.Fatal("vault file contains plaintext entry data")
	}

	got, err := LoadFile(path, password)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	entries, err := got.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	if len(entries) != 1 || entries[0].Password != "s3cret" {
		t.Fatalf("entries = %+v", entries)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("re-reading saved vault: %v", err)
	}
	if !bytes.Equal(raw, after) {
		t.Fatal("LoadFile modified the vault file")
	}
}

func TestSaveFilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "vault.gk")
	v := newTestVault(t, []byte("pw"))
	if err := v.SaveFile(path); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat vault: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Fatalf("vault mode = %#o, want 0600", perm)
	}
	di, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat vault directory: %v", err)
	}
	if perm := di.Mode().Perm(); perm != 0o700 {
		t.Fatalf("vault directory mode = %#o, want 0700", perm)
	}
}

func TestSaveFileKeepsBackup(t *testing.T) {
	password := []byte("correct horse battery staple")
	path := filepath.Join(t.TempDir(), "vault.gk")
	v := newTestVault(t, password)

	firstID, err := v.Add(Entry{Title: "first"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := v.SaveFile(path); err != nil {
		t.Fatalf("first SaveFile: %v", err)
	}
	if _, err := os.Stat(path + backupSuffix); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("backup exists after first save: %v", err)
	}

	if _, err := v.Add(Entry{Title: "second"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := v.SaveFile(path); err != nil {
		t.Fatalf("second SaveFile: %v", err)
	}

	backup, err := LoadFile(path+backupSuffix, password)
	if err != nil {
		t.Fatalf("LoadFile backup: %v", err)
	}
	oldEntries, err := backup.Entries()
	if err != nil {
		t.Fatalf("Entries backup: %v", err)
	}
	if len(oldEntries) != 1 || oldEntries[0].ID != firstID {
		t.Fatalf("backup entries = %+v, want only the first entry", oldEntries)
	}

	current, err := LoadFile(path, password)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	entries, err := current.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("current vault has %d entries, want 2", len(entries))
	}
}

func TestSaveFileLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	v := newTestVault(t, []byte("pw"))
	if err := v.SaveFile(filepath.Join(dir, "vault.gk")); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "vault.gk" {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("directory has %v, want only vault.gk", names)
	}
}

func TestLoadFileMissing(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "missing.gk"), []byte("pw"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("LoadFile() error = %v, want fs.ErrNotExist", err)
	}
}

func TestLoadFileWrongPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.gk")
	v := newTestVault(t, []byte("correct horse battery staple"))
	if err := v.SaveFile(path); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	if _, err := LoadFile(path, []byte("wrong password")); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("LoadFile() error = %v, want ErrInvalidPassword", err)
	}
}

func TestLoadFileTooLarge(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.gk")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := f.Truncate(maxFileSize + 1); err != nil {
		t.Fatalf("Truncate: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := LoadFile(path, []byte("pw")); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("LoadFile() error = %v, want ErrTooLarge", err)
	}
}

func TestLoadFileGarbage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.gk")
	if err := os.WriteFile(path, []byte("not a vault"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := LoadFile(path, []byte("pw")); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("LoadFile() error = %v, want ErrCorrupt", err)
	}
}

func TestSaveFileLocked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.gk")
	v := newTestVault(t, []byte("pw"))
	v.Lock()
	if err := v.SaveFile(path); !errors.Is(err, ErrLocked) {
		t.Fatalf("SaveFile() error = %v, want ErrLocked", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("locked vault wrote a file: %v", err)
	}
}
