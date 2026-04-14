package api

import (
	"github.com/akansh204/newsletter-backend-system/internal/api/handlers"
	"github.com/akansh204/newsletter-backend-system/internal/repository/postgres"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
)

func SetupRoutes(app *fiber.App, db *sqlx.DB) {
	//repositories
	subscriberRepo := postgres.NewSubscriberRepository(db)

	//handlers
	subscribeHandler := handlers.NewSubscribeHandler(subscriberRepo)
	confirmHandler := handlers.NewConfirmHandler(subscriberRepo)

	//routes
	api := app.Group("/api/v1")
	api.Post("/subscribe", subscribeHandler.Handle)
	api.Get("/confirm", confirmHandler.Handle)

}
