// Package main is the application entry point.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"

	"github.com/Kr3mu/runfive/internal/api"
	"github.com/Kr3mu/runfive/internal/auth"
	"github.com/Kr3mu/runfive/internal/config"
	"github.com/Kr3mu/runfive/internal/console"
	"github.com/Kr3mu/runfive/internal/database"
	"github.com/Kr3mu/runfive/internal/models"
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
	_ = godotenv.Load(".env")

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
	discord := auth.NewDiscordAuth(cfg.BaseURL, cfg.DiscordClientID, cfg.DiscordClientSecret)

	setupTokenStore := auth.NewSetupTokenStore()
	var userCount int64
	if err := db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		log.Fatalf("Failed to count users: %v", err)
	}

	var setupURL string
	if userCount == 0 {
		token, err := setupTokenStore.Generate()
		if err != nil {
			log.Fatalf("Failed to generate setup token: %v", err)
		}
		setupURL = fmt.Sprintf("%s/?setup=%s", cfg.BaseURL, token)
	}

	app := api.New(&appConfig, &api.AppDeps{
		DB:           db,
		ArtifactsDir: cfg.ArtifactsDir,
		SM:           sm,
		Cfx:          cfx,
		FE:           fe,
		Discord:      discord,
		ST:           setupTokenStore,
		BaseURL:      cfg.BaseURL,
	})

	if setupURL != "" {
		console.SetupBanner(setupURL)
	} else {
		log.Printf("Serving backend on: %s", cfg.Port)
	}
	log.Fatal(app.Listen(":"+cfg.Port, listenConfig))
}
