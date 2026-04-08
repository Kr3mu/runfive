// Package auth provides an encrypted codec for session data using AES-256-GCM.
//
// Session payloads are gob-encoded then encrypted before being stored
// in the database, ensuring data is unreadable at rest even if the
// SQLite file is compromised.
package auth

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/gob"
	"fmt"
	"io"
	"time"
)

func init() {
	gob.Register(uint(0))
	gob.Register(map[string]interface{}{})
}

// sessionData is the gob-serialized payload stored in each session.
type sessionData struct {
	// Absolute session expiry
	Deadline time.Time
	// Key-value pairs set via session Put/Get
	Values map[string]interface{}
}

// EncryptedCodec encodes session data as gob + AES-256-GCM ciphertext.
type EncryptedCodec struct {
	// AES-GCM cipher instance derived from the 32-byte encryption key
	gcm cipher.AEAD
}

// NewEncryptedCodec creates a codec that encrypts session data with the
// provided 32-byte AES-256 key.
func NewEncryptedCodec(key [32]byte) (*EncryptedCodec, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	return &EncryptedCodec{gcm: gcm}, nil
}

// Encode serializes session values with gob and encrypts the result
// with AES-256-GCM using a random nonce.
func (c *EncryptedCodec) Encode(deadline time.Time, values map[string]interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(sessionData{
		Deadline: deadline,
		Values:   values,
	})
	if err != nil {
		return nil, fmt.Errorf("gob encode: %w", err)
	}

	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	return c.gcm.Seal(nonce, nonce, buf.Bytes(), nil), nil
}

// Decode decrypts AES-256-GCM ciphertext and deserializes the gob payload.
func (c *EncryptedCodec) Decode(b []byte) (time.Time, map[string]interface{}, error) {
	nonceSize := c.gcm.NonceSize()
	if len(b) < nonceSize {
		return time.Time{}, nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := b[:nonceSize], b[nonceSize:]
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return time.Time{}, nil, fmt.Errorf("decrypt: %w", err)
	}

	var data sessionData
	if err := gob.NewDecoder(bytes.NewReader(plaintext)).Decode(&data); err != nil {
		return time.Time{}, nil, fmt.Errorf("gob decode: %w", err)
	}

	return data.Deadline, data.Values, nil
}
