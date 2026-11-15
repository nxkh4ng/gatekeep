package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// Argon2id cost for new vaults. See FORMAT.md for the accepted bounds.
const (
	defaultKDFTime    = 3
	defaultKDFMemory  = 64 << 10 // KiB
	defaultKDFThreads = 4
)

// defaultKDF is recorded in every vault created without an explicit choice.
var defaultKDF = kdfParams{
	time:    defaultKDFTime,
	memory:  defaultKDFMemory,
	threads: defaultKDFThreads,
}

// kdfParams are the Argon2id cost parameters stored in a vault header.
type kdfParams struct {
	time    uint32
	memory  uint32 // KiB
	threads uint8
}

var (
	// ErrInvalidPassword reports a vault that cannot be unlocked: either the
	// master password is wrong or the salt, KDF parameters, or wrapped key
	// were modified. It deliberately does not distinguish the two.
	ErrInvalidPassword = errors.New("gatekeep: invalid master password or corrupted vault")

	// ErrTooLarge reports a vault body that exceeds the format's size limit.
	ErrTooLarge = errors.New("gatekeep: vault body too large")
)

// kdf returns h's cost parameters.
func (h header) kdf() kdfParams {
	return kdfParams{time: h.kdfTime, memory: h.kdfMemory, threads: h.kdfThreads}
}

// sealVault encrypts plaintext into a complete vault file. It generates a
// fresh salt, data key, and nonces, and records kdf in the header.
func sealVault(password, plaintext []byte, kdf kdfParams) ([]byte, error) {
	if err := validateKDF(kdf.time, kdf.memory, kdf.threads); err != nil {
		return nil, fmt.Errorf("gatekeep: sealing: %w", err)
	}
	if len(plaintext)+tagSize > maxBodySize {
		return nil, fmt.Errorf("%w: %d bytes of plaintext", ErrTooLarge, len(plaintext))
	}

	h := header{
		kdfTime:    kdf.time,
		kdfMemory:  kdf.memory,
		kdfThreads: kdf.threads,
	}
	if err := randomFill(h.salt[:]); err != nil {
		return nil, err
	}
	if err := randomFill(h.wrapNonce[:]); err != nil {
		return nil, err
	}
	if err := randomFill(h.bodyNonce[:]); err != nil {
		return nil, err
	}

	var dek [keySize]byte
	if err := randomFill(dek[:]); err != nil {
		return nil, err
	}
	defer clear(dek[:])

	kek := deriveKey(password, h.salt, kdf)
	defer clear(kek)

	// The wrapped key is authenticated against the header up to the salt, so
	// KDF parameters cannot be swapped without detection.
	wrapped, err := gcmSeal(kek, h.wrapNonce[:], dek[:], h.marshal()[:offWrapNonce])
	if err != nil {
		return nil, err
	}
	copy(h.wrappedDEK[:], wrapped)

	// The body is authenticated against the whole header including the body
	// nonce.
	hdr := h.marshal()
	body, err := gcmSeal(dek[:], h.bodyNonce[:], plaintext, hdr)
	if err != nil {
		return nil, err
	}
	return append(hdr, body...), nil
}

// openVault decrypts a complete vault file. It returns ErrInvalidPassword when
// the wrapped data key cannot be opened, and ErrCorrupt when the header is
// malformed or the authenticated body fails to decrypt.
func openVault(data, password []byte) ([]byte, error) {
	h, ct, err := parseHeader(data)
	if err != nil {
		return nil, err
	}

	kek := deriveKey(password, h.salt, h.kdf())
	defer clear(kek)

	dek, err := gcmOpen(kek, h.wrapNonce[:], h.wrappedDEK[:], data[:offWrapNonce])
	if err != nil {
		return nil, fmt.Errorf("%w: cannot unwrap data key", ErrInvalidPassword)
	}
	defer clear(dek)
	if len(dek) != keySize {
		return nil, fmt.Errorf("%w: wrapped key is %d bytes", ErrCorrupt, len(dek))
	}

	body, err := gcmOpen(dek, h.bodyNonce[:], ct, data[:bodyOffset])
	if err != nil {
		return nil, fmt.Errorf("%w: body authentication failed", ErrCorrupt)
	}
	return body, nil
}

// deriveKey stretches the master password into a 32-byte AES key. kdf must
// have been validated by validateKDF; argon2 rejects anything outside it.
func deriveKey(password []byte, salt [saltSize]byte, kdf kdfParams) []byte {
	return argon2.IDKey(password, salt[:], kdf.time, kdf.memory, kdf.threads, keySize)
}

// randomFill fills b from crypto/rand.
func randomFill(b []byte) error {
	if _, err := rand.Read(b); err != nil {
		return fmt.Errorf("generating random bytes: %w", err)
	}
	return nil
}

// gcmSeal encrypts plaintext with AES-256-GCM, appending the tag.
func gcmSeal(key, nonce, plaintext, aad []byte) ([]byte, error) {
	aead, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	return aead.Seal(nil, nonce, plaintext, aad), nil
}

// gcmOpen authenticates and decrypts ciphertext.
func gcmOpen(key, nonce, ciphertext, aad []byte) ([]byte, error) {
	aead, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	return aead.Open(nil, nonce, ciphertext, aad)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	return aead, nil
}
