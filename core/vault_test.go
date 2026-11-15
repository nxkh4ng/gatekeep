package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"slices"
	"testing"
)

func newTestVault(t *testing.T, password []byte) *Vault {
	t.Helper()
	v, err := newVault(password, testKDF)
	if err != nil {
		t.Fatalf("newVault: %v", err)
	}
	return v
}

func TestVaultCRUD(t *testing.T) {
	v := newTestVault(t, []byte("correct horse battery staple"))

	id, err := v.Add(Entry{Title: "mail", Username: "alice", Password: "s3cret"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(id) != 32 {
		t.Fatalf("ID %q is not 32 hex characters", id)
	}
	if _, err := hex.DecodeString(id); err != nil {
		t.Fatalf("ID %q is not hex: %v", id, err)
	}

	got, err := v.Entry(id)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if got.Title != "mail" || got.Username != "alice" || got.Password != "s3cret" {
		t.Fatalf("Entry = %+v", got)
	}
	if got.Created.IsZero() || !got.Created.Equal(got.Updated) {
		t.Fatalf("timestamps on add: created %v, updated %v", got.Created, got.Updated)
	}

	got.Title = "Mail"
	got.Password = "n3w"
	if err := v.Update(id, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	updated, err := v.Entry(id)
	if err != nil {
		t.Fatalf("Entry after update: %v", err)
	}
	if updated.Title != "Mail" || updated.Password != "n3w" {
		t.Fatalf("Entry after update = %+v", updated)
	}
	if !updated.Created.Equal(got.Created) {
		t.Fatal("Update changed the creation time")
	}
	if updated.Updated.Before(updated.Created) {
		t.Fatalf("updated %v is before created %v", updated.Updated, updated.Created)
	}

	entries, err := v.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != id {
		t.Fatalf("Entries = %+v", entries)
	}

	if err := v.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := v.Entry(id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Entry after delete: error = %v, want ErrNotFound", err)
	}
	if err := v.Delete(id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Delete: error = %v, want ErrNotFound", err)
	}
	if err := v.Update(id, Entry{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update of unknown ID: error = %v, want ErrNotFound", err)
	}
}

func TestVaultSealOpen(t *testing.T) {
	password := []byte("correct horse battery staple")
	v := newTestVault(t, password)

	want := []Entry{
		{Title: "mail", Username: "alice", Password: "s3cret", URL: "https://mail.example", Notes: "two-factor on"},
		{Title: "bank", Username: "bob", Password: "hunter2"},
	}
	for _, e := range want {
		if _, err := v.Add(e); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	data, err := v.Seal()
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	opened, err := Open(data, password)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, err := opened.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("opened %d entries, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID == "" || got[i].Title != want[i].Title || got[i].Username != want[i].Username ||
			got[i].Password != want[i].Password || got[i].URL != want[i].URL || got[i].Notes != want[i].Notes {
			t.Fatalf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
		if got[i].Created.IsZero() || got[i].Updated.Before(got[i].Created) {
			t.Fatalf("entry %d has bad timestamps: %+v", i, got[i])
		}
	}
}

func TestVaultEmptySealSchema(t *testing.T) {
	password := []byte("correct horse battery staple")
	v := newTestVault(t, password)

	data, err := v.Seal()
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	_, body, err := unlock(data, password)
	if err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if want := `{"version":1,"entries":[]}`; string(body) != want {
		t.Fatalf("body = %s, want %s", body, want)
	}
}

func TestVaultEntriesCopy(t *testing.T) {
	v := newTestVault(t, []byte("correct horse battery staple"))
	if _, err := v.Add(Entry{Title: "mail", Password: "s3cret"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entries, err := v.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	entries[0].Password = "mutated"

	again, err := v.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	if again[0].Password != "s3cret" {
		t.Fatal("Entries exposed the vault's internal slice")
	}
	if slices.Equal(entries, again) {
		t.Fatal("Entries did not return a copy")
	}
}

func TestVaultLock(t *testing.T) {
	v := newTestVault(t, []byte("correct horse battery staple"))
	if _, err := v.Add(Entry{Title: "mail"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	v.Lock()
	v.Lock()

	if v.keys != (vaultKeys{}) {
		t.Fatal("Lock left key material in place")
	}
	if len(v.data.Entries) != 0 {
		t.Fatal("Lock left entries in memory")
	}
	if _, err := v.Entries(); !errors.Is(err, ErrLocked) {
		t.Fatalf("Entries: error = %v, want ErrLocked", err)
	}
	if _, err := v.Entry("id"); !errors.Is(err, ErrLocked) {
		t.Fatalf("Entry: error = %v, want ErrLocked", err)
	}
	if _, err := v.Add(Entry{}); !errors.Is(err, ErrLocked) {
		t.Fatalf("Add: error = %v, want ErrLocked", err)
	}
	if err := v.Update("id", Entry{}); !errors.Is(err, ErrLocked) {
		t.Fatalf("Update: error = %v, want ErrLocked", err)
	}
	if err := v.Delete("id"); !errors.Is(err, ErrLocked) {
		t.Fatalf("Delete: error = %v, want ErrLocked", err)
	}
	if _, err := v.Seal(); !errors.Is(err, ErrLocked) {
		t.Fatalf("Seal: error = %v, want ErrLocked", err)
	}
	if err := v.ChangePassword([]byte("new")); !errors.Is(err, ErrLocked) {
		t.Fatalf("ChangePassword: error = %v, want ErrLocked", err)
	}
}

func TestVaultOpenWrongPassword(t *testing.T) {
	password := []byte("correct horse battery staple")
	v := newTestVault(t, password)
	data, err := v.Seal()
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := Open(data, []byte("wrong password")); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("Open() error = %v, want ErrInvalidPassword", err)
	}
}

func TestVaultOpenTamperedBody(t *testing.T) {
	password := []byte("correct horse battery staple")
	v := newTestVault(t, password)
	data, err := v.Seal()
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	data[len(data)-1] ^= 0x01
	if _, err := Open(data, password); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Open() error = %v, want ErrCorrupt", err)
	}
}

func TestVaultOpenBadBody(t *testing.T) {
	password := []byte("correct horse battery staple")
	k, err := newKeys(password, testKDF)
	if err != nil {
		t.Fatalf("newKeys: %v", err)
	}

	for _, body := range []string{
		"not json",
		`{"version":2,"entries":[]}`,
		`{"entries":[]}`,
	} {
		data, err := k.seal([]byte(body))
		if err != nil {
			t.Fatalf("seal(%q): %v", body, err)
		}
		if _, err := Open(data, password); !errors.Is(err, ErrCorrupt) {
			t.Fatalf("Open(%q) error = %v, want ErrCorrupt", body, err)
		}
	}
}

func TestVaultChangePassword(t *testing.T) {
	oldPassword := []byte("correct horse battery staple")
	newPassword := []byte("a much better master password")
	v := newTestVault(t, oldPassword)
	id, err := v.Add(Entry{Title: "mail", Password: "s3cret"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := v.ChangePassword(newPassword); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	data, err := v.Seal()
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	if _, err := Open(data, oldPassword); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("Open with old password: error = %v, want ErrInvalidPassword", err)
	}
	opened, err := Open(data, newPassword)
	if err != nil {
		t.Fatalf("Open with new password: %v", err)
	}
	got, err := opened.Entry(id)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if got.Password != "s3cret" {
		t.Fatalf("entry password = %q", got.Password)
	}
}

func TestVaultSealDistinct(t *testing.T) {
	password := []byte("correct horse battery staple")
	v := newTestVault(t, password)
	if _, err := v.Add(Entry{Title: "mail"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	first, err := v.Seal()
	if err != nil {
		t.Fatalf("first Seal: %v", err)
	}
	second, err := v.Seal()
	if err != nil {
		t.Fatalf("second Seal: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("two seals of the same vault are identical")
	}
	for i, data := range [][]byte{first, second} {
		opened, err := Open(data, password)
		if err != nil {
			t.Fatalf("Open seal %d: %v", i, err)
		}
		entries, err := opened.Entries()
		if err != nil || len(entries) != 1 || entries[0].Title != "mail" {
			t.Fatalf("seal %d entries = %+v, err = %v", i, entries, err)
		}
	}
}

func TestVaultAddUniqueIDs(t *testing.T) {
	v := newTestVault(t, []byte("correct horse battery staple"))
	seen := make(map[string]bool)
	for range 64 {
		id, err := v.Add(Entry{Title: "same"})
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
		if seen[id] {
			t.Fatalf("duplicate ID %q", id)
		}
		seen[id] = true
	}
}

func TestNewUsesDefaultKDF(t *testing.T) {
	v, err := New([]byte("correct horse battery staple"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	data, err := v.Seal()
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	h, _, err := parseHeader(data)
	if err != nil {
		t.Fatalf("parseHeader: %v", err)
	}
	if h.kdf() != defaultKDF {
		t.Fatalf("header KDF = %+v, want %+v", h.kdf(), defaultKDF)
	}
}
