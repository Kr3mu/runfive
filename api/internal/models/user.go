// Package models provides the User domain model stored in the users table.
//
// Supports multiple authentication providers: local password, Cfx.re
// (Discourse User API Keys), and Discord (planned).
package models

import "gorm.io/gorm"

// User represents an authenticated user account.
//
// TODO: Add server-scoped RBAC. Servers are file-based (TOML configs in a
// directory structure, not DB rows). Users need a mapping of user -> server ID
// (from TOML) -> role, stored in DB (e.g. UserServerRole join table).
// Roles define granular permissions per server (view, manage players, manage
// resources, console access, etc.). IsOwner remains as the global superadmin
// bypass that has access to all servers.
type User struct {
	gorm.Model
	// Username is the unique login name chosen during registration.
	Username string `gorm:"uniqueIndex;not null"`
	// PasswordHash is the bcrypt hash of the password, NULL for OAuth-only accounts.
	PasswordHash *string `gorm:"column:password_hash"`
	// IsOwner is true for the first registered user (master account).
	IsOwner bool `gorm:"not null;default:false"`

	// CfxID is the Discourse user ID from forum.cfx.re.
	CfxID *int `gorm:"uniqueIndex"`
	// CfxUsername is the Cfx.re forum username.
	CfxUsername *string
	// CfxAvatarURL is the Cfx.re avatar URL template (contains {size} placeholder).
	CfxAvatarURL *string `gorm:"column:cfx_avatar_url"`
	// CfxAPIKey is the AES-256-GCM encrypted Discourse User API Key for refreshing user data.
	CfxAPIKey []byte `gorm:"column:cfx_api_key"`

	// DiscordID is the Discord user ID (snowflake as string), planned for future auth.
	DiscordID *string `gorm:"uniqueIndex"`
	// DiscordUsername is the Discord username.
	DiscordUsername *string
	// DiscordAvatar is the Discord avatar hash.
	DiscordAvatar *string
}
