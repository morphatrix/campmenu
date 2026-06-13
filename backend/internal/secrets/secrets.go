// Package secrets provides transparent encryption-at-rest for sensitive setting
// values (SMTP password, external DB DSN) stored in app_settings.
//
// The key comes from SETTINGS_ENC_KEY. When unset, the Cipher is a no-op so the
// app still works (values stay plaintext, as before) — encryption is opt-in.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log/slog"
	"strings"
)

// encPrefix tags an encrypted value so we can tell ciphertext from legacy
// plaintext (and migrate the latter on write).
const encPrefix = "enc:v1:"

// Cipher encrypts/decrypts strings with AES-256-GCM. The zero value (no key) is
// a valid no-op cipher.
type Cipher struct {
	aead    cipher.AEAD
	enabled bool
}

// New derives a 32-byte key from the passphrase (any length) via SHA-256. An
// empty passphrase yields a disabled cipher.
func New(passphrase string) *Cipher {
	if passphrase == "" {
		slog.Warn("SETTINGS_ENC_KEY not set — sensitive settings (SMTP/DB password) are stored in plaintext")
		return &Cipher{}
	}
	sum := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		slog.Error("secrets: cipher init failed, falling back to plaintext", "error", err)
		return &Cipher{}
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		slog.Error("secrets: GCM init failed, falling back to plaintext", "error", err)
		return &Cipher{}
	}
	return &Cipher{aead: aead, enabled: true}
}

// Enabled reports whether a key is configured.
func (c *Cipher) Enabled() bool { return c.enabled }

// IsEncrypted reports whether a stored value is ciphertext produced by Encrypt.
func IsEncrypted(value string) bool { return strings.HasPrefix(value, encPrefix) }

// Encrypt returns ciphertext for a non-empty plaintext when enabled; otherwise
// it returns the input unchanged (so disabling the key never corrupts data).
func (c *Cipher) Encrypt(plaintext string) string {
	if !c.enabled || plaintext == "" || IsEncrypted(plaintext) {
		return plaintext
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		slog.Error("secrets: nonce generation failed", "error", err)
		return plaintext
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(sealed)
}

// Decrypt reverses Encrypt. Plaintext (unprefixed) values pass through, so it is
// safe on legacy data. A prefixed value that cannot be decrypted returns "".
func (c *Cipher) Decrypt(value string) string {
	if !IsEncrypted(value) {
		return value
	}
	if !c.enabled {
		slog.Error("secrets: encrypted value found but SETTINGS_ENC_KEY is not set")
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, encPrefix))
	if err != nil || len(raw) < c.aead.NonceSize() {
		slog.Error("secrets: malformed ciphertext")
		return ""
	}
	nonce, ct := raw[:c.aead.NonceSize()], raw[c.aead.NonceSize():]
	pt, err := c.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		slog.Error("secrets: decryption failed (wrong key?)", "error", err)
		return ""
	}
	return string(pt)
}
