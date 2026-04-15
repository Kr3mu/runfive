// Package main is the application entry point.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"

	"github.com/runfivedev/runfive/internal/api"
	"github.com/runfivedev/runfive/internal/auth"
	"github.com/runfivedev/runfive/internal/config"
	"github.com/runfivedev/runfive/internal/console"
	"github.com/runfivedev/runfive/internal/database"
	"github.com/runfivedev/runfive/internal/models"
	"github.com/runfivedev/runfive/internal/runtimepath"
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

	if err := os.MkdirAll(runtimepath.Root(), 0o750); err != nil {
		log.Fatalf("Failed to create runtime root %q: %v", runtimepath.Root(), err)
	}

	if err := runtimepath.EnsureReadme(); err != nil {
		log.Printf("Warning: could not write runtime README: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Data root:     %s", runtimepath.Root())
	log.Printf("Servers dir:   %s", cfg.ServersDir)
	log.Printf("Artifacts dir: %s", cfg.ArtifactsDir)

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
		ServersDir:   cfg.ServersDir,
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
