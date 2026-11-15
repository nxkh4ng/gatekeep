package core

import (
	"bytes"
	"errors"
	"testing"
)

// testKDF keeps Argon2id cheap so tests can exercise many derivations. It is
// still within the bounds a parser accepts.
var testKDF = kdfParams{time: 1, memory: 8, threads: 1}

func sealedTestVault(t *testing.T) (sealed, password []byte) {
	t.Helper()
	password = []byte("correct horse battery staple")
	sealed, err := sealVault(password, []byte(`{"version":1,"entries":[]}`), testKDF)
	if err != nil {
		t.Fatalf("sealVault: %v", err)
	}
	return sealed, password
}

func TestSealOpenRoundTrip(t *testing.T) {
	password := []byte("correct horse battery staple")
	plaintext := []byte(`{"version":1,"entries":[]}`)

	sealed, err := sealVault(password, plaintext, testKDF)
	if err != nil {
		t.Fatalf("sealVault: %v", err)
	}
	if want := bodyOffset + len(plaintext) + tagSize; len(sealed) != want {
		t.Fatalf("sealed vault is %d bytes, want %d", len(sealed), want)
	}

	h, ct, err := parseHeader(sealed)
	if err != nil {
		t.Fatalf("parseHeader: %v", err)
	}
	if h.kdf() != testKDF {
		t.Fatalf("header KDF = %+v, want %+v", h.kdf(), testKDF)
	}
	if len(ct) != len(plaintext)+tagSize {
		t.Fatalf("ciphertext is %d bytes, want %d", len(ct), len(plaintext)+tagSize)
	}

	got, err := openVault(sealed, password)
	if err != nil {
		t.Fatalf("openVault: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("openVault = %q, want %q", got, plaintext)
	}
}

func TestOpenVaultWrongPassword(t *testing.T) {
	sealed, _ := sealedTestVault(t)
	if _, err := openVault(sealed, []byte("wrong password")); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("openVault() error = %v, want ErrInvalidPassword", err)
	}
}

func TestOpenVaultTampered(t *testing.T) {
	sealed, password := sealedTestVault(t)

	for i := range sealed {
		data := bytes.Clone(sealed)
		data[i] ^= 0x01

		_, err := openVault(data, password)
		if i < offBodyNonce {
			// Damage to the salt, memory cost, or wrapped key is
			// indistinguishable from a wrong password.
			if !errors.Is(err, ErrCorrupt) && !errors.Is(err, ErrInvalidPassword) {
				t.Fatalf("header byte %d: error = %v, want ErrCorrupt or ErrInvalidPassword", i, err)
			}
			continue
		}
		if !errors.Is(err, ErrCorrupt) {
			t.Fatalf("body byte %d: error = %v, want ErrCorrupt", i, err)
		}
	}
}

func TestOpenVaultTruncated(t *testing.T) {
	sealed, password := sealedTestVault(t)

	for _, n := range []int{0, 1, 50, offBodyNonce, bodyOffset, bodyOffset + tagSize - 1, len(sealed) - 1} {
		if _, err := openVault(sealed[:n], password); !errors.Is(err, ErrCorrupt) {
			t.Fatalf("openVault(%d bytes) error = %v, want ErrCorrupt", n, err)
		}
	}
}

func TestOpenVaultEmptyBody(t *testing.T) {
	password := []byte("correct horse battery staple")
	sealed, err := sealVault(password, nil, testKDF)
	if err != nil {
		t.Fatalf("sealVault: %v", err)
	}
	got, err := openVault(sealed, password)
	if err != nil {
		t.Fatalf("openVault: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("openVault returned %d bytes, want 0", len(got))
	}
}

func TestSealVaultDistinctOutput(t *testing.T) {
	password := []byte("correct horse battery staple")
	plaintext := []byte(`{"version":1,"entries":[]}`)

	first, err := sealVault(password, plaintext, testKDF)
	if err != nil {
		t.Fatalf("first sealVault: %v", err)
	}
	second, err := sealVault(password, plaintext, testKDF)
	if err != nil {
		t.Fatalf("second sealVault: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("two seals of the same vault are identical; nonces or salt are reused")
	}
}

func TestSealVaultRejectsOversizedBody(t *testing.T) {
	_, err := sealVault([]byte("pw"), make([]byte, maxBodySize-tagSize+1), testKDF)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("sealVault() error = %v, want ErrTooLarge", err)
	}
}

func TestKDFDefaults(t *testing.T) {
	if err := validateKDF(defaultKDF.time, defaultKDF.memory, defaultKDF.threads); err != nil {
		t.Fatalf("default KDF parameters are invalid: %v", err)
	}
}

func TestDeriveKey(t *testing.T) {
	password := []byte("correct horse battery staple")
	var salt [saltSize]byte
	for i := range salt {
		salt[i] = byte(i)
	}

	key := deriveKey(password, salt, testKDF)
	if len(key) != keySize {
		t.Fatalf("key is %d bytes, want %d", len(key), keySize)
	}
	if !bytes.Equal(key, deriveKey(password, salt, testKDF)) {
		t.Fatal("deriveKey is not deterministic")
	}

	otherSalt := salt
	otherSalt[0] ^= 0x01
	if bytes.Equal(key, deriveKey(password, otherSalt, testKDF)) {
		t.Fatal("a different salt produced the same key")
	}
	if bytes.Equal(key, deriveKey(password, salt, kdfParams{time: 2, memory: 8, threads: 1})) {
		t.Fatal("different KDF cost produced the same key")
	}
}
