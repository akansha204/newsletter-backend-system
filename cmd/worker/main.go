package main

import (
	"context"
	"fmt"
	"log"
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
)

func main() {
	cfg := config.Load()
	metrics.Init()

	fmt.Println("=== Newsletter Worker Starting ===")

	db := database.Connect(&cfg.DB)
	defer db.Close()

	queueConn := queue.NewConnection(cfg.RabbitMQ.URL)
	defer queueConn.Close()

	newsletterRepo := postgres.NewNewsletterRepository(db)
	emailProvider := email.NewSendGridProvider(cfg.Email.SendGridKey)
	consumer := queue.NewConsumer(queueConn, emailProvider, newsletterRepo)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

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
