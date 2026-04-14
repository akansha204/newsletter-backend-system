package domain

import "time"

type Subscriber struct {
	ID             string    `db:"id"`
	Email          string    `db:"email"`
	Confirmed      bool      `db:"confirmed"`
	Token          string    `db:"token"`
	TokenExpiresAt time.Time `db:"token_expires_at"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}
