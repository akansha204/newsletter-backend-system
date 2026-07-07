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

func (r *newsletterRepo) List(limit, offset int) ([]domain.NewsletterSend, error) {
	var sends []domain.NewsletterSend
	err := r.db.Select(
		&sends,
		"SELECT * FROM newsletter_sends ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		limit,
		offset,
	)
	return sends, err
}

func (r *newsletterRepo) CountAll() (int, error) {
	var count int
	err := r.db.Get(&count, "SELECT COUNT(*) FROM newsletter_sends")
	return count, err
}

func (r *newsletterRepo) SumCounts() (int, int, error) {
	var totals struct {
		Sent   int `db:"sent"`
		Failed int `db:"failed"`
	}
	err := r.db.Get(
		&totals,
		"SELECT COALESCE(SUM(sent_count), 0) AS sent, COALESCE(SUM(fail_count), 0) AS failed FROM newsletter_sends",
	)
	return totals.Sent, totals.Failed, err
}

func (r *newsletterRepo) IncrementFailCount(id string) error {
	_, err := r.db.Exec(
		"UPDATE newsletter_sends SET fail_count = fail_count + 1, updated_at = NOW() WHERE id = $1",
		id,
	)
	return err
}
