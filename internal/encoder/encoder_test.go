package encoder

import (
	"bytes"
	"testing"

	"athena/internal/crypto"
	"athena/internal/format"
	"golang.org/x/crypto/chacha20poly1305"
)

// decode mirrors what the C extension must do, validating the container is
// self-consistent and decrypts back to the original source.
func decode(t *testing.T, file, key []byte) []byte {
	t.Helper()
	idx := bytes.Index(file, format.Magic)
	if idx < 0 {
		t.Fatal("magic not found")
	}
	c := file[idx:]
	h, err := format.ParseHeader(c)
	if err != nil {
		t.Fatalf("parse header: %v", err)
	}
	if h.KeyID != format.KeyID(key) {
		t.Fatalf("keyid mismatch")
	}
	aad := c[:format.HeaderLen]
	nonce := c[format.HeaderLen : format.HeaderLen+format.NonceLen]
	ct := c[format.HeaderLen+format.NonceLen:]
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		t.Fatal(err)
	}
	pt, err := aead.Open(nil, nonce, ct, aad)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if uint32(len(pt)) != h.OrigLen {
		t.Fatalf("origlen mismatch: %d vs %d", len(pt), h.OrigLen)
	}
	return pt
}

func TestRoundTrip(t *testing.T) {
	key, err := crypto.NewKey()
	if err != nil {
		t.Fatal(err)
	}
	src := []byte("<?php\necho \"hello from athena\\n\";\n")
	enc, err := EncodeBytes(src, key)
	if err != nil {
		t.Fatal(err)
	}
	if !Already(enc) {
		t.Fatal("encoded file not detected as already encoded")
	}
	got := decode(t, enc, key)
	if !bytes.Equal(got, src) {
		t.Fatalf("round trip mismatch:\n got %q\nwant %q", got, src)
	}
}

func TestTamperRejected(t *testing.T) {
	key, _ := crypto.NewKey()
	src := []byte("<?php echo 1;")
	enc, _ := EncodeBytes(src, key)
	// Flip a byte in the ciphertext region.
	enc[len(enc)-1] ^= 0xff
	idx := bytes.Index(enc, format.Magic)
	c := enc[idx:]
	aead, _ := chacha20poly1305.NewX(key)
	nonce := c[format.HeaderLen : format.HeaderLen+format.NonceLen]
	ct := c[format.HeaderLen+format.NonceLen:]
	if _, err := aead.Open(nil, nonce, ct, c[:format.HeaderLen]); err == nil {
		t.Fatal("tampered ciphertext should not authenticate")
	}
}

func TestWrongKeyRejected(t *testing.T) {
	key, _ := crypto.NewKey()
	other, _ := crypto.NewKey()
	enc, _ := EncodeBytes([]byte("<?php echo 1;"), key)
	idx := bytes.Index(enc, format.Magic)
	c := enc[idx:]
	aead, _ := chacha20poly1305.NewX(other)
	nonce := c[format.HeaderLen : format.HeaderLen+format.NonceLen]
	ct := c[format.HeaderLen+format.NonceLen:]
	if _, err := aead.Open(nil, nonce, ct, c[:format.HeaderLen]); err == nil {
		t.Fatal("wrong key should not authenticate")
	}
}
