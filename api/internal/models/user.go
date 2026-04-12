// Package models provides the User domain model stored in the users table.
//
// Supports multiple authentication providers: local password, Cfx.re
// (Discourse User API Keys), and Discord (planned).
package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents an authenticated user account.
//
// Authorization is handled via RBAC: GlobalRoleID grants panel-wide permissions,
// while per-server roles are stored in the UserServerRole join table.
// IsOwner remains as the global superadmin bypass with access to everything.
type User struct {
	gorm.Model
	// Username is the unique login name chosen during registration.
	Username string `gorm:"uniqueIndex;not null"`
	// PasswordHash is the bcrypt hash of the password, NULL for OAuth-only accounts.
	PasswordHash *string `gorm:"column:password_hash"`
	// IsOwner is true for the first registered user (master account).
	IsOwner bool `gorm:"not null;default:false"`
	// SuspendedAt is set when the account is suspended; nil means active.
	SuspendedAt *time.Time
	// GlobalRoleID is the role granting panel-wide (non-server) permissions.
	// NULL means no global permissions beyond server-specific roles.
	GlobalRoleID *uint `gorm:"column:global_role_id"`
	// GlobalRole is the GORM association for eager loading.
	GlobalRole *Role `gorm:"foreignKey:GlobalRoleID"`

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
	// DiscordAccessToken is the AES-256-GCM encrypted OAuth2 access token.
	DiscordAccessToken []byte `gorm:"column:discord_access_token"`
}
