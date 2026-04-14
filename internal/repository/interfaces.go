package repository

import "github.com/akansh204/newsletter-backend-system/internal/domain"

type SubscriberRepository interface {
	Create(subscriber *domain.Subscriber) error
	FindByEmail(email string) (*domain.Subscriber, error)
	FindByToken(token string) (*domain.Subscriber, error)
	Confirm(id string) error
	FindAllConfirmed() ([]domain.Subscriber, error)
}
