package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"

	"github.com/akansh204/newsletter-backend-system/internal/config"
)

func main() {
	cfg := config.Load()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Name,
		cfg.DB.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	defer db.Close()

	total := 500

	now := time.Now()

	fmt.Println(" Seeding subscribers...")

	for i := 0; i < total; i++ {
		email := fmt.Sprintf("seed-%d-%d@test.com", i, time.Now().UnixNano())

		_, err := db.Exec(`
			INSERT INTO subscribers 
			(id, email, confirmed, token, token_expires_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
			uuid.New().String(),
			email,
			true, //explicitly confirmed
			"seed-token",
			now.Add(24*time.Hour),
			now,
			now,
		)

		if err != nil {
			log.Printf("failed insert %s: %v", email, err)
		}
	}

	fmt.Println("Done seeding subscribers")
}
