package permissions

import (
	"gorm.io/gorm"

	"github.com/Kr3mu/runfive/internal/models"
)

// RoleMeta holds display metadata for a role (used in API responses).
type RoleMeta struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// ResolvedPermissions holds the fully resolved permission set for a user.
// Loaded once per request by the LoadPermissions middleware.
type ResolvedPermissions struct {
	// IsOwner is true if the user is the panel owner (bypasses all checks).
	IsOwner bool
	// Global holds panel-wide permissions from the user's global role.
	Global PermissionMap
	// Servers maps server ID -> resolved PermissionMap from server role.
	Servers map[string]PermissionMap
	// ServerRoles maps server ID -> role metadata for API responses.
	ServerRoles map[string]RoleMeta
	// GlobalRole holds the global role metadata, nil if none assigned.
	GlobalRole *RoleMeta
}

// LoadForUser queries the database and builds the complete ResolvedPermissions
// for the given user. For the owner, all permissions are granted on all known
// servers without any DB queries for roles.
func LoadForUser(db *gorm.DB, user *models.User) (*ResolvedPermissions, error) {
	rp := &ResolvedPermissions{
		IsOwner:     user.IsOwner,
		Global:      PermissionMap{},
		Servers:     make(map[string]PermissionMap),
		ServerRoles: make(map[string]RoleMeta),
	}

	if user.IsOwner {
		rp.Global = FullAccessMap(GlobalResourceActions)
		return rp, nil
	}

	if err := loadGlobalRole(db, user, rp); err != nil {
		return nil, err
	}

	if err := loadServerRoles(db, user.ID, rp); err != nil {
		return nil, err
	}

	return rp, nil
}

// loadGlobalRole loads the user's global role and parses its permissions.
func loadGlobalRole(db *gorm.DB, user *models.User, rp *ResolvedPermissions) error {
	if user.GlobalRoleID == nil {
		return nil
	}

	var role models.Role
	if err := db.First(&role, *user.GlobalRoleID).Error; err != nil {
		return err
	}

	rp.GlobalRole = &RoleMeta{
		ID:    role.ID,
		Name:  role.Name,
		Color: role.Color,
	}

	parsed, err := Parse(role.GlobalPerms)
	if err != nil {
		return err
	}
	rp.Global = parsed

	return nil
}

// loadServerRoles loads all server role assignments for the user and parses permissions.
func loadServerRoles(db *gorm.DB, userID uint, rp *ResolvedPermissions) error {
	var assignments []models.UserServerRole
	if err := db.Preload("Role").Where("user_id = ?", userID).Find(&assignments).Error; err != nil {
		return err
	}

	for i := range assignments {
		a := &assignments[i]
		parsed, err := Parse(a.Role.ServerPerms)
		if err != nil {
			return err
		}
		rp.Servers[a.ServerID] = parsed
		rp.ServerRoles[a.ServerID] = RoleMeta{
			ID:    a.Role.ID,
			Name:  a.Role.Name,
			Color: a.Role.Color,
		}
	}

	return nil
}
