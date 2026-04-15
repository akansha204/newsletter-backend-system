package middleware

import "github.com/gofiber/fiber/v2"

func APIKeyAuth(apiKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Get("X-API-Key")

		if key == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing API key",
			})
		}

		if key != apiKey {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "invalid API key",
			})
		}

		return c.Next()
	}
}
