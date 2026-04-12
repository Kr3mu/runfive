// Package models provides the UserServerRole join model for per-server RBAC.
//
// Maps a user to a role on a specific server. ServerID is a string matching
// the TOML directory name (not a database foreign key).
package models

import "gorm.io/gorm"

// UserServerRole maps a user to a role on a specific server.
// Each user can have at most one role per server (enforced by unique index).
type UserServerRole struct {
	gorm.Model
	// UserID is the user this assignment belongs to.
	UserID uint `gorm:"not null;uniqueIndex:idx_user_server" json:"userId"`
	// ServerID is the server directory name from the TOML config.
	ServerID string `gorm:"not null;uniqueIndex:idx_user_server" json:"serverId"`
	// RoleID is the role assigned to this user on this server.
	RoleID uint `gorm:"not null" json:"roleId"`
	// Role is the GORM association for eager loading.
	Role Role `gorm:"foreignKey:RoleID" json:"role"`
}
