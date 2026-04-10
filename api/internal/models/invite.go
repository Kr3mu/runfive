package models

import (
	"time"

	"gorm.io/gorm"
)

// Invite represents a magic-link invitation token for new user registration.
type Invite struct {
	gorm.Model
	// TokenHash is the SHA-256 hash of the raw token (unique lookup key).
	TokenHash string `gorm:"uniqueIndex;not null"`
	// TokenRaw is the plaintext base64url token, retrievable from the pending list.
	TokenRaw string `gorm:"not null"`
	// CreatedBy is the user ID of the owner who generated the invite.
	CreatedBy uint `gorm:"not null"`
	// ExpiresAt is when the invite becomes invalid (24h after creation).
	ExpiresAt time.Time `gorm:"not null"`
	// UsedAt is set when the invite is redeemed; NULL means still pending.
	UsedAt *time.Time
	// UsedBy is the user ID of the account created from this invite.
	UsedBy *uint
}

// InviteCreateResponse is returned by POST /v1/invites.
type InviteCreateResponse struct {
	ID        uint   `json:"id"`
	Token     string `json:"token"`
	URL       string `json:"url"`
	ExpiresAt string `json:"expiresAt"`
}

// InviteListItem is a single entry in GET /v1/invites.
type InviteListItem struct {
	ID        uint   `json:"id"`
	Token     string `json:"token"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
}

// InviteValidateResponse is returned by GET /v1/invites/:token/validate.
type InviteValidateResponse struct {
	Valid     bool   `json:"valid"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

// InviteAcceptRequest is the body for POST /v1/invites/:token/accept.
type InviteAcceptRequest struct {
	Username string `json:"username" validate:"required,min=3,max=32"`
	Password string `json:"password" validate:"required,min=8"`
}
