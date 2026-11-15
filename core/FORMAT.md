# GateKeep vault format

Version: 1 (stable once shipped; any layout change bumps the version byte).

The vault is a single file. Everything except the fixed-size header is
encrypted and authenticated. The header itself is plaintext but is bound as
AAD, so any modification makes decryption fail.

All integers are big-endian.

## Layout

| Offset | Size | Field                                              |
| ------ | ---- | -------------------------------------------------- |
| 0      | 4    | magic, ASCII `GKPV`                                |
| 4      | 1    | format version, `1`                                |
| 5      | 1    | KDF id, `1` = Argon2id                             |
| 6      | 4    | Argon2id time cost (iterations)                    |
| 10     | 4    | Argon2id memory cost, KiB                          |
| 14     | 1    | Argon2id parallelism (lanes)                       |
| 15     | 16   | salt, random per vault                             |
| 31     | 12   | wrap nonce, random per seal                        |
| 43     | 48   | wrapped DEK: 32-byte AES-256 key + 16-byte GCM tag |
| 91     | 12   | body nonce, random per seal                        |
| 103    | n    | body ciphertext: JSON + 16-byte GCM tag            |

The header is bytes `[0, 91)`. The body nonce sits between the header and the
ciphertext, at `[91, 103)`.

## Key hierarchy

1. `KEK = Argon2id(master password, salt, time, memory, threads, 32)`.
2. `DEK` is 32 random bytes generated once when the vault is created.
3. `wrapped DEK = AES-256-GCM(KEK, wrapNonce, DEK, AAD = bytes [0,31))`.
4. `body = AES-256-GCM(DEK, bodyNonce, JSON, AAD = bytes [0,103))`.

The split lets a master password change rewrap only the 32-byte DEK. It also
leaves room for alternative unlock methods (OS keychain, WebAuthn) that wrap
the same DEK differently.

A wrong password fails DEK unwrap; there is no separate verifier and no way to
distinguish a wrong password from a damaged file.

## KDF parameters

Defaults: time `3`, memory `65536` KiB (64 MiB), threads `4`.

Parsers must reject a header, before any allocation or key derivation, when:

- magic, format version, or KDF id is unknown;
- time is `0` or above `10`;
- threads is `0` or above `16`;
- memory is below `8 * threads` KiB (Argon2 minimum) or above `512 MiB`;
- the body exceeds 64 MiB.

Parameters are authenticated by the AAD, so an attacker cannot weaken an
existing vault without the master password. The bounds exist to stop memory
exhaustion from a hostile file.

## Body

The plaintext body is UTF-8 JSON:

```json
{
  "version": 1,
  "entries": [
    {
      "id": "0123456789abcdef0123456789abcdef",
      "title": "example",
      "username": "alice",
      "password": "s3cret",
      "url": "https://example.com",
      "notes": "",
      "created": "2026-11-16T05:00:00Z",
      "updated": "2026-11-16T05:00:00Z"
    }
  ]
}
```

- `id` is 16 random bytes, lowercase hex. It is stable across updates.
- `url`, `notes`, and other optional fields are omitted when empty.
- Timestamps are RFC 3339 in UTC.
- Body versioning is independent of the file format version.

## Test vector

103-byte header with salt `0x00..0x0f`, wrap nonce `0x10..0x1b`, a dummy
wrapped DEK `0x20..0x4f`, and body nonce `0x50..0x5b`:

```
474b5056010100000003000100000400
0102030405060708090a0b0c0d0e0f10
1112131415161718191a1b2021222324
25262728292a2b2c2d2e2f3031323334
35363738393a3b3c3d3e3f4041424344
45464748494a4b4c4d4e4f5051525354
55565758595a5b
```

Parsing then re-marshalling this header must reproduce the same 103 bytes.
`core/format_test.go` pins this vector.

## File handling (implemented with persistence in M4)

- Vault file mode `0600`, parent directory `0700`.
- Reject files larger than `103 + 64 MiB` before reading them into memory.
- Writes go to a temp file in the same directory, then `fsync` and `rename`.
- Keep the previous sealed file as `vault.bak`.

## Out of scope for v1

- TOTP secrets and attachments.
- Sync, network, and rollback protection.
- Metadata hiding beyond encryption of the whole body.
