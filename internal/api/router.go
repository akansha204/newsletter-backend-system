package api

import (
	"github.com/akansh204/newsletter-backend-system/internal/api/handlers"
	"github.com/akansh204/newsletter-backend-system/internal/api/middleware"
	"github.com/akansh204/newsletter-backend-system/internal/queue"
	"github.com/akansh204/newsletter-backend-system/internal/repository/postgres"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
)

func SetupRoutes(app *fiber.App, db *sqlx.DB, publisher *queue.Publisher, adminAPIKey string) {
	//repositories
	subscriberRepo := postgres.NewSubscriberRepository(db)
	newsletterRepo := postgres.NewNewsletterRepository(db)

	//handlers
	subscribeHandler := handlers.NewSubscribeHandler(subscriberRepo, publisher)
	confirmHandler := handlers.NewConfirmHandler(subscriberRepo)
	newsletterHandler := handlers.NewNewsletterHandler(subscriberRepo, newsletterRepo, publisher)

	//routes
	api := app.Group("/api/v1")
	api.Post("/subscribe", subscribeHandler.Handle)
	api.Get("/confirm", confirmHandler.Handle)

	newsletterapi := api.Group("/newsletter", middleware.APIKeyAuth(adminAPIKey))
	newsletterapi.Post("/send", newsletterHandler.HandleSend)

}
