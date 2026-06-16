// Package crypto wraps XChaCha20-Poly1305 (IETF) for Athena containers. The
// construction is wire-compatible with libsodium's
// crypto_aead_xchacha20poly1305_ietf_* used by the decoder extension.
package crypto

import (
	"crypto/rand"
	"fmt"

	"athena/internal/format"

	"golang.org/x/crypto/chacha20poly1305"
)

// NewKey returns a fresh random 256-bit key.
func NewKey() ([]byte, error) {
	k := make([]byte, format.KeyLen)
	if _, err := rand.Read(k); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	return k, nil
}

// Seal encrypts plaintext with key, authenticating aad (the container header),
// and returns nonce and ciphertext (which includes the Poly1305 tag).
func Seal(key, plaintext, aad []byte) (nonce, ciphertext []byte, err error) {
	if len(key) != format.KeyLen {
		return nil, nil, fmt.Errorf("athena: key must be %d bytes", format.KeyLen)
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, format.NonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext = aead.Seal(nil, nonce, plaintext, aad)
	return nonce, ciphertext, nil
}
