// Package models provides UserSession which tracks active login sessions per user
// for multi-session management. Each row maps to one SCS session token in the sessions table.
package models

import "time"

// UserSession records metadata for an active login session.
type UserSession struct {
	// ID is the auto-incremented primary key.
	ID uint `gorm:"primaryKey" json:"id"`
	// UserID is the foreign key to users.id; cascade-deletes when user is removed.
	UserID uint `gorm:"not null;index" json:"-"`
	// TokenHash is the SHA-256 hex digest of the SCS session token (never store raw tokens).
	TokenHash string `gorm:"not null;index" json:"-"`
	// UserAgent is the client User-Agent header at session creation.
	UserAgent string `json:"userAgent"`
	// CreatedAt is the timestamp when the session was created.
	CreatedAt time.Time `gorm:"not null" json:"createdAt"`
	// LastSeenAt is the timestamp of the last authenticated request on this session.
	LastSeenAt time.Time `gorm:"not null" json:"lastSeenAt"`
}
