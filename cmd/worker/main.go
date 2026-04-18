package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akansh204/newsletter-backend-system/internal/config"
	"github.com/akansh204/newsletter-backend-system/internal/database"
	"github.com/akansh204/newsletter-backend-system/internal/email"
	"github.com/akansh204/newsletter-backend-system/internal/metrics"
	"github.com/akansh204/newsletter-backend-system/internal/queue"
	"github.com/akansh204/newsletter-backend-system/internal/repository/postgres"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.Load()
	if err := cfg.ValidateForWorker(); err != nil {
		log.Fatalf("invalid worker configuration: %v", err)
	}
	metrics.Init()

	fmt.Println("=== Newsletter Worker Starting ===")

	db := database.Connect(&cfg.DB)
	defer db.Close()

	queueConn := queue.NewConnection(cfg.RabbitMQ.URL)
	defer queueConn.Close()

	newsletterRepo := postgres.NewNewsletterRepository(db)
	emailProvider, err := email.NewProvider(cfg.Email)
	if err != nil {
		log.Fatalf("failed to initialize email provider: %v", err)
	}
	consumer := queue.NewConsumer(queueConn, emailProvider, newsletterRepo)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	metricsServer := &http.Server{
		Addr:              ":" + cfg.Worker.MetricsPort,
		Handler:           metricsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("worker metrics server listening on port %s", cfg.Worker.MetricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("worker metrics server stopped: %v", err)
		}
	}()

	if err := consumer.StartConfirmationWorker(workerCtx); err != nil {
		log.Fatalf("failed to boot confirmation worker: %v", err)
	}
	if err := consumer.StartNewsletterWorker(workerCtx); err != nil {
		log.Fatalf("failed to boot newsletter worker: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("shutting down workers...")

	workerCancel()
	metricsShutdownCtx, metricsShutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer metricsShutdownCancel()

	if err := metricsServer.Shutdown(metricsShutdownCtx); err != nil {
		log.Printf("worker metrics server shutdown failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		consumer.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("workers drained successfully")
	case <-time.After(5 * time.Second):
		log.Println("worker shutdown timed out; exiting with in-flight work left for retry")
	}
}
