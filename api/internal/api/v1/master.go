package v1

import (
	"encoding/base64"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"

	"github.com/runfivedev/runfive/internal/models"

	gofiberclient "github.com/gofiber/fiber/v3/client"
)

// envPath is the file where Discord credentials are persisted.
const envPath = ".env"

// verifyDiscordCredentials tests the given client ID and secret against
// the Discord API using a client_credentials grant.
func verifyDiscordCredentials(clientId, clientSecret string) error {
	cc := gofiberclient.New()

	resp, err := cc.Post("https://discord.com/api/oauth2/token", gofiberclient.Config{
		Header: map[string]string{
			"Content-Type":  "application/x-www-form-urlencoded",
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(clientId+":"+clientSecret)),
		},
		FormData: map[string]string{
			"grant_type": "client_credentials",
			"scope":      "identify",
		},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to reach Discord")
	}
	defer resp.Close()

	if resp.StatusCode() == fiber.StatusUnauthorized {
		return fiber.NewError(fiber.StatusBadRequest, "invalid client_id or client_secret")
	}
	if resp.StatusCode() != fiber.StatusOK {
		return fiber.NewError(fiber.StatusBadGateway, "Discord returned unexpected status")
	}

	return nil
}

// DiscordStatus returns whether Discord OAuth2 is configured and working.
// Reads credentials from the .env file and does a live verify against Discord.
// GET /v1/auth/master/discord-status
func (h *AuthHandler) DiscordStatus(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"configured": h.discord.IsConfigured(),
	})
}

// SaveDiscordAuthentication validates, persists, and hot-reloads Discord OAuth credentials.
//
// POST /v1/auth/master/savediscord
func (h *AuthHandler) SaveDiscordAuthentication(c fiber.Ctx) error {
	var req models.DiscordAuthentication

	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if req.ClientId != "" && req.ClientSecret != "" {
		if err := verifyDiscordCredentials(req.ClientId, req.ClientSecret); err != nil {
			return err
		}
	}

	envMap, _ := godotenv.Read(envPath)
	if envMap == nil {
		envMap = make(map[string]string)
	}

	envMap["DISCORD_CLIENT_ID"] = req.ClientId
	envMap["DISCORD_CLIENT_SECRET"] = req.ClientSecret

	if err := godotenv.Write(envMap, envPath); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to write .env")
	}

	// Hot-reload — no restart needed.
	h.discord.Reconfigure(req.ClientId, req.ClientSecret)

	return c.SendStatus(fiber.StatusOK)
}

// GetDiscordAuthentication returns the currently configured Discord OAuth credentials.
//
// GET /v1/auth/master/getdiscord
func (h *AuthHandler) GetDiscordAuthentication(c fiber.Ctx) error {
	envMap, err := godotenv.Read(envPath)
	if err != nil {
		envMap = make(map[string]string)
	}

	return c.JSON(fiber.Map{
		"clientId":     envMap["DISCORD_CLIENT_ID"],
		"clientSecret": envMap["DISCORD_CLIENT_SECRET"],
	})
}
