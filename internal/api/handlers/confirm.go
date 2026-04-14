package handlers

import (
	"time"

	"github.com/akansh204/newsletter-backend-system/internal/repository"
	"github.com/gofiber/fiber/v2"
)

type ConfirmHandler struct {
	repo repository.SubscriberRepository
}

func NewConfirmHandler(repo repository.SubscriberRepository) *ConfirmHandler {
	return &ConfirmHandler{repo: repo}
}

func (h *ConfirmHandler) Handle(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "token is required",
		})
	}

	subscriber, err := h.repo.FindByToken(token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "something went wrong",
		})
	}

	if subscriber == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "invalid confirmation token",
		})
	}

	if subscriber.Confirmed {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "email already confirmed",
		})
	}

	if time.Now().After(subscriber.TokenExpiresAt) {
		return c.Status(fiber.StatusGone).JSON(fiber.Map{
			"error": "confirmation token has expired, please subscribe again",
		})
	}

	if err := h.repo.Confirm(subscriber.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to confirm subscription",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "subscription confirmed successfully",
	})
}
