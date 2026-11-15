package core

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"testing"
)

// headerHex is the test vector documented in FORMAT.md: the 103-byte header
// prefix of a vault with time=3, memory=65536 KiB, threads=4, and byte
// patterns 0x00, 0x10, 0x20, 0x50 for salt, wrap nonce, wrapped DEK, and body
// nonce.
const headerHex = "" +
	"474b5056010100000003000100000400" +
	"0102030405060708090a0b0c0d0e0f10" +
	"1112131415161718191a1b2021222324" +
	"25262728292a2b2c2d2e2f3031323334" +
	"35363738393a3b3c3d3e3f4041424344" +
	"45464748494a4b4c4d4e4f5051525354" +
	"55565758595a5b"

func mustHeader(tb testing.TB) []byte {
	tb.Helper()
	b, err := hex.DecodeString(headerHex)
	if err != nil {
		tb.Fatalf("decoding test vector: %v", err)
	}
	if len(b) != bodyOffset {
		tb.Fatalf("test vector is %d bytes, want %d", len(b), bodyOffset)
	}
	return b
}

func TestHeaderVector(t *testing.T) {
	hdr := mustHeader(t)
	data := append(bytes.Clone(hdr), make([]byte, tagSize)...)

	h, body, err := parseHeader(data)
	if err != nil {
		t.Fatalf("parseHeader: %v", err)
	}
	if h.kdfTime != 3 || h.kdfMemory != 65536 || h.kdfThreads != 4 {
		t.Fatalf("KDF = time %d, memory %d, threads %d", h.kdfTime, h.kdfMemory, h.kdfThreads)
	}
	patterns := []struct {
		name string
		got  []byte
		base byte
	}{
		{"salt", h.salt[:], 0x00},
		{"wrap nonce", h.wrapNonce[:], 0x10},
		{"wrapped DEK", h.wrappedDEK[:], 0x20},
		{"body nonce", h.bodyNonce[:], 0x50},
	}
	for _, p := range patterns {
		for i, b := range p.got {
			if b != p.base+byte(i) {
				t.Fatalf("%s[%d] = %#x, want %#x", p.name, i, b, p.base+byte(i))
			}
		}
	}
	if !bytes.Equal(h.marshal(), hdr) {
		t.Fatal("marshal does not reproduce the test vector")
	}
	if !bytes.Equal(body, data[bodyOffset:]) {
		t.Fatal("body does not alias the ciphertext")
	}
}

func TestParseHeaderRejects(t *testing.T) {
	base := append(mustHeader(t), make([]byte, tagSize)...)

	tests := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{"empty", func([]byte) []byte { return nil }},
		{"header only", func(b []byte) []byte { return b[:bodyOffset] }},
		{"short tag", func(b []byte) []byte { return b[:bodyOffset+tagSize-1] }},
		{"bad magic", func(b []byte) []byte { b[0] = 'X'; return b }},
		{"bad version", func(b []byte) []byte { b[offVersion] = 2; return b }},
		{"bad kdf", func(b []byte) []byte { b[offKDF] = 2; return b }},
		{"zero time", func(b []byte) []byte {
			binary.BigEndian.PutUint32(b[offTime:offMemory], 0)
			return b
		}},
		{"time too high", func(b []byte) []byte {
			binary.BigEndian.PutUint32(b[offTime:offMemory], maxKDFTime+1)
			return b
		}},
		{"zero threads", func(b []byte) []byte { b[offThreads] = 0; return b }},
		{"threads too high", func(b []byte) []byte { b[offThreads] = maxKDFThreads + 1; return b }},
		{"memory below argon2 minimum", func(b []byte) []byte {
			b[offThreads] = 4
			binary.BigEndian.PutUint32(b[offMemory:offThreads], 8*4-1)
			return b
		}},
		{"memory too high", func(b []byte) []byte {
			binary.BigEndian.PutUint32(b[offMemory:offThreads], maxKDFMemory+1)
			return b
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.mutate(bytes.Clone(base))
			if _, _, err := parseHeader(data); !errors.Is(err, ErrCorrupt) {
				t.Fatalf("parseHeader() error = %v, want ErrCorrupt", err)
			}
		})
	}
}

func TestParseHeaderRejectsOversizedBody(t *testing.T) {
	data := make([]byte, bodyOffset+maxBodySize+1)
	copy(data, mustHeader(t))
	if _, _, err := parseHeader(data); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("parseHeader() error = %v, want ErrCorrupt", err)
	}
}

func FuzzParseHeader(f *testing.F) {
	hdr := mustHeader(f)
	f.Add(hdr)
	f.Add(append(bytes.Clone(hdr), make([]byte, tagSize)...))
	f.Add(hdr[:50])

	f.Fuzz(func(t *testing.T, data []byte) {
		h, body, err := parseHeader(data)
		if err != nil {
			if !errors.Is(err, ErrCorrupt) {
				t.Fatalf("unexpected error: %v", err)
			}
			return
		}
		if len(body) < tagSize {
			t.Fatalf("accepted %d-byte body", len(body))
		}
		if got := h.marshal(); !bytes.Equal(got, data[:bodyOffset]) {
			t.Fatal("header does not round-trip")
		}
	})
}
