package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/akansh204/newsletter-backend-system/internal/api"
	"github.com/akansh204/newsletter-backend-system/internal/repository/postgres"

	"github.com/akansh204/newsletter-backend-system/internal/config"
	"github.com/akansh204/newsletter-backend-system/internal/database"
	"github.com/akansh204/newsletter-backend-system/internal/email"
	"github.com/akansh204/newsletter-backend-system/internal/metrics"
	"github.com/akansh204/newsletter-backend-system/internal/queue"
)

func main() {
	cfg := config.Load()
	metrics.Init()

	fmt.Println("=== Newsletter System Starting ===")

	db := database.Connect(&cfg.DB)
	defer db.Close()

	queueConn := queue.NewConnection(cfg.RabbitMQ.URL)
	defer queueConn.Close()

	publisher := queue.NewPublisher(queueConn)
	newsletterRepo := postgres.NewNewsletterRepository(db)

	emailProvider := email.NewSendGridProvider(cfg.Email.SendGridKey)

	consumer := queue.NewConsumer(queueConn, emailProvider, newsletterRepo)
	consumer.StartConfirmationWorker()
	consumer.StartNewsletterWorker()

	app := fiber.New(fiber.Config{
		AppName: "Newsletter System v1",
	})

	api.SetupRoutes(app, db, publisher, cfg.Admin.APIKey)
	log.Printf("server starting on port %s", cfg.App.Port)
	if err := app.Listen(":" + cfg.App.Port); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}

}
