// Package v1 provides authentication HTTP handlers for registration, login,
// logout, session management, and OAuth provider flows.
package v1

import (
	"errors"
	"fmt"
	"time"

	"github.com/Kr3mu/runfive/internal/auth"
	"github.com/Kr3mu/runfive/internal/models"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// AuthHandler groups all auth-related HTTP handlers and their dependencies.
type AuthHandler struct {
	db             *gorm.DB
	sm             *auth.SessionManager
	cfx            *auth.CfxAuth
	discord        *auth.DiscordAuth
	fieldEncryptor *auth.FieldEncryptor
}

// NewAuthHandler creates the auth handler with its dependencies.
func NewAuthHandler(db *gorm.DB, sm *auth.SessionManager, cfx *auth.CfxAuth, discord *auth.DiscordAuth, fe *auth.FieldEncryptor) *AuthHandler {
	return &AuthHandler{db: db, sm: sm, cfx: cfx, discord: discord, fieldEncryptor: fe}
}

// SetupStatus returns whether the application needs initial setup.
//
// GET /v1/auth/setup-status
func (h *AuthHandler) SetupStatus(c fiber.Ctx) error {
	var count int64
	if err := h.db.Model(&models.User{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}
	return c.JSON(models.SetupStatusResponse{NeedsSetup: count == 0})
}

// Register creates the master (owner) account. Only succeeds when no users exist.
//
// POST /v1/auth/register
func (h *AuthHandler) Register(c fiber.Ctx) error {
	var req models.RegisterRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if len(req.Username) < 3 || len(req.Username) > 32 {
		return fiber.NewError(fiber.StatusBadRequest, "username must be 3-32 characters")
	}
	if len(req.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	var count int64
	if err := h.db.Model(&models.User{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}
	if count > 0 {
		return fiber.NewError(fiber.StatusForbidden, "setup already completed")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to hash password")
	}

	user := models.User{
		Username:     req.Username,
		PasswordHash: &hash,
		IsOwner:      true,
	}
	if err := h.db.Create(&user).Error; err != nil {
		return fiber.NewError(fiber.StatusConflict, "username already taken")
	}

	if err := h.createSessionForUser(c, &user); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create session")
	}

	return c.Status(fiber.StatusCreated).JSON(buildMeResponse(&user))
}

// Login authenticates a user with username and password.
//
// POST /v1/auth/login
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	var user models.User
	if err := h.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if user.PasswordHash == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "password login not available for this account")
	}

	if !auth.CheckPassword(*user.PasswordHash, req.Password) {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	if err := h.createSessionForUser(c, &user); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create session")
	}

	return c.JSON(buildMeResponse(&user))
}

// Logout destroys the current session and clears the cookie.
//
// POST /v1/auth/logout
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	token := auth.GetSessionToken(c)
	if token == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	tokenHash := auth.HashToken(token)
	h.db.Where("token_hash = ?", tokenHash).Delete(&models.UserSession{})

	if err := h.sm.DestroySession(c, token); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to destroy session")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Me returns the currently authenticated user's profile.
//
// GET /v1/auth/me
func (h *AuthHandler) Me(c fiber.Ctx) error {
	user := auth.GetUser(c)
	if user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}
	return c.JSON(buildMeResponse(user))
}

// Sessions lists all active sessions for the authenticated user.
//
// GET /v1/auth/sessions
func (h *AuthHandler) Sessions(c fiber.Ctx) error {
	user := auth.GetUser(c)
	if user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	currentToken := auth.GetSessionToken(c)
	currentHash := auth.HashToken(currentToken)

	var sessions []models.UserSession
	if err := h.db.Where("user_id = ?", user.ID).Order("last_seen_at DESC").Find(&sessions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	response := make([]models.SessionResponse, 0, len(sessions))
	for _, s := range sessions {
		response = append(response, models.SessionResponse{
			ID:         s.ID,
			UserAgent:  s.UserAgent,
			CreatedAt:  s.CreatedAt.Format(time.RFC3339),
			LastSeenAt: s.LastSeenAt.Format(time.RFC3339),
			IsCurrent:  s.TokenHash == currentHash,
		})
	}

	return c.JSON(response)
}

// RevokeSession destroys a specific session by its ID.
// Users can only revoke their own sessions.
//
// DELETE /v1/auth/sessions/:id
func (h *AuthHandler) RevokeSession(c fiber.Ctx) error {
	user := auth.GetUser(c)
	if user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}

	sessionID := fiber.Params[uint](c, "id")
	if sessionID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid session id")
	}

	var userSession models.UserSession
	if err := h.db.Where("id = ? AND user_id = ?", sessionID, user.ID).First(&userSession).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "session not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	_ = h.sm.DestroySessionByHash(userSession.TokenHash)
	h.db.Delete(&userSession)

	currentToken := auth.GetSessionToken(c)
	if auth.HashToken(currentToken) == userSession.TokenHash {
		_ = h.sm.DestroySession(c, currentToken)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// createSessionForUser creates a session and tracks it in user_sessions.
func (h *AuthHandler) createSessionForUser(c fiber.Ctx, user *models.User) error {
	token, err := h.sm.CreateSession(c, user.ID)
	if err != nil {
		return err
	}

	userSession := models.UserSession{
		UserID:     user.ID,
		TokenHash:  auth.HashToken(token),
		UserAgent:  c.Get("User-Agent"),
		CreatedAt:  time.Now(),
		LastSeenAt: time.Now(),
	}
	return h.db.Create(&userSession).Error
}

// CfxRedirect initiates the Cfx.re authentication flow.
// If the user is already authenticated, this becomes an account-linking flow.
//
// GET /v1/auth/cfx
func (h *AuthHandler) CfxRedirect(c fiber.Ctx) error {
	var linkUserID *uint
	if user := auth.GetUser(c); user != nil {
		linkUserID = &user.ID
	}

	redirectURL, err := h.cfx.StartAuth(linkUserID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("failed to start cfx auth: %s", err))
	}

	return c.Redirect().To(redirectURL)
}

