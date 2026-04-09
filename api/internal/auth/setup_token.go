// Package auth provides an ephemeral setup token used to gate the
// initial owner-account registration endpoint.
//
// The token is generated at startup if and only if the users table is
// empty. It lives in memory only (no disk persistence) and is cleared
// after the master account is successfully created. A restart while the
// database is still empty produces a new token.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"sync"
)

// SetupTokenStore holds the single ephemeral setup token that protects
// POST /v1/auth/register during the initial bootstrap window.
type SetupTokenStore struct {
	mu    sync.RWMutex
	token string
}

// NewSetupTokenStore returns an empty store. Callers must explicitly
// invoke Generate when initial setup is required; otherwise the store
// remains inactive and all match attempts fail.
func NewSetupTokenStore() *SetupTokenStore {
	return &SetupTokenStore{}
}

// Generate produces a new 8-character hex token sourced from crypto/rand
// and stores it in the formatted form "xxxx-xxxx". The formatted token
// is returned so the caller can present it to the operator.
func (s *SetupTokenStore) Generate() (string, error) {
	raw := make([]byte, 4)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	encoded := hex.EncodeToString(raw)
	formatted := encoded[:4] + "-" + encoded[4:]

	s.mu.Lock()
	s.token = formatted
	s.mu.Unlock()

	return formatted, nil
}

// IsActive reports whether a setup token is currently loaded. It is
// used to distinguish "invalid code" from "no setup pending" without
// leaking the token itself.
func (s *SetupTokenStore) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token != ""
}

// Match returns true only when a token is active and the candidate is
// identical to the stored token after case-folding the hex characters.
// The stored token is always lowercase (hex.EncodeToString output) so
// the candidate is lowercased before the constant-time comparison.
func (s *SetupTokenStore) Match(candidate string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.token == "" {
		return false
	}
	normalized := strings.ToLower(candidate)
	return subtle.ConstantTimeCompare([]byte(s.token), []byte(normalized)) == 1
}

// Clear removes the stored token. Called after a successful owner
// registration so that no further /register attempts can ever succeed
// for the lifetime of the process.
func (s *SetupTokenStore) Clear() {
	s.mu.Lock()
	s.token = ""
	s.mu.Unlock()
}
