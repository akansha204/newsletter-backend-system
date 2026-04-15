package repository

import "github.com/akansh204/newsletter-backend-system/internal/domain"

type SubscriberRepository interface {
	Create(subscriber *domain.Subscriber) error
	FindByEmail(email string) (*domain.Subscriber, error)
	FindByToken(token string) (*domain.Subscriber, error)
	Confirm(id string) error
	FindAllConfirmed() ([]domain.Subscriber, error)
}

type NewsletterRepository interface {
	Create(n *domain.NewsletterSend) error
	UpdateStatus(id string, status string) error
	IncrementSentCount(id string) error
	IncrementFailCount(id string) error
	FindByID(id string) (*domain.NewsletterSend, error)
}
