package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/akansh204/newsletter-backend-system/internal/api"

	"github.com/akansh204/newsletter-backend-system/internal/config"
	"github.com/akansh204/newsletter-backend-system/internal/database"
	"github.com/akansh204/newsletter-backend-system/internal/metrics"
	"github.com/akansh204/newsletter-backend-system/internal/queue"
	redisclient "github.com/akansh204/newsletter-backend-system/internal/redis"
)

func main() {
	cfg := config.Load()
	if err := cfg.ValidateForAPI(); err != nil {
		log.Fatalf("invalid API configuration: %v", err)
	}
	metrics.Init()

	fmt.Println("=== Newsletter System Starting ===")

	db := database.Connect(&cfg.DB)
	defer db.Close()

	queueConn := queue.NewConnection(cfg.RabbitMQ.URL)
	defer queueConn.Close()
	rdb := redisclient.NewRedisClient(cfg.Redis.Host + ":" + cfg.Redis.Port)
	defer rdb.Close()

	publisher := queue.NewPublisher(queueConn)

	app := fiber.New(fiber.Config{
		AppName: "Newsletter System v1",
	})

	api.SetupRoutes(app, db, rdb, queueConn, publisher, cfg.Admin.APIKey, cfg.RateLimit, cfg.Idempotency)
	log.Printf("server starting on port %s", cfg.App.Port)
	go func() {
		if err := app.Listen(":" + cfg.App.Port); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("server shutdown failed: %v", err)
	}

	log.Println("server exited properly")

}
