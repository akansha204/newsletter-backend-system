package postgres

import (
	"database/sql"
	"errors"

	"github.com/akansh204/newsletter-backend-system/internal/domain"
	"github.com/jmoiron/sqlx"
)

type newsletterRepo struct {
	db *sqlx.DB
}

func NewNewsletterRepository(db *sqlx.DB) *newsletterRepo {
	return &newsletterRepo{db: db}
}

func (r *newsletterRepo) Create(n *domain.NewsletterSend) error {
	query := `
		INSERT INTO newsletter_sends 
		(id, subject, body, status, sent_count, fail_count, created_at, updated_at)
		VALUES (:id, :subject, :body, :status, :sent_count, :fail_count, :created_at, :updated_at)
	`
	_, err := r.db.NamedExec(query, n)
	return err
}

func (r *newsletterRepo) FindByID(id string) (*domain.NewsletterSend, error) {
	var n domain.NewsletterSend
	err := r.db.Get(&n, "SELECT * FROM newsletter_sends WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &n, err
}

func (r *newsletterRepo) UpdateStatus(id string, status string) error {
	_, err := r.db.Exec(
		"UPDATE newsletter_sends SET status = $1, updated_at = NOW() WHERE id = $2",
		status,
		id,
	)
	return err
}

func (r *newsletterRepo) IncrementSentCount(id string) error {
	_, err := r.db.Exec(
		"UPDATE newsletter_sends SET sent_count = sent_count + 1, updated_at = NOW() WHERE id = $1",
		id,
	)
	return err
}

func (r *newsletterRepo) IncrementFailCount(id string) error {
	_, err := r.db.Exec(
		"UPDATE newsletter_sends SET fail_count = fail_count + 1, updated_at = NOW() WHERE id = $1",
		id,
	)
	return err
}
