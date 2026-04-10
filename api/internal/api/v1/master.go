// master.go provides owner-only ("master") admin endpoints for configuring
// external service integrations like Discord OAuth. These endpoints are
// protected by the RequireMaster middleware which enforces IsOwner.

package v1

import (
	"encoding/base64"
	"fmt"

	"github.com/Kr3mu/runfive/internal/models"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/client"
	"github.com/joho/godotenv"
)

// envPath is the file where Discord credentials are persisted.
const envPath = ".env"

// verifyDiscordCredentials tests the given client ID and secret against
// the Discord API using a client_credentials grant. Returns nil if the
// credentials are valid, or a descriptive error if Discord rejects them.
func verifyDiscordCredentials(clientId, clientSecret string) error {
	cc := client.New()

	// Use Basic auth (base64 of client_id:client_secret) per Discord docs.
	resp, err := cc.Post("https://discord.com/api/oauth2/token", client.Config{
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
		return fmt.Errorf("failed to reach Discord: %w", err)
	}
	defer resp.Close()

	if resp.StatusCode() == fiber.StatusUnauthorized {
		return fmt.Errorf("invalid client_id or client_secret")
	}
	if resp.StatusCode() != fiber.StatusOK {
		return fmt.Errorf("Discord returned status %d", resp.StatusCode())
	}

	return nil
}

// SaveDiscordAuthentication validates and persists Discord OAuth credentials.
// The credentials are verified against the Discord API before being written
// to the .env file so the owner gets immediate feedback on typos.
//
// POST /v1/auth/master/savediscord
func (h *AuthHandler) SaveDiscordAuthentication(c fiber.Ctx) error {
	var req models.DiscordAuthentication

	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// Validate against Discord before persisting — catches wrong credentials early.
	if err := verifyDiscordCredentials(req.ClientId, req.ClientSecret); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Read existing .env to preserve other keys, then merge Discord credentials.
	envMap, _ := godotenv.Read(envPath)
	if envMap == nil {
		envMap = make(map[string]string)
	}

	envMap["DISCORD_CLIENT_ID"] = req.ClientId
	envMap["DISCORD_CLIENT_SECRET"] = req.ClientSecret

	if err := godotenv.Write(envMap, envPath); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to write .env")
	}

	return c.SendStatus(fiber.StatusOK)
}

// GetDiscordAuthentication returns the currently configured Discord OAuth
// credentials so the frontend can pre-fill the form. Called on page load
// of the Master Actions panel.
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
