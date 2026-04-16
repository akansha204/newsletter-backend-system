package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

func RateLimiter(rdb *redis.Client, limit int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if rdb == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "rate limiter unavailable",
			})
		}

		key := "rate_limit:" + c.IP()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "rate limit error",
			})
		}

		if count == 1 {
			if err := rdb.Expire(ctx, key, window).Err(); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "rate limit error",
				})
			}
		}

		if count > int64(limit) {
			ttl, err := rdb.TTL(ctx, key).Result()
			if err == nil && ttl > 0 {
				c.Set("Retry-After", fmt.Sprintf("%.0f", ttl.Seconds()))
			}

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "too many requests",
			})
		}

		return c.Next()
	}
}
