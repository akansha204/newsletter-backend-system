package handlers

import (
	"log"
	"time"

	"github.com/akansh204/newsletter-backend-system/internal/domain"
	"github.com/akansh204/newsletter-backend-system/internal/queue"
	"github.com/akansh204/newsletter-backend-system/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type NewsletterHandler struct {
	subscriberRepo repository.SubscriberRepository
	newsletterRepo repository.NewsletterRepository
	publisher      *queue.Publisher
}

func NewNewsletterHandler(
	subRepo repository.SubscriberRepository,
	newsRepo repository.NewsletterRepository,
	publisher *queue.Publisher,
) *NewsletterHandler {
	return &NewsletterHandler{
		subscriberRepo: subRepo,
		newsletterRepo: newsRepo,
		publisher:      publisher,
	}
}

type newsletterRequest struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (h *NewsletterHandler) HandleSend(c *fiber.Ctx) error {
	var req newsletterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	if req.Subject == "" || req.Body == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "subject and body required",
		})
	}

	now := time.Now()
	newsletter := &domain.NewsletterSend{
		ID:        uuid.New().String(),
		Subject:   req.Subject,
		Body:      req.Body,
		Status:    domain.StatusPending,
		SentCount: 0,
		FailCount: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.newsletterRepo.Create(newsletter); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create newsletter",
		})
	}

	subscribers, err := h.subscriberRepo.FindAllConfirmed()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch subscribers",
		})
	}

	if err := h.newsletterRepo.UpdateStatus(newsletter.ID, domain.StatusSending); err != nil {
		log.Printf("failed to update newsletter status: %v", err)
	}

	for _, sub := range subscribers {
		err := h.publisher.PublishNewsletter(queue.NewsletterPayload{
			NewsletterID: newsletter.ID,
			Email:        sub.Email,
			Subject:      req.Subject,
			Body:         req.Body,
		})
		if err != nil {
			log.Printf("failed to publish newsletter job for %s: %v", sub.Email, err)
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "newsletter dispatch started",
		"total":   len(subscribers),
		"id":      newsletter.ID,
	})
}

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

func (h *NewsletterHandler) HandleList(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", defaultListLimit)
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	offset := c.QueryInt("offset", 0)
	if offset < 0 {
		offset = 0
	}

	sends, err := h.newsletterRepo.List(limit, offset)
	if err != nil {
		log.Printf("failed to list newsletter sends: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list newsletter sends",
		})
	}
	if sends == nil {
		sends = []domain.NewsletterSend{}
	}

	total, err := h.newsletterRepo.CountAll()
	if err != nil {
		log.Printf("failed to count newsletter sends: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list newsletter sends",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"items":  sends,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *NewsletterHandler) HandleGet(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "newsletter send not found",
		})
	}

	send, err := h.newsletterRepo.FindByID(id)
	if err != nil {
		log.Printf("failed to fetch newsletter send %s: %v", id, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch newsletter send",
		})
	}

	if send == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "newsletter send not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(send)
}
