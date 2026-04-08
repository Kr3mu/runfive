// Package auth provides a server-side session manager built on top of a GORM-backed store.
//
// Uses the gormstore for persistence (same schema as SCS) but integrates
// natively with Fiber instead of net/http. Session tokens are SHA-256 hashed
// before being stored in the database so a DB leak does not expose valid tokens.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

const (
	// cookieName is the HTTP cookie name for the session token.
	cookieName = "session"
	// tokenLength is the number of random bytes in a session token (32 bytes = 43 base64 chars).
	tokenLength = 32
	// sessionLifetime is the absolute maximum session duration.
	sessionLifetime = 7 * 24 * time.Hour
)

// sessionStore defines the persistence interface for session data.
type sessionStore interface {
	Find(token string) (b []byte, found bool, err error)
	Commit(token string, b []byte, expiry time.Time) error
	Delete(token string) error
}

// SessionManager handles session lifecycle with encrypted, server-side storage.
type SessionManager struct {
	// store persists session data to the database
	store sessionStore
	// codec encrypts/decrypts session payloads before storage
	codec *EncryptedCodec
}

// NewSessionManager creates a session manager backed by the provided GORM database.
func NewSessionManager(db *gorm.DB, encryptKey [32]byte) (*SessionManager, error) {
	codec, err := NewEncryptedCodec(encryptKey)
	if err != nil {
		return nil, fmt.Errorf("create encrypted codec: %w", err)
	}

	store, err := newGormStore(db)
	if err != nil {
		return nil, fmt.Errorf("create gorm store: %w", err)
	}

	return &SessionManager{
		store: store,
		codec: codec,
	}, nil
}

// CreateSession generates a new session token, stores encrypted session data,
// and sets the session cookie on the Fiber response.
// Returns the raw session token (before hashing) for user_sessions tracking.
func (sm *SessionManager) CreateSession(c fiber.Ctx, userID uint) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	deadline := time.Now().Add(sessionLifetime)
	values := map[string]interface{}{
		"userID": userID,
	}

	encoded, err := sm.codec.Encode(deadline, values)
	if err != nil {
		return "", fmt.Errorf("encode session: %w", err)
	}

	hashedToken := hashToken(token)
	if err := sm.store.Commit(hashedToken, encoded, deadline); err != nil {
		return "", fmt.Errorf("commit session: %w", err)
	}

	sm.setCookie(c, token, deadline)
	return token, nil
}

// LoadSession reads the session cookie, looks up the session in the store,
// decrypts it, and returns the stored user ID.
// Returns 0 and an empty string if no valid session exists.
func (sm *SessionManager) LoadSession(c fiber.Ctx) (uint, string, error) {
	token := c.Cookies(cookieName)
	if token == "" {
		return 0, "", nil
	}

	hashedToken := hashToken(token)
	data, found, err := sm.store.Find(hashedToken)
	if err != nil {
		return 0, "", fmt.Errorf("find session: %w", err)
	}
	if !found {
		return 0, "", nil
	}

	deadline, values, err := sm.codec.Decode(data)
	if err != nil {
		_ = sm.store.Delete(hashedToken)
		return 0, "", err
	}

	if time.Now().After(deadline) {
		_ = sm.store.Delete(hashedToken)
		return 0, "", nil
	}

	userIDRaw, ok := values["userID"]
	if !ok {
		return 0, "", nil
	}

	userID, ok := userIDRaw.(uint)
	if !ok {
		return 0, "", nil
	}

	return userID, token, nil
}

// DestroySession removes the session from the store and clears the cookie.
func (sm *SessionManager) DestroySession(c fiber.Ctx, token string) error {
	hashedToken := hashToken(token)
	if err := sm.store.Delete(hashedToken); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	sm.clearCookie(c)
	return nil
}

// DestroySessionByHash removes a session from the store using the pre-hashed token.
// Used for revoking other sessions where only the hash from user_sessions is available.
func (sm *SessionManager) DestroySessionByHash(tokenHash string) error {
	return sm.store.Delete(tokenHash)
}

// HashToken computes the SHA-256 hex digest of a raw session token.
// Exported for use in user_sessions tracking.
func HashToken(token string) string {
	return hashToken(token)
}

func (sm *SessionManager) setCookie(c fiber.Ctx, token string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func (sm *SessionManager) clearCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func generateToken() (string, error) {
	b := make([]byte, tokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
