// Package auth provides field-level AES-256-GCM encryption for storing
// sensitive values like Cfx.re API keys in the database.
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// FieldEncryptor encrypts and decrypts individual byte slices with AES-256-GCM.
type FieldEncryptor struct {
	// AES-GCM cipher instance
	gcm cipher.AEAD
}

// NewFieldEncryptor creates an encryptor for database field values.
func NewFieldEncryptor(key [32]byte) (*FieldEncryptor, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	return &FieldEncryptor{gcm: gcm}, nil
}

// Encrypt produces nonce || ciphertext || tag from plaintext.
func (e *FieldEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	return e.gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt reverses Encrypt, returning the original plaintext.
func (e *FieldEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, data := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return e.gcm.Open(nil, nonce, data, nil)
}
