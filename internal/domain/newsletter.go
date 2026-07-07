package domain

import "time"

type NewsletterSend struct {
	ID        string    `db:"id" json:"id"`
	Subject   string    `db:"subject" json:"subject"`
	Body      string    `db:"body" json:"body"`
	Status    string    `db:"status" json:"status"`
	SentCount int       `db:"sent_count" json:"sent_count"`
	FailCount int       `db:"fail_count" json:"fail_count"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

const (
	StatusPending = "pending"
	StatusSending = "sending"
	StatusDone    = "done"
)
