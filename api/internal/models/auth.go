// Package models provides request and response DTOs for authentication endpoints.
package models

// LoginRequest is the body for POST /v1/auth/login.
type LoginRequest struct {
	// Username of the account
	Username string `json:"username" validate:"required,min=3,max=32"`
	// Plaintext password
	Password string `json:"password" validate:"required,min=8"`
}

// RegisterRequest is the body for POST /v1/auth/register (master account setup).
type RegisterRequest struct {
	// Desired username for the master account
	Username string `json:"username" validate:"required,min=3,max=32"`
	// Plaintext password (min 8 chars)
	Password string `json:"password" validate:"required,min=8"`
	// Code is the formatted setup token ("xxxx-xxxx") printed to the
	// server console at first startup. Required to bootstrap the owner.
	Code string `json:"code" validate:"required,len=9"`
}

// MeResponse is returned by GET /v1/auth/me.
//
// TODO: Add a ServerRoles field that maps server ID (TOML dir name) ->
// role/permissions, so the frontend knows what the user can do per server.
type MeResponse struct {
	// User database ID
	ID uint `json:"id"`
	// Username
	Username string `json:"username"`
	// IsOwner indicates whether this user is the owner (master account).
	IsOwner bool `json:"isOwner"`
	// Providers contains linked authentication providers.
	Providers ProviderInfo `json:"providers"`
}

// ProviderInfo contains optional linked OAuth provider details.
type ProviderInfo struct {
	// Cfx is the Cfx.re account info, nil if not linked.
	Cfx *CfxInfo `json:"cfx"`
	// Discord is the Discord account info, nil if not linked.
	Discord *DiscordInfo `json:"discord"`
}

// CfxInfo contains Cfx.re (Discourse) account details.
type CfxInfo struct {
	// ID is the Discourse user ID on forum.cfx.re.
	ID int `json:"id"`
	// Username is the Discourse username.
	Username string `json:"username"`
	// AvatarURL is the avatar URL template.
	AvatarURL string `json:"avatarUrl"`
}

// DiscordInfo contains Discord account details (planned).
type DiscordInfo struct {
	// ID is the Discord user ID (snowflake).
	ID string `json:"id"`
	// Username is the Discord username.
	Username string `json:"username"`
	// Avatar is the Discord avatar hash.
	Avatar string `json:"avatar"`
}

// SessionResponse is a single entry in the GET /v1/auth/sessions list.
type SessionResponse struct {
	// ID is the session database ID.
	ID uint `json:"id"`
	// UserAgent is the client User-Agent.
	UserAgent string `json:"userAgent"`
	// CreatedAt is when the session was created.
	CreatedAt string `json:"createdAt"`
	// LastSeenAt is when the session was last active.
	LastSeenAt string `json:"lastSeenAt"`
	// IsCurrent indicates whether this is the session making the current request.
	IsCurrent bool `json:"isCurrent"`
}

// SetupStatusResponse is returned by GET /v1/auth/setup-status.
type SetupStatusResponse struct {
	// NeedsSetup is true if no users exist and the master account needs to be created.
	NeedsSetup bool `json:"needsSetup"`
}
