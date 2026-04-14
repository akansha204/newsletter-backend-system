package postgres

import (
	"database/sql"
	"errors"

	"github.com/akansh204/newsletter-backend-system/internal/domain"
	"github.com/jmoiron/sqlx"
)

type subscriberRepo struct {
	db *sqlx.DB
}

func NewSubscriberRepository(db *sqlx.DB) *subscriberRepo {
	return &subscriberRepo{db: db}
}

func (r *subscriberRepo) Create(s *domain.Subscriber) error {
	query := `
		INSERT INTO subscribers (id, email, confirmed, token, token_expires_at, created_at, updated_at)
		VALUES (:id, :email, :confirmed, :token, :token_expires_at, :created_at, :updated_at)
	`
	_, err := r.db.NamedExec(query, s)
	return err
}

func (r *subscriberRepo) FindByEmail(email string) (*domain.Subscriber, error) {
	var s domain.Subscriber
	err := r.db.Get(&s, "SELECT * FROM subscribers WHERE email = $1", email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &s, err
}

func (r *subscriberRepo) FindByToken(token string) (*domain.Subscriber, error) {
	var s domain.Subscriber
	err := r.db.Get(&s, "SELECT * FROM subscribers WHERE token = $1", token)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &s, err
}

func (r *subscriberRepo) Confirm(id string) error {
	_, err := r.db.Exec(
		"UPDATE subscribers SET confirmed = true, updated_at = NOW() WHERE id = $1",
		id,
	)
	return err
}

func (r *subscriberRepo) FindAllConfirmed() ([]domain.Subscriber, error) {
	var subscribers []domain.Subscriber
	err := r.db.Select(&subscribers, "SELECT * FROM subscribers WHERE confirmed = true")
	return subscribers, err
}
