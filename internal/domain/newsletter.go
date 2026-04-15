package domain

import "time"

type NewsletterSend struct {
	ID        string    `db:"id"`
	Subject   string    `db:"subject"`
	Body      string    `db:"body"`
	Status    string    `db:"status"`
	SentCount int       `db:"sent_count"`
	FailCount int       `db:"fail_count"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

const (
	StatusPending = "pending"
	StatusSending = "sending"
	StatusDone    = "done"
)
