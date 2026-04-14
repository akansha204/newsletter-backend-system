package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"github.com/akansh204/newsletter-backend-system/internal/domain"
	"github.com/akansh204/newsletter-backend-system/internal/queue"
	"github.com/akansh204/newsletter-backend-system/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type SubscribeHandler struct {
	repo      repository.SubscriberRepository
	publisher *queue.Publisher
}

func NewSubscribeHandler(repo repository.SubscriberRepository, publisher *queue.Publisher) *SubscribeHandler {
	return &SubscribeHandler{repo: repo, publisher: publisher}
}

type subscribeRequest struct {
	Email string `json:"email"`
}

func (h *SubscribeHandler) Handle(c *fiber.Ctx) error {
	var req subscribeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "email is required",
		})
	}

	existing, err := h.repo.FindByEmail(req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "something went wrong",
		})
	}

	if existing != nil && existing.Confirmed {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "email already subscribed",
		})
	}

	token, err := generateToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "something went wrong",
		})
	}

	now := time.Now()
	subscriber := &domain.Subscriber{
		ID:             uuid.New().String(),
		Email:          req.Email,
		Confirmed:      false,
		Token:          token,
		TokenExpiresAt: now.Add(24 * time.Hour),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := h.repo.Create(subscriber); err != nil {
		log.Printf("failed to create subscriber: %v", err)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save subscriber",
		})
	}

	if err := h.publisher.PublishConfirmation(queue.ConfirmationPayload{
		Email: subscriber.Email,
		Token: subscriber.Token,
	}); err != nil {
		log.Printf("failed to publish confirmation email job: %v", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "please check your email to confirm your subscription",
	})
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
