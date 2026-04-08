// Package auth provides Fiber middleware for session-based authentication.
//
// Loads the session from the cookie, resolves the user from the database,
// and stores both in Fiber locals for downstream handlers.
package auth

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Kr3mu/runfive/internal/models"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

const (
	// localsUserKey is the Fiber locals key for the authenticated User.
	localsUserKey = "user"
	// localsTokenKey is the Fiber locals key for the raw session token.
	localsTokenKey = "sessionToken"
	// lastSeenDebounce prevents updating last_seen_at more than once per minute.
	lastSeenDebounce = time.Minute
)

// RequireAuth returns Fiber middleware that enforces authentication.
//
// On valid session: sets c.Locals("user") to *models.User and
// c.Locals("sessionToken") to the raw token string, then calls next.
// On missing/invalid session: returns 401.
func RequireAuth(sm *SessionManager, db *gorm.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, token, err := sm.LoadSession(c)
		if err != nil {
			log.Printf("session load error: %v", err)
			return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
		}
		if userID == 0 {
			return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
		}

		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = sm.DestroySession(c, token)
				return fiber.NewError(fiber.StatusUnauthorized, "user not found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}

		c.Locals(localsUserKey, &user)
		c.Locals(localsTokenKey, token)

		go updateLastSeen(db, token)

		return c.Next()
	}
}

// OptionalAuth returns Fiber middleware that loads the session if present
// but does not require it. Downstream handlers can check GetUser(c).
func OptionalAuth(sm *SessionManager, db *gorm.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, token, err := sm.LoadSession(c)
		if err != nil || userID == 0 {
			return c.Next()
		}

		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			return c.Next()
		}

		c.Locals(localsUserKey, &user)
		c.Locals(localsTokenKey, token)
		return c.Next()
	}
}

// GetUser extracts the authenticated user from Fiber locals.
// Returns nil if the request is not authenticated.
func GetUser(c fiber.Ctx) *models.User {
	user, ok := c.Locals(localsUserKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// GetSessionToken extracts the raw session token from Fiber locals.
// Returns an empty string if the request is not authenticated.
func GetSessionToken(c fiber.Ctx) string {
	token, ok := c.Locals(localsTokenKey).(string)
	if !ok {
		return ""
	}
	return token
}

// updateLastSeen debounces the last_seen_at update to avoid excessive writes.
// Only updates if the existing last_seen_at is older than lastSeenDebounce.
func updateLastSeen(db *gorm.DB, token string) {
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(token)))
	now := time.Now()
	threshold := now.Add(-lastSeenDebounce)

	db.Model(&models.UserSession{}).
		Where("token_hash = ? AND last_seen_at < ?", tokenHash, threshold).
		Update("last_seen_at", now)
}
