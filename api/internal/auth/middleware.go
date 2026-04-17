// Package auth provides Fiber middleware for session-based authentication
// and RBAC permission enforcement.
//
// Loads the session from the cookie, resolves the user from the database,
// and stores both in Fiber locals for downstream handlers.
// Permission middleware loads role-based permissions once per request and
// provides RequireGlobalPerm / RequireServerPerm guards.
package auth

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/runfivedev/runfive/internal/models"
	"github.com/runfivedev/runfive/internal/permissions"
)

const (
	// localsUserKey is the Fiber locals key for the authenticated User.
	localsUserKey = "user"
	// localsTokenKey is the Fiber locals key for the raw session token.
	localsTokenKey = "sessionToken"
	// localsPermsKey is the Fiber locals key for resolved permissions.
	localsPermsKey = "permissions"
	// lastSeenDebounce prevents updating last_seen_at more than once per minute.
	lastSeenDebounce = time.Minute
)

// RequireAuth returns Fiber middleware that enforces authentication.
//
// On valid session: sets c.Locals("user") to *models.User and
// c.Locals("sessionToken") to the raw token string, then calls next.
// On missing/invalid session: returns 401.
//
// TODO: Add RequirePermission(server, permission) middleware that checks the
// user's role on a specific server. Should load the user-server-role mapping
// and verify the required permission before calling next. IsOwner bypasses.
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

		if user.SuspendedAt != nil {
			_ = sm.DestroySession(c, token)
			return fiber.NewError(fiber.StatusUnauthorized, "account suspended")
		}

		c.Locals(localsUserKey, &user)
		c.Locals(localsTokenKey, token)

		go updateLastSeen(db, token)

		return c.Next()
	}
}

// LoadPermissions returns Fiber middleware that loads the user's RBAC permissions
// once per request. Must be chained after RequireAuth. Stores the resolved
// permissions in c.Locals("permissions") for use by RequireGlobalPerm and
// RequireServerPerm.
func LoadPermissions(db *gorm.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := GetUser(c)
		if user == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
		}
		perms, err := permissions.LoadForUser(db, user)
		if err != nil {
			log.Printf("permission load error for user %d: %v", user.ID, err)
			return fiber.NewError(fiber.StatusInternalServerError, "failed to load permissions")
		}
		c.Locals(localsPermsKey, perms)
		return c.Next()
	}
}

// RequireGlobalPerm returns Fiber middleware that checks a panel-wide permission.
// Must be chained after LoadPermissions. Owner always bypasses.
func RequireGlobalPerm(resource, action string) fiber.Handler {
	return func(c fiber.Ctx) error {
		perms := GetPermissions(c)
		if perms == nil {
			return fiber.NewError(fiber.StatusForbidden, "no permissions loaded")
		}
		if perms.IsOwner {
			return c.Next()
		}
		if !perms.Global.Has(resource, action) {
			return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
		}
		return c.Next()
	}
}

// RequireServerPerm returns Fiber middleware that checks a per-server permission.
// Reads the server ID from the ":serverId" route parameter.
// Must be chained after LoadPermissions. Owner always bypasses.
func RequireServerPerm(resource, action string) fiber.Handler {
	return func(c fiber.Ctx) error {
		perms := GetPermissions(c)
		if perms == nil {
			return fiber.NewError(fiber.StatusForbidden, "no permissions loaded")
		}
		if perms.IsOwner {
			return c.Next()
		}
		serverID := c.Params("serverId")
		if serverID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing server ID")
		}
		serverPerms, ok := perms.Servers[serverID]
		if !ok {
			return fiber.NewError(fiber.StatusForbidden, "no access to this server")
		}
		if !serverPerms.Has(resource, action) {
			return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
		}
		return c.Next()
	}
}

// GetPermissions extracts the resolved permissions from Fiber locals.
// Returns nil if permissions have not been loaded.
func GetPermissions(c fiber.Ctx) *permissions.ResolvedPermissions {
	p, ok := c.Locals(localsPermsKey).(*permissions.ResolvedPermissions)
	if !ok {
		return nil
	}
	return p
}

// RequireMaster is Fiber middleware that restricts access to the owner (master)
// account. Must be chained after RequireAuth so that c.Locals("user") is set.
// Returns 403 for any non-owner user.
func RequireMaster(c fiber.Ctx) error {
	user, ok := c.Locals(localsUserKey).(*models.User)

	if !ok {
		return fiber.NewError(fiber.StatusForbidden, "Not a master")
	}

	if user.IsOwner {
		return c.Next()
	}

	return fiber.NewError(fiber.StatusForbidden, "Not a master")
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
