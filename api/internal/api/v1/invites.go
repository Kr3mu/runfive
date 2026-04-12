package v1

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Kr3mu/runfive/internal/auth"
	"github.com/Kr3mu/runfive/internal/models"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

const inviteTTL = 24 * time.Hour

// InviteHandler groups invite-related HTTP handlers.
type InviteHandler struct {
	db      *gorm.DB
	sm      *auth.SessionManager
	baseURL string
}

// NewInviteHandler creates the invite handler with its dependencies.
func NewInviteHandler(db *gorm.DB, sm *auth.SessionManager, baseURL string) *InviteHandler {
	return &InviteHandler{db: db, sm: sm, baseURL: baseURL}
}

// generateToken creates a 32-byte random token and its SHA-256 hash.
func generateToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(raw))
	hash = base64.RawURLEncoding.EncodeToString(h[:])
	return raw, hash, nil
}

// hashToken produces the SHA-256 hash of a raw token string.
func hashInviteToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// lookupPendingInvite finds a valid (unused, unexpired) invite by raw token.
func (h *InviteHandler) lookupPendingInvite(rawToken string) (*models.Invite, error) {
	tokenHash := hashInviteToken(rawToken)
	var invite models.Invite
	if err := h.db.Where("token_hash = ?", tokenHash).First(&invite).Error; err != nil {
		return nil, err
	}
	if invite.UsedAt != nil {
		return nil, errors.New("invite already used")
	}
	if time.Now().After(invite.ExpiresAt) {
		return nil, errors.New("invite expired")
	}
	return &invite, nil
}

// Create generates a new invite token.
// Requires global "users.create" permission (enforced by middleware).
//
// POST /v1/invites
func (h *InviteHandler) Create(c fiber.Ctx) error {
	user := auth.GetUser(c)
	if user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	raw, hash, err := generateToken()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate token")
	}

	invite := models.Invite{
		TokenHash: hash,
		TokenRaw:  raw,
		CreatedBy: user.ID,
		ExpiresAt: time.Now().Add(inviteTTL),
	}
	if err := h.db.Create(&invite).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create invite")
	}

	return c.Status(fiber.StatusCreated).JSON(models.InviteCreateResponse{
		ID:        invite.ID,
		Token:     raw,
		URL:       fmt.Sprintf("%s/invite/accept?token=%s", h.baseURL, raw),
		ExpiresAt: invite.ExpiresAt.Format(time.RFC3339),
	})
}

// List returns all pending (unused, unexpired) invites.
// Requires global "users.read" permission (enforced by middleware).
//
// GET /v1/invites
func (h *InviteHandler) List(c fiber.Ctx) error {
	var invites []models.Invite
	if err := h.db.Where("used_at IS NULL AND expires_at > ?", time.Now()).
		Order("created_at DESC").Find(&invites).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	response := make([]models.InviteListItem, 0, len(invites))
	for _, inv := range invites {
		response = append(response, models.InviteListItem{
			ID:        inv.ID,
			Token:     inv.TokenRaw,
			CreatedAt: inv.CreatedAt.Format(time.RFC3339),
			ExpiresAt: inv.ExpiresAt.Format(time.RFC3339),
		})
	}

	return c.JSON(response)
}

// Revoke deletes an invite by ID.
// Requires global "users.delete" permission (enforced by middleware).
//
// DELETE /v1/invites/:id
func (h *InviteHandler) Revoke(c fiber.Ctx) error {
	id := fiber.Params[uint](c, "id")
	if id == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid invite id")
	}

	result := h.db.Delete(&models.Invite{}, id)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "invite not found")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Validate checks whether an invite token is valid (exists, not used, not expired).
//
// GET /v1/invites/:token/validate
func (h *InviteHandler) Validate(c fiber.Ctx) error {
	rawToken := c.Params("token")
	invite, err := h.lookupPendingInvite(rawToken)
	if err != nil {
		return c.JSON(models.InviteValidateResponse{Valid: false})
	}

	return c.JSON(models.InviteValidateResponse{
		Valid:     true,
		ExpiresAt: invite.ExpiresAt.Format(time.RFC3339),
	})
}

// Accept creates a new user account from an invite token (password registration).
//
// POST /v1/invites/:token/accept
func (h *InviteHandler) Accept(c fiber.Ctx) error {
	rawToken := c.Params("token")
	invite, err := h.lookupPendingInvite(rawToken)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid or expired invite")
	}

	var req models.InviteAcceptRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 || len(req.Username) > 32 {
		return fiber.NewError(fiber.StatusBadRequest, "username must be 3-32 characters")
	}
	if len(req.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to hash password")
	}

	now := time.Now()
	user := models.User{
		Username:     req.Username,
		PasswordHash: &hash,
		IsOwner:      false,
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return tx.Model(invite).Updates(map[string]interface{}{
			"used_at": now,
			"used_by": user.ID,
		}).Error
	})
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, "username already taken")
	}

	if err := createSessionForUser(h.sm, h.db, c, &user); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create session")
	}

	return c.Status(fiber.StatusCreated).JSON(buildMeResponse(&user, nil))
}
