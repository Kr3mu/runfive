package api

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"github.com/gofiber/fiber/v3"
)

func New(appConfig fiber.Config, humaConfig huma.Config) *fiber.App {
	app := fiber.New(appConfig)
	humaApp := humafiber.New(app, humaConfig)

	SetupRoutes(&humaApp)

	return app
}

func SetupRoutes(app *huma.API) {

}
