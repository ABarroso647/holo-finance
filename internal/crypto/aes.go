// Package crypto provides AES-256-GCM encryption for sensitive values stored in the DB.
// Encrypted values are prefixed with "enc:v1:" to distinguish them from legacy plaintext.
// Decrypt handles both formats, so existing plaintext tokens keep working until next write.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

const prefix = "enc:v1:"

// KeyFromEnv returns the 32-byte AES key.
// Priority: ENCRYPTION_KEY (hex-encoded 32 bytes) → derived from SESSION_SECRET → dev fallback.
func KeyFromEnv() []byte {
	if raw := os.Getenv("ENCRYPTION_KEY"); raw != "" {
		b, err := hex.DecodeString(raw)
		if err == nil && len(b) == 32 {
			return b
		}
	}
	if secret := os.Getenv("SESSION_SECRET"); secret != "" {
		h := sha256.Sum256([]byte(secret + ":plaid-token-key"))
		return h[:]
	}
	// Dev fallback — same warning is already issued for SESSION_SECRET
	h := sha256.Sum256([]byte("dev-secret-change-in-production:plaid-token-key"))
	return h[:]
}

// Encrypt encrypts plaintext with AES-256-GCM and returns "enc:v1:<base64url(nonce+ciphertext)>".
func Encrypt(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return prefix + base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a value produced by Encrypt.
// If the value does not have the "enc:v1:" prefix it is returned as-is (legacy plaintext).
func Decrypt(key []byte, value string) (string, error) {
	if !strings.HasPrefix(value, prefix) {
		return value, nil // legacy plaintext — transparently pass through
	}
	raw, err := base64.URLEncoding.DecodeString(strings.TrimPrefix(value, prefix))
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}
