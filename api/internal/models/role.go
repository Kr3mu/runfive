// Package models provides the Role domain model for RBAC.
//
// Roles define named permission sets with JSON blobs for global (panel-wide)
// and server-scoped permissions. Roles are defined once and assigned per-server
// via UserServerRole.
package models

import "gorm.io/gorm"

// Role defines a named set of permissions assignable to users.
// GlobalPerms controls panel-wide access (user management, settings).
// ServerPerms controls per-server resource access (players, console, bans).
type Role struct {
	gorm.Model
	// Name is the unique human-readable role name (e.g. "Admin", "Moderator").
	Name string `gorm:"uniqueIndex;not null" json:"name"`
	// Description is an optional explanation of what this role grants.
	Description string `json:"description"`
	// Color is a hex color for UI badges (e.g. "#ef4444").
	Color string `gorm:"not null;default:'#6b7280'" json:"color"`
	// GlobalPerms is a JSON blob of panel-wide permissions.
	// Structure: {"users": {"read": true, "create": true, ...}, "roles": {...}, ...}
	GlobalPerms string `gorm:"column:global_perms;type:text;not null;default:'{}'" json:"globalPerms"`
	// ServerPerms is a JSON blob of per-server resource permissions.
	// Structure: {"players": {"read": true, "create": true, ...}, "console": {...}, ...}
	ServerPerms string `gorm:"column:server_perms;type:text;not null;default:'{}'" json:"serverPerms"`
	// IsSystem marks roles created by the seed. System roles cannot be deleted.
	IsSystem bool `gorm:"not null;default:false" json:"isSystem"`
	// Position determines display order (lower = higher priority).
	Position int `gorm:"not null;default:0" json:"position"`
}
