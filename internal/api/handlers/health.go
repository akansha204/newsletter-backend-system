package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

type HealthDependency interface {
	HealthCheck(ctx context.Context) error
}

type HealthHandler struct {
	dependencies map[string]HealthDependency
}

func NewHealthHandler(dependencies map[string]HealthDependency) *HealthHandler {
	return &HealthHandler{dependencies: dependencies}
}

func (h *HealthHandler) Check(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	checks := make(fiber.Map, len(h.dependencies))
	healthy := true

	for name, dependency := range h.dependencies {
		if err := dependency.HealthCheck(ctx); err != nil {
			checks[name] = fiber.Map{
				"status": "down",
				"error":  err.Error(),
			}
			healthy = false
			continue
		}

		checks[name] = fiber.Map{
			"status": "up",
		}
	}

	statusCode := fiber.StatusOK
	status := "ok"
	if !healthy {
		statusCode = fiber.StatusServiceUnavailable
		status = "degraded"
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"status": status,
		"checks": checks,
	})
}
