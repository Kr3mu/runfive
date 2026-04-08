// Package main is the application entry point. It loads configuration,
// connects to the database, initializes the session manager and auth
// providers, and starts the HTTP server.
package main

import (
	"errors"
	"flag"
	"log"

	"github.com/Kr3mu/runfive/internal/api"
	"github.com/Kr3mu/runfive/internal/auth"
	"github.com/Kr3mu/runfive/internal/config"
	"github.com/Kr3mu/runfive/internal/database"
	"github.com/gofiber/fiber/v3"
)

var appConfig = fiber.Config{
	ServerHeader: "RunFive API",
	AppName:      "RunFive API",
	ErrorHandler: func(c fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			code = fiberErr.Code
		}
		return c.Status(code).JSON(map[string]string{
			"error": err.Error(),
		})
	},
}

var listenConfig = fiber.ListenConfig{
	DisableStartupMessage: true,
}

var listenPort = flag.String("port", "", "HTTP listen port (overrides PORT env)")

func main() {
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *listenPort != "" {
		cfg.Port = *listenPort
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sm, err := auth.NewSessionManager(db, cfg.SessionEncryptKey)
	if err != nil {
		log.Fatalf("Failed to create session manager: %v", err)
	}

	fe, err := auth.NewFieldEncryptor(cfg.CfxAPIKeySecret)
	if err != nil {
		log.Fatalf("Failed to create field encryptor: %v", err)
	}

	cfx := auth.NewCfxAuth(cfg.BaseURL)

	app := api.New(appConfig, api.AppDeps{
		DB:  db,
		SM:  sm,
		Cfx: cfx,
		FE:  fe,
	})

	log.Printf("Serving backend on: %s", cfg.Port)
	log.Fatal(app.Listen(":"+cfg.Port, listenConfig))
}
