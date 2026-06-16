// Package format defines the on-disk Athena container layout shared by the Go
// encoder and the C decoder extension. See docs/FORMAT.md for the spec.
package format

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
)

// Magic marks the start of the binary container within an encoded .php file.
var Magic = []byte{0x41, 0x54, 0x48, 0x4E, 0x00, 0x01} // "ATHN",0x00,0x01

const (
	Version    = 1
	MagicLen   = 6
	HeaderLen  = 18 // magic(6)+ver(1)+flags(1)+reserved(2)+keyid(4)+origlen(4)
	NonceLen   = 24 // XChaCha20-Poly1305 IETF
	TagLen     = 16 // Poly1305
	KeyLen     = 32 // XChaCha20-Poly1305 key
	PrefixLen  = HeaderLen + NonceLen
)

// flag bits
const (
	FlagOpcodes    = 1 << 0 // payload is serialized opcodes (reserved, unused in v1)
	FlagCompressed = 1 << 1 // payload is compressed (reserved, unused in v1)
)

var (
	ErrShort      = errors.New("athena: data too short for container")
	ErrMagic      = errors.New("athena: bad magic")
	ErrVersion    = errors.New("athena: unsupported container version")
	ErrOrigLenMis = errors.New("athena: plaintext length mismatch")
)

// KeyID returns the CRC32 used to detect key mismatches.
func KeyID(key []byte) uint32 { return crc32.ChecksumIEEE(key) }

// Header describes a parsed container header (the AEAD associated data).
type Header struct {
	Version byte
	Flags   byte
	KeyID   uint32
	OrigLen uint32
}

// BuildHeader serializes a container header. The returned slice is exactly
// HeaderLen bytes and doubles as the AEAD associated data.
func BuildHeader(h Header) []byte {
	b := make([]byte, HeaderLen)
	copy(b, Magic)
	b[6] = h.Version
	b[7] = h.Flags
	// b[8:10] reserved = 0
	binary.LittleEndian.PutUint32(b[10:14], h.KeyID)
	binary.LittleEndian.PutUint32(b[14:18], h.OrigLen)
	return b
}

// ParseHeader validates and decodes a container header from the front of b.
func ParseHeader(b []byte) (Header, error) {
	if len(b) < HeaderLen {
		return Header{}, ErrShort
	}
	for i := 0; i < MagicLen; i++ {
		if b[i] != Magic[i] {
			return Header{}, ErrMagic
		}
	}
	h := Header{
		Version: b[6],
		Flags:   b[7],
		KeyID:   binary.LittleEndian.Uint32(b[10:14]),
		OrigLen: binary.LittleEndian.Uint32(b[14:18]),
	}
	if h.Version != Version {
		return h, ErrVersion
	}
	return h, nil
}
