package v1

import (
	"errors"
	"time"

	"github.com/Kr3mu/runfive/internal/auth"
	"github.com/Kr3mu/runfive/internal/models"
	"github.com/Kr3mu/runfive/internal/permissions"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// Type aliases for convenience within this file.
type permMap = permissions.PermissionMap

var permParse = permissions.Parse

// UserHandler groups user management HTTP handlers.
type UserHandler struct {
	db *gorm.DB
	sm *auth.SessionManager
}

// NewUserHandler creates the user handler with its dependencies.
func NewUserHandler(db *gorm.DB, sm *auth.SessionManager) *UserHandler {
	return &UserHandler{db: db, sm: sm}
}

// List returns all users with their role information.
// Requires global "users.read" permission (enforced by middleware).
//
// GET /v1/users
func (h *UserHandler) List(c fiber.Ctx) error {
	var users []models.User
	if err := h.db.Preload("GlobalRole").Order("created_at ASC").Find(&users).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	response := make([]models.UserListItem, 0, len(users))
	for _, u := range users {
		item := models.UserListItem{
			ID:          u.ID,
			Username:    u.Username,
			IsOwner:     u.IsOwner,
			HasPassword: u.PasswordHash != nil,
			CreatedAt:   u.CreatedAt.Format(time.RFC3339),
		}

		if u.SuspendedAt != nil {
			s := u.SuspendedAt.Format(time.RFC3339)
			item.SuspendedAt = &s
		}

		if u.GlobalRole != nil {
			item.GlobalRole = &models.RoleInfo{
				ID:    u.GlobalRole.ID,
				Name:  u.GlobalRole.Name,
				Color: u.GlobalRole.Color,
			}
		}

		var serverRoleCount int64
		h.db.Model(&models.UserServerRole{}).Where("user_id = ?", u.ID).Count(&serverRoleCount)
		item.ServerRoleCount = int(serverRoleCount)

		if u.CfxID != nil {
			username := ""
			if u.CfxUsername != nil {
				username = *u.CfxUsername
			}
			avatarURL := ""
			if u.CfxAvatarURL != nil {
				avatarURL = *u.CfxAvatarURL
			}
			item.Providers.Cfx = &models.CfxInfo{
				ID:        *u.CfxID,
				Username:  username,
				AvatarURL: avatarURL,
			}
		}

		if u.DiscordID != nil {
			discordUsername := ""
			if u.DiscordUsername != nil {
				discordUsername = *u.DiscordUsername
			}
			avatar := ""
			if u.DiscordAvatar != nil {
				avatar = *u.DiscordAvatar
			}
			item.Providers.Discord = &models.DiscordInfo{
				ID:       *u.DiscordID,
				Username: discordUsername,
				Avatar:   avatar,
			}
		}

		response = append(response, item)
	}

	return c.JSON(response)
}

// Suspend blocks a user's login and revokes all their sessions.
// Requires global "users.update" permission (enforced by middleware).
//
// POST /v1/users/:id/suspend
func (h *UserHandler) Suspend(c fiber.Ctx) error {
	caller := auth.GetUser(c)
	if caller == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	targetID := fiber.Params[uint](c, "id")
	if targetID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid user id")
	}

	var target models.User
	if err := h.db.First(&target, targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if target.IsOwner {
		return fiber.NewError(fiber.StatusForbidden, "cannot suspend owner")
	}
	if target.ID == caller.ID {
		return fiber.NewError(fiber.StatusForbidden, "cannot suspend yourself")
	}

	now := time.Now()
	if err := h.db.Model(&target).Update("suspended_at", now).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	h.revokeAllSessions(target.ID)

	return c.SendStatus(fiber.StatusNoContent)
}

// Unsuspend restores a suspended user's access.
// Requires global "users.update" permission (enforced by middleware).
//
// POST /v1/users/:id/unsuspend
func (h *UserHandler) Unsuspend(c fiber.Ctx) error {
	targetID := fiber.Params[uint](c, "id")
	if targetID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid user id")
	}

	var target models.User
	if err := h.db.First(&target, targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if err := h.db.Model(&target).Updates(map[string]interface{}{"suspended_at": nil}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Delete permanently removes a user, their sessions, and all role assignments.
// Requires global "users.delete" permission (enforced by middleware).
//
// DELETE /v1/users/:id
func (h *UserHandler) Delete(c fiber.Ctx) error {
	caller := auth.GetUser(c)
	if caller == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	targetID := fiber.Params[uint](c, "id")
	if targetID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid user id")
	}

	var target models.User
	if err := h.db.First(&target, targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if target.IsOwner {
		return fiber.NewError(fiber.StatusForbidden, "cannot delete owner")
	}
	if target.ID == caller.ID {
		return fiber.NewError(fiber.StatusForbidden, "cannot delete yourself")
	}

	h.revokeAllSessions(target.ID)

	// Remove server role assignments
	h.db.Where("user_id = ?", target.ID).Delete(&models.UserServerRole{})

	if err := h.db.Unscoped().Delete(&target).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// SetGlobalRole assigns or clears the global role for a user.
// Requires global "users.update" permission (enforced by middleware).
//
// PUT /v1/users/:id/global-role
func (h *UserHandler) SetGlobalRole(c fiber.Ctx) error {
	targetID := fiber.Params[uint](c, "id")
	if targetID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid user id")
	}

	var req models.SetGlobalRoleRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	var target models.User
	if err := h.db.First(&target, targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if target.IsOwner {
		return fiber.NewError(fiber.StatusForbidden, "cannot change owner role")
	}

	if req.RoleID != nil {
		var role models.Role
		if err := h.db.First(&role, *req.RoleID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "role not found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}

		if err := checkEscalation(c, role); err != nil {
			return err
		}
	}

	if err := h.db.Model(&target).Update("global_role_id", req.RoleID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ListServerRoles returns all server role assignments for a user.
// Requires global "users.read" permission (enforced by middleware).
//
// GET /v1/users/:id/server-roles
func (h *UserHandler) ListServerRoles(c fiber.Ctx) error {
	targetID := fiber.Params[uint](c, "id")
	if targetID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid user id")
	}

	var assignments []models.UserServerRole
	if err := h.db.Preload("Role").Where("user_id = ?", targetID).Find(&assignments).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	response := make([]models.UserServerRoleEntry, 0, len(assignments))
	for _, a := range assignments {
		response = append(response, models.UserServerRoleEntry{
			ServerID: a.ServerID,
			Role: models.RoleInfo{
				ID:    a.Role.ID,
				Name:  a.Role.Name,
				Color: a.Role.Color,
			},
		})
	}

	return c.JSON(response)
}

// SetServerRole assigns a role to a user on a specific server.
// Requires global "users.update" permission (enforced by middleware).
//
// PUT /v1/users/:id/server-roles/:serverId
func (h *UserHandler) SetServerRole(c fiber.Ctx) error {
	targetID := fiber.Params[uint](c, "id")
	if targetID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid user id")
	}
	serverID := c.Params("serverId")
	if serverID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing server id")
	}

	var req models.SetServerRoleRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	var target models.User
	if err := h.db.First(&target, targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if target.IsOwner {
		return fiber.NewError(fiber.StatusForbidden, "cannot assign server role to owner")
	}

	var role models.Role
	if err := h.db.First(&role, req.RoleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "role not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if err := checkEscalation(c, role); err != nil {
		return err
	}

	// Upsert: create or update the assignment
	var existing models.UserServerRole
	err := h.db.Where("user_id = ? AND server_id = ?", targetID, serverID).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		assignment := models.UserServerRole{
			UserID:   targetID,
			ServerID: serverID,
			RoleID:   req.RoleID,
		}
		if err := h.db.Create(&assignment).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}
	} else {
		if err := h.db.Model(&existing).Update("role_id", req.RoleID).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RemoveServerRole removes a user's role assignment on a specific server.
// Requires global "users.update" permission (enforced by middleware).
//
// DELETE /v1/users/:id/server-roles/:serverId
func (h *UserHandler) RemoveServerRole(c fiber.Ctx) error {
	targetID := fiber.Params[uint](c, "id")
	if targetID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid user id")
	}
	serverID := c.Params("serverId")
	if serverID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing server id")
	}

	result := h.db.Where("user_id = ? AND server_id = ?", targetID, serverID).Delete(&models.UserServerRole{})
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "assignment not found")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// revokeAllSessions destroys all sessions for a given user.
func (h *UserHandler) revokeAllSessions(userID uint) {
	var sessions []models.UserSession
	if err := h.db.Where("user_id = ?", userID).Find(&sessions).Error; err != nil {
		return
	}
	for _, s := range sessions {
		_ = h.sm.DestroySessionByHash(s.TokenHash)
	}
	h.db.Where("user_id = ?", userID).Delete(&models.UserSession{})
}

// checkEscalation validates that a non-owner caller is not assigning a role
// with permissions they don't have themselves.
func checkEscalation(c fiber.Ctx, role models.Role) error {
	perms := auth.GetPermissions(c)
	if perms == nil || perms.IsOwner {
		return nil
	}

	globalPerms, err := parsePermissionMap(role.GlobalPerms)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "invalid role permissions")
	}
	if !globalPerms.IsSubsetOf(perms.Global) {
		return fiber.NewError(fiber.StatusForbidden, "cannot assign role with permissions you don't have")
	}

	serverPerms, err := parsePermissionMap(role.ServerPerms)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "invalid role permissions")
	}
	// For server perms, check against the union of all the caller's server perms
	callerServerUnion := mergeServerPerms(perms.Servers)
	if !serverPerms.IsSubsetOf(callerServerUnion) {
		return fiber.NewError(fiber.StatusForbidden, "cannot assign role with permissions you don't have")
	}

	return nil
}

// parsePermissionMap is a convenience wrapper around permissions.Parse.
func parsePermissionMap(jsonStr string) (permMap, error) {
	pm, err := permParse(jsonStr)
	return pm, err
}

// mergeServerPerms creates a union of all server permission maps.
func mergeServerPerms(servers map[string]permMap) permMap {
	merged := make(permMap)
	for _, perms := range servers {
		for resource, actions := range perms {
			if _, ok := merged[resource]; !ok {
				merged[resource] = make(map[string]bool)
			}
			for action, granted := range actions {
				if granted {
					merged[resource][action] = true
				}
			}
		}
	}
	return merged
}
