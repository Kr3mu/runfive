// Package main is the application entry point.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"

	"github.com/runfivedev/runfive/internal/api"
	"github.com/runfivedev/runfive/internal/artifacts"
	"github.com/runfivedev/runfive/internal/auth"
	"github.com/runfivedev/runfive/internal/config"
	"github.com/runfivedev/runfive/internal/console"
	"github.com/runfivedev/runfive/internal/database"
	"github.com/runfivedev/runfive/internal/launcher"
	"github.com/runfivedev/runfive/internal/models"
	"github.com/runfivedev/runfive/internal/runtimepath"
	"github.com/runfivedev/runfive/internal/serverfs"
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

	lifecycleCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	if err := os.MkdirAll(runtimepath.Root(), 0o750); err != nil {
		log.Printf("Failed to create runtime root %q: %v", runtimepath.Root(), err)
		return
	}

	if err := runtimepath.EnsureReadme(); err != nil {
		log.Printf("Warning: could not write runtime README: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		return
	}

	log.Printf("Data root:     %s", runtimepath.Root())
	log.Printf("Servers dir:   %s", cfg.ServersDir)
	log.Printf("Artifacts dir: %s", cfg.ArtifactsDir)

	if *listenPort != "" {
		cfg.Port = *listenPort
	}

	db, err := database.Connect()
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return
	}

	sm, err := auth.NewSessionManager(db, cfg.SessionEncryptKey)
	if err != nil {
		log.Printf("Failed to create session manager: %v", err)
		return
	}

	fe, err := auth.NewFieldEncryptor(cfg.CfxAPIKeySecret)
	if err != nil {
		log.Printf("Failed to create field encryptor: %v", err)
		return
	}

	cfx := auth.NewCfxAuth(cfg.BaseURL)
	discord := auth.NewDiscordAuth(cfg.BaseURL, cfg.DiscordClientID, cfg.DiscordClientSecret)

	artifactManager, err := artifacts.NewManager(cfg.ArtifactsDir)
	if err != nil {
		log.Printf("Failed to create artifact manager: %v", err)
		return
	}

	serverRegistry, err := serverfs.NewRegistry(cfg.ServersDir, artifactManager, fe)
	if err != nil {
		log.Printf("Failed to create server registry: %v", err)
		return
	}
	serverRegistry.StartWatcher(lifecycleCtx)

	launcherManager := launcher.NewManager(serverRegistry, artifactManager)

	setupTokenStore := auth.NewSetupTokenStore()
	var userCount int64
	if err := db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		log.Printf("Failed to count users: %v", err)
		return
	}

	var setupURL string
	if userCount == 0 {
		token, err := setupTokenStore.Generate()
		if err != nil {
			log.Printf("Failed to generate setup token: %v", err)
			return
		}
		setupURL = fmt.Sprintf("%s/?setup=%s", cfg.BaseURL, token)
	}

	app := api.New(&appConfig, &api.AppDeps{
		DB:              db,
		ArtifactManager: artifactManager,
		ServerRegistry:  serverRegistry,
		Launcher:        launcherManager,
		SM:              sm,
		Cfx:             cfx,
		FE:              fe,
		Discord:         discord,
		ST:              setupTokenStore,
		BaseURL:         cfg.BaseURL,
	})

	if setupURL != "" {
		console.SetupBanner(setupURL)
	} else {
		log.Printf("Serving backend on: %s", cfg.Port)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Listen(":"+cfg.Port, listenConfig)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			log.Printf("HTTP server error: %v", err)
			return
		}
	case <-lifecycleCtx.Done():
		log.Printf("Shutdown signal received, stopping managed servers...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		if err := launcherManager.ShutdownAll(shutdownCtx); err != nil {
			log.Printf("Managed server shutdown error: %v", err)
		}
		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			log.Printf("HTTP shutdown error: %v", err)
		}

		if err := <-errCh; err != nil {
			log.Printf("HTTP server error during shutdown: %v", err)
			return
		}
	}
}
