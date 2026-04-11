// Package v1 provides V1 API route registration.
package v1

import (
	"github.com/Kr3mu/runfive/internal/auth"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// RegisterRouter mounts all v1 API routes on the provided router group.
//
// TODO: Add server management routes under /v1/servers with RBAC middleware.
// Servers are file-based (TOML configs in a directory structure), not DB rows.
// The API reads available servers from the TOML directory at startup.
//   - POST   /v1/servers              (owner only: create server dir + TOML, install server)
//   - GET    /v1/servers              (list servers the user has access to, filtered by RBAC)
//   - GET    /v1/servers/:id          (requires server view permission)
//   - PUT    /v1/servers/:id/roles    (owner only: assign user roles per server)
//   - DELETE /v1/servers/:id          (owner only: remove server dir)
// Each server endpoint should use RequirePermission middleware to check
// the user's role on that specific server.
//
// TODO: Add TOML config reader that scans the servers/ directory structure
// and parses each server.toml into a typed Go struct. Needs a file watcher
// or reload endpoint to pick up config changes without restart.

func RegisterRouter(r fiber.Router, db *gorm.DB, sm *auth.SessionManager, cfx *auth.CfxAuth, fe *auth.FieldEncryptor, discord *auth.DiscordAuth, st *auth.SetupTokenStore, baseURL string) {
	authHandler := NewAuthHandler(db, sm, cfx, fe, discord, st)
	authGroup := r.Group("/auth")

	authGroup.Get("/setup-status", authHandler.SetupStatus)
	authGroup.Get("/discord-status", authHandler.DiscordStatus)
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)

	cfxGroup := authGroup.Group("/cfx", auth.OptionalAuth(sm, db))
	cfxGroup.Get("", authHandler.CfxRedirect)
	cfxGroup.Get("/callback", authHandler.CfxCallback)

	discordGroup := authGroup.Group("/discord", auth.OptionalAuth(sm, db))
	discordGroup.Get("", authHandler.DiscordRedirect)
	discordGroup.Get("/callback", authHandler.DiscordCallback)

	protected := authGroup.Group("", auth.RequireAuth(sm, db))
	protected.Post("/logout", authHandler.Logout)
	protected.Get("/me", authHandler.Me)
	protected.Get("/sessions", authHandler.Sessions)
	protected.Delete("/sessions/:id", authHandler.RevokeSession)

  // Invite endpoints
	inviteHandler := NewInviteHandler(db, sm, baseURL)
	inviteGroup := r.Group("/invites")

	inviteGroup.Get("/:token/validate", inviteHandler.Validate)
	inviteGroup.Post("/:token/accept", inviteHandler.Accept)

	inviteProtected := inviteGroup.Group("", auth.RequireAuth(sm, db))
	inviteProtected.Post("", inviteHandler.Create)
	inviteProtected.Get("", inviteHandler.List)
	inviteProtected.Delete("/:id", inviteHandler.Revoke)

	// User management endpoints (owner-only)
	userHandler := NewUserHandler(db, sm)
	userGroup := r.Group("/users", auth.RequireAuth(sm, db))
	userGroup.Get("", userHandler.List)
	userGroup.Post("/:id/suspend", userHandler.Suspend)
	userGroup.Post("/:id/unsuspend", userHandler.Unsuspend)
	userGroup.Delete("/:id", userHandler.Delete)
  
	master := protected.Group("/master", auth.RequireMaster)
	master.Post("/savediscord", authHandler.SaveDiscordAuthentication)
	master.Get("/getdiscord", authHandler.GetDiscordAuthentication)
}
