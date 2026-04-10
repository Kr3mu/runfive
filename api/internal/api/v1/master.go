package v1

import (
	"encoding/base64"
	"fmt"

	"github.com/Kr3mu/runfive/internal/models"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/client"
	"github.com/joho/godotenv"
)

func verifyDiscordCredentials(clientId, clientSecret string) error {
	cc := client.New()

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

// POST /v1/auth/master/savediscord
func (h *AuthHandler) SaveDiscordAuthentication(c fiber.Ctx) error {
	var req models.DiscordAuthentication

	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := verifyDiscordCredentials(req.ClientId, req.ClientSecret); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	envPath := ".env"

	envMap, _ := godotenv.Read(envPath)

	if envMap == nil {
		envMap = make(map[string]string)
	}

	envMap["DISCORD_CLIENT_ID"] = req.ClientId
	envMap["DISCORD_CLIENT_SECRET"] = req.ClientSecret

	err := godotenv.Write(envMap, envPath)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to write .env")
	}

	return c.SendStatus(fiber.StatusOK)
}

// POST /v1/auth/master/getdiscord
func (h *AuthHandler) GetDiscordAuthentication(c fiber.Ctx) error {
	envPath := ".env"

	envMap, err := godotenv.Read(envPath)
	if err != nil {
		envMap = make(map[string]string)
	}

	return c.JSON(fiber.Map{
		"clientId":     envMap["DISCORD_CLIENT_ID"],
		"clientSecret": envMap["DISCORD_CLIENT_SECRET"],
	})
}
