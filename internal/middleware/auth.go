package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nexus/gowhats/config"
)

func Protected(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("apikey")
		if apiKey == "" {
			apiKey = c.Query("apikey")
		}

		if apiKey != cfg.GlobalApiKey {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Invalid API Key",
			})
		}

		return c.Next()
	}
}