# Athena container format (v1)

An Athena-encoded `.php` file has two parts:

```
[ ASCII PHP stub ]\n[ binary container ]
```

The **stub** is plain PHP that runs only when the `athena` extension is absent;
it prints a friendly error and exits. The extension's `zend_compile_file` hook
never executes the stub — it locates the binary container and decrypts it.

## Locating the container
The extension scans the file bytes for the 6-byte **MAGIC** and treats everything
from MAGIC onward as the binary container. The stub is authored to never contain
MAGIC.

```
MAGIC = 41 54 48 4E 00 01      ("ATHN", 0x00, 0x01)
```

## Binary container layout
All multi-byte integers are **little-endian**.

| Offset | Size | Field      | Description                                             |
|-------:|-----:|------------|---------------------------------------------------------|
| 0      | 6    | magic      | `41 54 48 4E 00 01`                                      |
| 6      | 1    | version    | format version, currently `1`                           |
| 7      | 1    | flags      | bit0 payload kind (0=source,1=opcodes); bit1 compressed |
| 8      | 2    | reserved   | zero                                                     |
| 10     | 4    | keyid      | CRC32 of the project key (detects key mismatch)         |
| 14     | 4    | origlen    | length in bytes of the original cleartext payload       |
| 18     | 24   | nonce      | XChaCha20-Poly1305 (IETF) nonce, random per file        |
| 42     | N    | ciphertext | XChaCha20-Poly1305 ciphertext incl. 16-byte Poly1305 tag|

- **Header** = bytes `[0, 18)` (magic … origlen). It is passed as the AEAD
  **associated data**, so version/flags/keyid/origlen are authenticated and any
  tampering fails decryption.
- Cleartext length = `len(ciphertext) - 16`. It must equal `origlen`.

## Crypto
- AEAD: **XChaCha20-Poly1305 (IETF)**, 256-bit key, 192-bit nonce.
- Key: 32 random bytes from `athena keygen`. Same key must be compiled into the
  decoder extension and used by the encoder.
- `keyid = crc32(key)` — lets the decoder reject files encoded with a different key.

## Versioning
`version` gates the whole layout. Unknown versions are rejected by the decoder and
the file is passed through to the normal PHP compiler (which then hits the stub).
