package api

import (
	"context"

	"github.com/akansh204/newsletter-backend-system/internal/api/handlers"
	"github.com/akansh204/newsletter-backend-system/internal/api/middleware"
	"github.com/akansh204/newsletter-backend-system/internal/queue"
	"github.com/akansh204/newsletter-backend-system/internal/repository/postgres"
	"github.com/gofiber/adaptor/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
)

type dbHealthChecker struct {
	db *sqlx.DB
}

func (d dbHealthChecker) HealthCheck(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

func SetupRoutes(app *fiber.App, db *sqlx.DB, queueConn *queue.Connection, publisher *queue.Publisher, adminAPIKey string) {
	//repositories
	subscriberRepo := postgres.NewSubscriberRepository(db)
	newsletterRepo := postgres.NewNewsletterRepository(db)

	//handlers
	subscribeHandler := handlers.NewSubscribeHandler(subscriberRepo, publisher)
	confirmHandler := handlers.NewConfirmHandler(subscriberRepo)
	newsletterHandler := handlers.NewNewsletterHandler(subscriberRepo, newsletterRepo, publisher)
	healthHandler := handlers.NewHealthHandler(map[string]handlers.HealthDependency{
		"database": dbHealthChecker{db: db},
		"rabbitmq": queueConn,
	})

	//routes
	api := app.Group("/api/v1")
	api.Post("/subscribe", subscribeHandler.Handle)
	api.Get("/confirm", confirmHandler.Handle)
	api.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
	api.Get("/health", healthHandler.Check)

	newsletterapi := api.Group("/newsletter", middleware.APIKeyAuth(adminAPIKey))
	newsletterapi.Post("/send", newsletterHandler.HandleSend)

}
