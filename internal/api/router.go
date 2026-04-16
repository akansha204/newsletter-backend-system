package api

import (
	"context"

	"github.com/akansh204/newsletter-backend-system/internal/api/handlers"
	"github.com/akansh204/newsletter-backend-system/internal/api/middleware"
	"github.com/akansh204/newsletter-backend-system/internal/config"
	"github.com/akansh204/newsletter-backend-system/internal/queue"
	"github.com/akansh204/newsletter-backend-system/internal/repository/postgres"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

type dbHealthChecker struct {
	db *sqlx.DB
}

func (d dbHealthChecker) HealthCheck(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

type redisHealthChecker struct {
	client *redis.Client
}

func (r redisHealthChecker) HealthCheck(ctx context.Context) error {
	if r.client == nil {
		return fiber.ErrServiceUnavailable
	}
	return r.client.Ping(ctx).Err()
}

func SetupRoutes(app *fiber.App, db *sqlx.DB, redisClient *redis.Client, queueConn *queue.Connection, publisher *queue.Publisher, adminAPIKey string, rateLimitCfg config.RateLimitConfig) {
	//repositories
	subscriberRepo := postgres.NewSubscriberRepository(db)
	newsletterRepo := postgres.NewNewsletterRepository(db)

	//handlers
	subscribeHandler := handlers.NewSubscribeHandler(subscriberRepo, publisher)
	confirmHandler := handlers.NewConfirmHandler(subscriberRepo)
	newsletterHandler := handlers.NewNewsletterHandler(subscriberRepo, newsletterRepo, publisher)
	healthDependencies := map[string]handlers.HealthDependency{
		"database": dbHealthChecker{db: db},
		"rabbitmq": queueConn,
	}
	if redisClient != nil {
		healthDependencies["redis"] = redisHealthChecker{client: redisClient}
	}
	healthHandler := handlers.NewHealthHandler(healthDependencies)

	//routes
	api := app.Group("/api/v1")
	api.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
	api.Get("/health", healthHandler.Check)

	if rateLimitCfg.Enabled {
		public := api.Group("", middleware.RateLimiter(redisClient, rateLimitCfg.Limit, rateLimitCfg.Window))
		public.Post("/subscribe", subscribeHandler.Handle)
		public.Get("/confirm", confirmHandler.Handle)
	} else {
		api.Post("/subscribe", subscribeHandler.Handle)
		api.Get("/confirm", confirmHandler.Handle)
	}

	newsletterapi := api.Group("/newsletter", middleware.APIKeyAuth(adminAPIKey))
	newsletterapi.Post("/send", newsletterHandler.HandleSend)

}