// CfxCallback handles the redirect back from forum.cfx.re.
//
// GET /v1/auth/cfx/callback
func (h *AuthHandler) CfxCallback(c fiber.Ctx) error {
	state := c.Query("state")
	payload := c.Query("payload")
	if state == "" || payload == "" {
		return c.Redirect().To("/?error=invalid_callback")
	}

	userData, apiKey, linkUserID, err := h.cfx.HandleCallback(state, payload)
	if err != nil {
		return c.Redirect().To("/?error=auth_failed")
	}

	encryptedKey, err := h.fieldEncryptor.Encrypt([]byte(apiKey))
	if err != nil {
		return c.Redirect().To("/?error=internal_error")
	}

	avatarURL := auth.CfxForumURL + userData.AvatarTemplate

	if linkUserID != nil {
		result := h.db.Model(&models.User{}).Where("id = ?", *linkUserID).Updates(map[string]interface{}{
			"cfx_id":         userData.ID,
			"cfx_username":   userData.Username,
			"cfx_avatar_url": avatarURL,
			"cfx_api_key":    encryptedKey,
		})
		if result.Error != nil {
			return c.Redirect().To("/dashboard?error=link_failed")
		}
		return c.Redirect().To("/dashboard")
	}

	var user models.User
	result := h.db.Where("cfx_id = ?", userData.ID).First(&user)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Redirect().To("/?error=database_error")
	}
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Redirect().To("/?error=account_not_found")
	}

	h.db.Model(&user).Updates(map[string]interface{}{
		"cfx_username":   userData.Username,
		"cfx_avatar_url": avatarURL,
		"cfx_api_key":    encryptedKey,
	})

	if err := h.createSessionForUser(c, &user); err != nil {
		return c.Redirect().To("/?error=session_failed")
	}

	return c.Redirect().To("/dashboard")
}

// DiscordRedirect initiates the Discord OAuth2 flow.
// If the user is already authenticated, this becomes an account-linking flow.
//
// GET /v1/auth/discord
func (h *AuthHandler) DiscordRedirect(c fiber.Ctx) error {
	var linkUserID *uint
	if user := auth.GetUser(c); user != nil {
		linkUserID = &user.ID
	}

	redirectURL, err := h.discord.StartAuth(linkUserID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("failed to start discord auth: %s", err))
	}

	return c.Redirect().To(redirectURL)
}

// DiscordCallback handles the redirect back from Discord with the OAuth2 code.
//
// GET /v1/auth/discord/callback
func (h *AuthHandler) DiscordCallback(c fiber.Ctx) error {
	state := c.Query("state")
	code := c.Query("code")
	if state == "" || code == "" {
		return c.Redirect().To("/?error=invalid_callback")
	}

	userData, _, linkUserID, err := h.discord.HandleCallback(state, code)
	if err != nil {
		return c.Redirect().To("/?error=auth_failed")
	}

	avatarURL := userData.AvatarURL()

	if linkUserID != nil {
		result := h.db.Model(&models.User{}).Where("id = ?", *linkUserID).Updates(map[string]interface{}{
			"discord_id":       userData.ID,
			"discord_username": userData.Username,
			"discord_avatar":   avatarURL,
		})
		if result.Error != nil {
			return c.Redirect().To("/dashboard?error=link_failed")
		}
		return c.Redirect().To("/dashboard")
	}

	var user models.User
	result := h.db.Where("discord_id = ?", userData.ID).First(&user)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Redirect().To("/?error=database_error")
	}
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return c.Redirect().To("/?error=account_not_found")
	}

	h.db.Model(&user).Updates(map[string]interface{}{
		"discord_username": userData.Username,
		"discord_avatar":   avatarURL,
	})

	if err := h.createSessionForUser(c, &user); err != nil {
		return c.Redirect().To("/?error=session_failed")
	}

	return c.Redirect().To("/dashboard")
}

// buildMeResponse converts a User model into the API response DTO.
func buildMeResponse(user *models.User) models.MeResponse {
	resp := models.MeResponse{
		ID:       user.ID,
		Username: user.Username,
		IsOwner:  user.IsOwner,
	}

	if user.CfxID != nil {
		avatarURL := ""
		if user.CfxAvatarURL != nil {
			avatarURL = *user.CfxAvatarURL
		}
		username := ""
		if user.CfxUsername != nil {
			username = *user.CfxUsername
		}
		resp.Providers.Cfx = &models.CfxInfo{
			ID:        *user.CfxID,
			Username:  username,
			AvatarURL: avatarURL,
		}
	}

	if user.DiscordID != nil {
		discordUsername := ""
		if user.DiscordUsername != nil {
			discordUsername = *user.DiscordUsername
		}
		avatar := ""
		if user.DiscordAvatar != nil {
			avatar = *user.DiscordAvatar
		}
		resp.Providers.Discord = &models.DiscordInfo{
			ID:       *user.DiscordID,
			Username: discordUsername,
			Avatar:   avatar,
		}
	}

	return resp
}
