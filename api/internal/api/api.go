// Package api provides HTTP application setup and route registration.
//
// Configures Fiber middleware, mounts versioned API routes with
// authentication, and registers the SPA fallback handler.
package api

import (
	"github.com/Kr3mu/runfive/internal/auth"
	v1 "github.com/Kr3mu/runfive/internal/api/v1"
	"github.com/Kr3mu/runfive/internal/spa"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"gorm.io/gorm"
)

// AppDeps bundles all dependencies needed to construct the HTTP application.
type AppDeps struct {
	// DB is the GORM database connection.
	DB *gorm.DB
	// SM is the session manager.
	SM *auth.SessionManager
	// Cfx handles Cfx.re authentication.
	Cfx *auth.CfxAuth
	// FE encrypts sensitive database fields.
	FE *auth.FieldEncryptor
}

// New creates the Fiber application with all middleware and routes.
func New(appConfig fiber.Config, deps AppDeps) *fiber.App {
	app := fiber.New(appConfig)

	app.Use(logger.New())
	app.Use(helmet.New())

	setupRoutes(app, deps)
	spa.Register(app)

	return app
}

func setupRoutes(app *fiber.App, deps AppDeps) {
	v1Group := app.Group("/v1")
	v1.RegisterRouter(v1Group, deps.DB, deps.SM, deps.Cfx, deps.FE)
}
