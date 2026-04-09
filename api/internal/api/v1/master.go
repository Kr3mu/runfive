package v1

import (
	"github.com/Kr3mu/runfive/internal/models"
	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
)

// POST /v1/auth/master/savediscord
func (h *AuthHandler) SaveDiscordAuthentication(c fiber.Ctx) error {
	var req models.DiscordAuthentication

	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
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
		"clientid":     envMap["DISCORD_CLIENT_ID"],
		"clientsecret": envMap["DISCORD_CLIENT_SECRET"],
	})
}
