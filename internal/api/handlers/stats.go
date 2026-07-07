package handlers

import (
	"log"

	"github.com/akansh204/newsletter-backend-system/internal/repository"
	"github.com/gofiber/fiber/v2"
)

type StatsHandler struct {
	subscriberRepo repository.SubscriberRepository
	newsletterRepo repository.NewsletterRepository
}

func NewStatsHandler(
	subRepo repository.SubscriberRepository,
	newsRepo repository.NewsletterRepository,
) *StatsHandler {
	return &StatsHandler{
		subscriberRepo: subRepo,
		newsletterRepo: newsRepo,
	}
}

func (h *StatsHandler) Handle(c *fiber.Ctx) error {
	subscriberTotal, err := h.subscriberRepo.CountAll()
	if err != nil {
		log.Printf("failed to count subscribers: %v", err)
		return statsError(c)
	}

	subscriberConfirmed, err := h.subscriberRepo.CountConfirmed()
	if err != nil {
		log.Printf("failed to count confirmed subscribers: %v", err)
		return statsError(c)
	}

	newsletterTotal, err := h.newsletterRepo.CountAll()
	if err != nil {
		log.Printf("failed to count newsletter sends: %v", err)
		return statsError(c)
	}

	sentTotal, failTotal, err := h.newsletterRepo.SumCounts()
	if err != nil {
		log.Printf("failed to sum newsletter counters: %v", err)
		return statsError(c)
	}

	lastSends, err := h.newsletterRepo.List(1, 0)
	if err != nil {
		log.Printf("failed to fetch last newsletter send: %v", err)
		return statsError(c)
	}

	newsletters := fiber.Map{
		"total":      newsletterTotal,
		"sent_total": sentTotal,
		"fail_total": failTotal,
	}
	if len(lastSends) > 0 {
		newsletters["last_send"] = lastSends[0]
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"subscribers": fiber.Map{
			"total":     subscriberTotal,
			"confirmed": subscriberConfirmed,
		},
		"newsletters": newsletters,
	})
}

func statsError(c *fiber.Ctx) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "failed to load stats",
	})
}
