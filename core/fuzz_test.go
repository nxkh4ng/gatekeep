package core

import (
	"bytes"
	"errors"
	"testing"
	"unicode/utf8"
)

const fuzzMaxInput = 1 << 20

// FuzzOpen checks that arbitrary vault bytes never panic Open and only
// produce the documented errors. Headers with a different KDF are skipped so
// fuzzing stays fast; FuzzParseHeader covers the KDF bounds.
func FuzzOpen(f *testing.F) {
	password := []byte("fuzz-password")
	sealed, err := sealVault(password, []byte(`{"version":1,"entries":[]}`), testKDF)
	if err != nil {
		f.Fatalf("sealVault: %v", err)
	}

	f.Add([]byte(nil))
	f.Add(make([]byte, bodyOffset))
	f.Add(make([]byte, bodyOffset+tagSize))
	f.Add(sealed)
	f.Add(sealed[:bodyOffset])
	f.Add(append(bytes.Clone(sealed), 0))
	f.Add(mustHeader(f))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > fuzzMaxInput {
			return
		}

		h, _, err := parseHeader(data)
		if err == nil && h.kdf() != testKDF {
			return
		}

		v, err := Open(data, password)
		if err != nil {
			if !errors.Is(err, ErrCorrupt) && !errors.Is(err, ErrInvalidPassword) {
				t.Fatalf("Open returned unexpected error: %v", err)
			}
			return
		}
		if _, err := v.Entries(); err != nil {
			t.Fatalf("Entries on opened vault: %v", err)
		}
		v.Lock()
	})
}

// FuzzSealOpenRoundTrip checks that entry fields survive Seal and Open.
func FuzzSealOpenRoundTrip(f *testing.F) {
	f.Add("mail", "alice", "s3cret", "https://mail.example", "notes")
	f.Add("", "", "", "", "")

	f.Fuzz(func(t *testing.T, title, username, password, url, notes string) {
		if !utf8.ValidString(title) || !utf8.ValidString(username) ||
			!utf8.ValidString(password) || !utf8.ValidString(url) ||
			!utf8.ValidString(notes) {
			return
		}

		v, err := newVault([]byte("fuzz-password"), testKDF)
		if err != nil {
			t.Fatalf("newVault: %v", err)
		}
		defer v.Lock()

		want := Entry{
			Title:    title,
			Username: username,
			Password: password,
			URL:      url,
			Notes:    notes,
		}
		if _, err := v.Add(want); err != nil {
			t.Fatalf("Add: %v", err)
		}

		data, err := v.Seal()
		if err != nil {
			if errors.Is(err, ErrTooLarge) {
				return
			}
			t.Fatalf("Seal: %v", err)
		}

		opened, err := Open(data, []byte("fuzz-password"))
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer opened.Lock()

		entries, err := opened.Entries()
		if err != nil {
			t.Fatalf("Entries: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("entries = %d, want 1", len(entries))
		}

		got := entries[0]
		if got.Title != title || got.Username != username ||
			got.Password != password || got.URL != url || got.Notes != notes {
			t.Fatalf("entry = %+v, want %+v", got, want)
		}
		if got.ID == "" || got.Created.IsZero() || got.Updated.Before(got.Created) {
			t.Fatalf("invalid generated metadata: %+v", got)
		}
	})
}
