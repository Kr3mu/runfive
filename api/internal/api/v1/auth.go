package v1

import (
	"log"

	"github.com/Kr3mu/runfive/internal/models"
	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
)

func hash_password(passwd []byte) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(passwd, bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// True means the same
func compare_passwords(hash string, passwd []byte) bool {
	byte_hash := []byte(hash)

	err := bcrypt.CompareHashAndPassword(byte_hash, passwd)

	return err == nil
}

func login(c fiber.Ctx) error {
	request := new(models.LoginRequest)

	if err := c.Bind().Body(request); err != nil {
		log.Println(c.SendStatus(fiber.StatusBadRequest))
		return fiber.NewError(fiber.StatusBadRequest, "Bad Request") // maybe add real message of requirements
	}

	return nil
}

func RegisterRouter(r fiber.Router) {
	auth := r.Group("/auth")

	auth.Post("/login", login)
}
