package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// TokenCipher provides AES-256-GCM encryption/decryption for OAuth tokens at rest.
type TokenCipher struct {
	aead cipher.AEAD
}

// LoadOrCreate loads an existing 32-byte key from keyPath, or generates and saves a new one.
// The key file is created with 0600 permissions (owner-read-write only).
func LoadOrCreate(keyPath string) (*TokenCipher, error) {
	keyPath = filepath.Clean(keyPath)

	var key []byte
	data, err := os.ReadFile(keyPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read key file: %w", err)
		}
		key = make([]byte, 32)
		if _, genErr := io.ReadFull(rand.Reader, key); genErr != nil {
			return nil, fmt.Errorf("generate key: %w", genErr)
		}
		if writeErr := os.WriteFile(keyPath, key, 0o600); writeErr != nil {
			return nil, fmt.Errorf("write key file: %w", writeErr)
		}
		slog.Info("token cipher key created", "path", keyPath)
	} else {
		if len(data) != 32 {
			return nil, fmt.Errorf("invalid key length %d (expected 32)", len(data))
		}
		key = data
		slog.Info("token cipher key loaded", "path", keyPath)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	return &TokenCipher{aead: aead}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with a random 12-byte nonce.
// Returns base64url(nonce || ciphertext). Empty string is passed through unchanged.
func (c *TokenCipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sealed := c.aead.Seal(nil, nonce, []byte(plaintext), nil)
	combined := make([]byte, len(nonce)+len(sealed))
	copy(combined, nonce)
	copy(combined[len(nonce):], sealed)
	return base64.RawURLEncoding.EncodeToString(combined), nil
}

// Decrypt decrypts a base64url-encoded ciphertext produced by Encrypt.
// Empty string is passed through unchanged.
func (c *TokenCipher) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	data, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	nonceSize := c.aead.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	plaintext, err := c.aead.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}
