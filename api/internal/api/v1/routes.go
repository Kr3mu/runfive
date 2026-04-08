// Package v1 provides V1 API route registration.
//
// Mounts auth endpoints (public and protected) under /v1/auth.
package v1

import (
	"github.com/Kr3mu/runfive/internal/auth"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// RegisterRouter mounts all v1 API routes on the provided router group.
func RegisterRouter(r fiber.Router, db *gorm.DB, sm *auth.SessionManager, cfx *auth.CfxAuth, fe *auth.FieldEncryptor) {
	authHandler := NewAuthHandler(db, sm, cfx, fe)
	authGroup := r.Group("/auth")

	authGroup.Get("/setup-status", authHandler.SetupStatus)
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)

	cfxGroup := authGroup.Group("/cfx", auth.OptionalAuth(sm, db))
	cfxGroup.Get("", authHandler.CfxRedirect)
	cfxGroup.Get("/callback", authHandler.CfxCallback)

	protected := authGroup.Group("", auth.RequireAuth(sm, db))
	protected.Post("/logout", authHandler.Logout)
	protected.Get("/me", authHandler.Me)
	protected.Get("/sessions", authHandler.Sessions)
	protected.Delete("/sessions/:id", authHandler.RevokeSession)
}
