package main

import (
	"flag"
	"log"

	"github.com/Kr3mu/runfive/internal/api"
	"github.com/gofiber/fiber/v3"
)

var appConfig = fiber.Config{
	ServerHeader: "RunFive API",
	AppName:      "RunFive API",
	ErrorHandler: func(c fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}
		return c.Status(code).JSON(map[string]string{
			"error": err.Error(),
		})
	},
}

var listenConfig = fiber.ListenConfig{
	DisableStartupMessage: true,
}

var listenPort = flag.String("port", "5000", "Starting ")

func main() {
	app := api.New(appConfig)

	log.Printf("Serving backend on: %s", *listenPort)
	log.Fatal(app.Listen(":"+*listenPort, listenConfig))
}
