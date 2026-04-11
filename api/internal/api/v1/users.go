package v1

import (
	"errors"
	"time"

	"github.com/Kr3mu/runfive/internal/auth"
	"github.com/Kr3mu/runfive/internal/models"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// UserHandler groups user management HTTP handlers.
type UserHandler struct {
	db *gorm.DB
	sm *auth.SessionManager
}

// NewUserHandler creates the user handler with its dependencies.
func NewUserHandler(db *gorm.DB, sm *auth.SessionManager) *UserHandler {
	return &UserHandler{db: db, sm: sm}
}

// List returns all users. Owner-only.
//
// GET /v1/users
func (h *UserHandler) List(c fiber.Ctx) error {
	user := auth.GetUser(c)
	if user == nil || !user.IsOwner {
		return fiber.NewError(fiber.StatusForbidden, "owner only")
	}

	var users []models.User
	if err := h.db.Order("created_at ASC").Find(&users).Error; err != nil {
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

// Suspend blocks a user's login and revokes all their sessions. Owner-only.
//
// POST /v1/users/:id/suspend
func (h *UserHandler) Suspend(c fiber.Ctx) error {
	caller := auth.GetUser(c)
	if caller == nil || !caller.IsOwner {
		return fiber.NewError(fiber.StatusForbidden, "owner only")
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

// Unsuspend restores a suspended user's access. Owner-only.
//
// POST /v1/users/:id/unsuspend
func (h *UserHandler) Unsuspend(c fiber.Ctx) error {
	caller := auth.GetUser(c)
	if caller == nil || !caller.IsOwner {
		return fiber.NewError(fiber.StatusForbidden, "owner only")
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

	if err := h.db.Model(&target).Updates(map[string]interface{}{"suspended_at": nil}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Delete permanently removes a user and all their sessions. Owner-only.
//
// DELETE /v1/users/:id
func (h *UserHandler) Delete(c fiber.Ctx) error {
	caller := auth.GetUser(c)
	if caller == nil || !caller.IsOwner {
		return fiber.NewError(fiber.StatusForbidden, "owner only")
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

	if err := h.db.Unscoped().Delete(&target).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
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
