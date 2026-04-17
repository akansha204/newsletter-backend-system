package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

const idempotencyHeader = "Idempotency-Key"

type cachedResponse struct {
	Fingerprint string `json:"fingerprint"`
	StatusCode  int    `json:"status_code"`
	Body        []byte `json:"body"`
	ContentType string `json:"content_type"`
}

func Idempotency(rdb *redis.Client, ttl time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if rdb == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "idempotency store unavailable",
			})
		}

		key := c.Get(idempotencyHeader)
		if key == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "missing Idempotency-Key header",
			})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		fingerprint := requestFingerprint(c)
		lockKey := "idempotency:lock:" + key
		responseKey := "idempotency:response:" + key

		replayed, err := replayCachedResponse(ctx, c, rdb, responseKey, fingerprint)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "idempotency lookup failed",
			})
		}
		if replayed {
			return nil
		}

		acquired, err := rdb.SetNX(ctx, lockKey, fingerprint, ttl).Result()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "idempotency lock failed",
			})
		}

		if !acquired {
			replayed, err := replayCachedResponse(ctx, c, rdb, responseKey, fingerprint)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "idempotency lookup failed",
				})
			}
			if replayed {
				return nil
			}

			lockFingerprint, err := rdb.Get(ctx, lockKey).Result()
			if err == nil && lockFingerprint != fingerprint {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": "idempotency key already used for a different request",
				})
			}

			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "request with this idempotency key is already in progress",
			})
		}

		err = c.Next()
		if err != nil {
			_, _ = rdb.Del(ctx, lockKey).Result()
			return err
		}

		statusCode := c.Response().StatusCode()
		if statusCode >= fiber.StatusInternalServerError {
			_, _ = rdb.Del(ctx, lockKey).Result()
			return nil
		}

		record := cachedResponse{
			Fingerprint: fingerprint,
			StatusCode:  statusCode,
			Body:        append([]byte(nil), c.Response().Body()...),
			ContentType: string(c.Response().Header.ContentType()),
		}

		payload, err := json.Marshal(record)
		if err != nil {
			_, _ = rdb.Del(ctx, lockKey).Result()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to store idempotent response",
			})
		}

		if err := rdb.Set(ctx, responseKey, payload, ttl).Err(); err != nil {
			_, _ = rdb.Del(ctx, lockKey).Result()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to store idempotent response",
			})
		}

		_, _ = rdb.Del(ctx, lockKey).Result()
		c.Set("X-Idempotency-Status", "stored")
		return nil
	}
}

func requestFingerprint(c *fiber.Ctx) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", c.Method(), c.Path(), string(c.Body()))))
	return hex.EncodeToString(sum[:])
}

func replayCachedResponse(ctx context.Context, c *fiber.Ctx, rdb *redis.Client, responseKey, fingerprint string) (bool, error) {
	raw, err := rdb.Get(ctx, responseKey).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var cached cachedResponse
	if err := json.Unmarshal(raw, &cached); err != nil {
		return false, err
	}

	if cached.Fingerprint != fingerprint {
		return true, c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "idempotency key already used for a different request",
		})
	}

	if cached.ContentType != "" {
		c.Set(fiber.HeaderContentType, cached.ContentType)
	}
	c.Set("X-Idempotency-Status", "replayed")
	return true, c.Status(cached.StatusCode).Send(cached.Body)
}
