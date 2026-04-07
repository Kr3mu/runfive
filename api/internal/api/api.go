package api

import (
	v1 "github.com/Kr3mu/runfive/internal/api/v1"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

func New(appConfig fiber.Config) *fiber.App {
	app := fiber.New(appConfig)

	app.Use(logger.New())
	app.Use(helmet.New())

	SetupRoutes(app)

	return app
}

func SetupRoutes(app *fiber.App) {
	v1Group := app.Group("/v1")

	v1.RegisterRouter(v1Group)
}
