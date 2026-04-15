// Package api provides HTTP application setup and route registration.
package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"gorm.io/gorm"

	v1 "github.com/runfivedev/runfive/internal/api/v1"
	"github.com/runfivedev/runfive/internal/auth"
	"github.com/runfivedev/runfive/internal/spa"
)

// AppDeps bundles all dependencies needed to construct the HTTP application.
type AppDeps struct {
	// DB is the GORM database connection.
	DB *gorm.DB
	// ArtifactsDir is the filesystem root for shared artifact installs.
	ArtifactsDir string
	// SM is the session manager.
	SM *auth.SessionManager
	// Cfx handles Cfx.re authentication.
	Cfx *auth.CfxAuth
	// Discord handles Discord authentication.
	Discord *auth.DiscordAuth
	// FE encrypts sensitive database fields.
	FE *auth.FieldEncryptor
	// ST holds the ephemeral setup token gating the initial /register call.
	ST *auth.SetupTokenStore
	// BaseURL is the public base URL for constructing invite links.
	BaseURL string
}

// New creates the Fiber application with all middleware and routes.
//
// While an initial setup token is active the request logger middleware
// is suppressed on a per-request basis so the setup banner stays the
// only visible output in the terminal. Access logging resumes
// automatically once the owner account is created and the store is
// cleared — no restart required.
func New(appConfig *fiber.Config, deps *AppDeps) *fiber.App {
	app := fiber.New(*appConfig)

	setupActive := deps.ST != nil && deps.ST.IsActive()

	app.Use(logger.New(logger.Config{
		Next: func(_ fiber.Ctx) bool {
			return deps.ST != nil && deps.ST.IsActive()
		},
	}))
	app.Use(helmet.New())

	setupRoutes(app, deps)
	spa.Register(app, setupActive)

	return app
}

func setupRoutes(app *fiber.App, deps *AppDeps) {
	v1Group := app.Group("/v1")
	v1.RegisterRouter(v1Group, deps.DB, deps.SM, deps.Cfx, deps.FE, deps.Discord, deps.ST, deps.BaseURL, deps.ArtifactsDir)
}
