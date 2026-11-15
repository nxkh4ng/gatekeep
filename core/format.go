// Package core implements the GateKeep password vault: an encrypted,
// single-file password store that every client (CLI, web extension) shares.
package core

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// The vault file layout is documented in FORMAT.md. All integers are
// big-endian.
const (
	saltSize      = 16
	nonceSize     = 12 // AES-GCM standard nonce
	keySize       = 32 // AES-256
	tagSize       = 16
	wrappedDEKLen = keySize + tagSize

	formatVersion = 1
	kdfArgon2id   = 1

	offMagic      = 0
	offVersion    = offMagic + 4
	offKDF        = offVersion + 1
	offTime       = offKDF + 1
	offMemory     = offTime + 4
	offThreads    = offMemory + 4
	offSalt       = offThreads + 1
	offWrapNonce  = offSalt + saltSize
	offWrappedDEK = offWrapNonce + nonceSize
	offBodyNonce  = offWrappedDEK + wrappedDEKLen // header ends here
	bodyOffset    = offBodyNonce + nonceSize      // first ciphertext byte
)

// Limits applied before any allocation or key derivation.
const (
	maxKDFTime    = 10
	maxKDFMemory  = 512 << 10 // KiB
	maxKDFThreads = 16
	maxBodySize   = 64 << 20
)

var magic = [4]byte{'G', 'K', 'P', 'V'}

// ErrCorrupt reports a vault file that is truncated, malformed, or uses an
// unknown format version or KDF. It is returned before any key derivation, so
// it never distinguishes a wrong password from a damaged file.
var ErrCorrupt = errors.New("gatekeep: corrupt or unsupported vault")

// header is the plaintext, authenticated prefix of a vault file.
type header struct {
	kdfTime    uint32
	kdfMemory  uint32 // KiB
	kdfThreads uint8
	salt       [saltSize]byte
	wrapNonce  [nonceSize]byte
	wrappedDEK [wrappedDEKLen]byte
	bodyNonce  [nonceSize]byte
}

// parseHeader validates data and returns its header along with the encrypted
// body. The returned slice aliases data.
func parseHeader(data []byte) (header, []byte, error) {
	var h header
	if len(data) < bodyOffset+tagSize {
		return h, nil, fmt.Errorf("%w: file too short", ErrCorrupt)
	}
	if [4]byte(data[:4]) != magic {
		return h, nil, fmt.Errorf("%w: bad magic", ErrCorrupt)
	}
	if data[offVersion] != formatVersion {
		return h, nil, fmt.Errorf("%w: format version %d", ErrCorrupt, data[offVersion])
	}
	if data[offKDF] != kdfArgon2id {
		return h, nil, fmt.Errorf("%w: KDF id %d", ErrCorrupt, data[offKDF])
	}

	h.kdfTime = binary.BigEndian.Uint32(data[offTime:offMemory])
	h.kdfMemory = binary.BigEndian.Uint32(data[offMemory:offThreads])
	h.kdfThreads = data[offThreads]
	copy(h.salt[:], data[offSalt:offWrapNonce])
	copy(h.wrapNonce[:], data[offWrapNonce:offWrappedDEK])
	copy(h.wrappedDEK[:], data[offWrappedDEK:offBodyNonce])
	copy(h.bodyNonce[:], data[offBodyNonce:bodyOffset])

	if err := validateKDF(h.kdfTime, h.kdfMemory, h.kdfThreads); err != nil {
		return header{}, nil, fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	if len(data)-bodyOffset > maxBodySize {
		return header{}, nil, fmt.Errorf("%w: body of %d bytes", ErrCorrupt, len(data)-bodyOffset)
	}
	return h, data[bodyOffset:], nil
}

// validateKDF reports whether the Argon2id cost parameters are both safe to
// derive with and within the bounds the format accepts.
func validateKDF(time, memory uint32, threads uint8) error {
	switch {
	case time == 0 || time > maxKDFTime:
		return fmt.Errorf("KDF time %d", time)
	case threads == 0 || threads > maxKDFThreads:
		return fmt.Errorf("KDF threads %d", threads)
	case memory < 8*uint32(threads) || memory > maxKDFMemory:
		return fmt.Errorf("KDF memory %d KiB", memory)
	}
	return nil
}

// marshal encodes h into a full header plus body nonce. The layout is fixed,
// so marshal(parseHeader(b)) reproduces b's first bodyOffset bytes exactly.
func (h header) marshal() []byte {
	b := make([]byte, bodyOffset)
	copy(b[offMagic:offVersion], magic[:])
	b[offVersion] = formatVersion
	b[offKDF] = kdfArgon2id
	binary.BigEndian.PutUint32(b[offTime:offMemory], h.kdfTime)
	binary.BigEndian.PutUint32(b[offMemory:offThreads], h.kdfMemory)
	b[offThreads] = h.kdfThreads
	copy(b[offSalt:offWrapNonce], h.salt[:])
	copy(b[offWrapNonce:offWrappedDEK], h.wrapNonce[:])
	copy(b[offWrappedDEK:offBodyNonce], h.wrappedDEK[:])
	copy(b[offBodyNonce:bodyOffset], h.bodyNonce[:])
	return b
}
