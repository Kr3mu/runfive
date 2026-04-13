// Package v1 provides V1 API route registration.
package v1

import (
	"github.com/Kr3mu/runfive/internal/artifacts"
	"github.com/Kr3mu/runfive/internal/auth"
	"github.com/Kr3mu/runfive/internal/permissions"
	"github.com/Kr3mu/runfive/internal/runtimepath"
	"github.com/Kr3mu/runfive/internal/serverfs"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// RegisterRouter mounts all v1 API routes on the provided router group.
func RegisterRouter(r fiber.Router, db *gorm.DB, sm *auth.SessionManager, cfx *auth.CfxAuth, fe *auth.FieldEncryptor, discord *auth.DiscordAuth, st *auth.SetupTokenStore, baseURL string, artifactsDir string) {
	artifactManager, err := artifacts.NewManager(artifactsDir)
	if err != nil {
		panic(err)
	}
	serverRegistry := serverfs.NewRegistry(runtimepath.Resolve("servers"), artifactManager)

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

	// Protected auth endpoints with RBAC permissions loaded
	protected := authGroup.Group("", auth.RequireAuth(sm, db), auth.LoadPermissions(db))
	protected.Post("/logout", authHandler.Logout)
	protected.Get("/me", authHandler.Me)
	protected.Get("/sessions", authHandler.Sessions)
	protected.Delete("/sessions/:id", authHandler.RevokeSession)

	// Master-only endpoints (owner-only, not permission-based)
	master := protected.Group("/master", auth.RequireMaster)
	master.Post("/savediscord", authHandler.SaveDiscordAuthentication)
	master.Get("/getdiscord", authHandler.GetDiscordAuthentication)

	// Invite endpoints — public routes MUST be registered before any
	// auth middleware is mounted on the parent router, otherwise Fiber's
	// Use() will gate them too.
	inviteHandler := NewInviteHandler(db, sm, baseURL)
	inviteGroup := r.Group("/invites")
	inviteGroup.Get("/:token/validate", inviteHandler.Validate)
	inviteGroup.Post("/:token/accept", inviteHandler.Accept)

	inviteProtected := inviteGroup.Group("", auth.RequireAuth(sm, db), auth.LoadPermissions(db))
	inviteProtected.Post("", auth.RequireGlobalPerm(permissions.GlobalUsers, permissions.ActionCreate), inviteHandler.Create)
	inviteProtected.Get("", auth.RequireGlobalPerm(permissions.GlobalUsers, permissions.ActionRead), inviteHandler.List)
	inviteProtected.Delete("/:id", auth.RequireGlobalPerm(permissions.GlobalUsers, permissions.ActionDelete), inviteHandler.Revoke)

	// User management endpoints (permission-based)
	userHandler := NewUserHandler(db, sm)
	userGroup := r.Group("/users", auth.RequireAuth(sm, db), auth.LoadPermissions(db))
	userGroup.Get("", auth.RequireGlobalPerm(permissions.GlobalUsers, permissions.ActionRead), userHandler.List)
	userGroup.Post("/:id/suspend", auth.RequireGlobalPerm(permissions.GlobalUsers, permissions.ActionUpdate), userHandler.Suspend)
	userGroup.Post("/:id/unsuspend", auth.RequireGlobalPerm(permissions.GlobalUsers, permissions.ActionUpdate), userHandler.Unsuspend)
	userGroup.Delete("/:id", auth.RequireGlobalPerm(permissions.GlobalUsers, permissions.ActionDelete), userHandler.Delete)

	// User role assignment endpoints (permission-based)
	userGroup.Put("/:id/global-role", auth.RequireGlobalPerm(permissions.GlobalUsers, permissions.ActionUpdate), userHandler.SetGlobalRole)
	userGroup.Get("/:id/server-roles", auth.RequireGlobalPerm(permissions.GlobalUsers, permissions.ActionRead), userHandler.ListServerRoles)
	userGroup.Put("/:id/server-roles/:serverId", auth.RequireGlobalPerm(permissions.GlobalUsers, permissions.ActionUpdate), userHandler.SetServerRole)
	userGroup.Delete("/:id/server-roles/:serverId", auth.RequireGlobalPerm(permissions.GlobalUsers, permissions.ActionUpdate), userHandler.RemoveServerRole)

	// Role management endpoints (permission-based)
	roleHandler := NewRoleHandler(db)
	roleGroup := r.Group("/roles", auth.RequireAuth(sm, db), auth.LoadPermissions(db))
	roleGroup.Get("", auth.RequireGlobalPerm(permissions.GlobalRoles, permissions.ActionRead), roleHandler.List)
	roleGroup.Post("", auth.RequireGlobalPerm(permissions.GlobalRoles, permissions.ActionCreate), roleHandler.Create)
	roleGroup.Get("/:id", auth.RequireGlobalPerm(permissions.GlobalRoles, permissions.ActionRead), roleHandler.Get)
	roleGroup.Put("/:id", auth.RequireGlobalPerm(permissions.GlobalRoles, permissions.ActionUpdate), roleHandler.Update)
	roleGroup.Delete("/:id", auth.RequireGlobalPerm(permissions.GlobalRoles, permissions.ActionDelete), roleHandler.Delete)

	// Server management endpoints (permission-based)
	serverHandler := NewServerHandler(serverRegistry, artifactManager)
	serverGroup := r.Group("/servers", auth.RequireAuth(sm, db), auth.LoadPermissions(db))
	serverGroup.Get("", serverHandler.List)
	serverGroup.Post("", auth.RequireGlobalPerm(permissions.GlobalServers, permissions.ActionCreate), serverHandler.Create)

	// Artifact management endpoints (permission-based)
	artifactHandler := NewArtifactHandler(artifactManager, serverRegistry)
	artifactGroup := r.Group("/artifacts", auth.RequireAuth(sm, db), auth.LoadPermissions(db))
	artifactGroup.Get("", auth.RequireGlobalPerm(permissions.GlobalServers, permissions.ActionCreate), artifactHandler.List)
	artifactGroup.Post("/download", auth.RequireGlobalPerm(permissions.GlobalServers, permissions.ActionCreate), artifactHandler.Download)
	artifactGroup.Delete("/:version", auth.RequireGlobalPerm(permissions.GlobalServers, permissions.ActionDelete), artifactHandler.Delete)

	// Permission schema endpoint (any authenticated user)
	permGroup := r.Group("/permissions", auth.RequireAuth(sm, db), auth.LoadPermissions(db))
	permGroup.Get("/schema", PermissionSchema)
}
