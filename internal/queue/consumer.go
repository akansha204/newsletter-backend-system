package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/akansh204/newsletter-backend-system/internal/email"
	"github.com/akansh204/newsletter-backend-system/internal/metrics"
	"github.com/akansh204/newsletter-backend-system/internal/repository"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	channel        *amqp.Channel
	emailProvider  email.Provider
	newsletterRepo repository.NewsletterRepository
}

func NewConsumer(conn *Connection, emailProvider email.Provider, newsletterRepo repository.NewsletterRepository) *Consumer {
	return &Consumer{
		channel:        conn.Channel,
		emailProvider:  emailProvider,
		newsletterRepo: newsletterRepo,
	}
}

func (c *Consumer) StartConfirmationWorker() {
	msgs, err := c.channel.Consume(
		QueueConfirmation,
		"",    // consumer tag
		false, // auto-ack — false means we manually ack after processing
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatalf("failed to start confirmation worker: %v", err)
	}

	log.Println("confirmation worker started, waiting for messages...")

	go func() {
		for msg := range msgs {
			var payload ConfirmationPayload
			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				log.Printf("failed to parse confirmation message: %v", err)
				msg.Nack(false, false) //failed parsed, dont requeue
				continue
			}

			subject := "Confirm your newsletter subscription"
			body := fmt.Sprintf(
				"Hi! Please confirm your subscription by clicking this link:\n\n"+
					"http://localhost:3001/api/v1/confirm?token=%s\n\n"+
					"This link expires in 24 hours.",
				payload.Token,
			)

			if err := c.emailProvider.Send(payload.Email, subject, body); err != nil {
				log.Printf("failed to send confirmation email to %s: %v", payload.Email, err)
				metrics.EmailsFailed.Inc()
				msg.Nack(false, true) //requue
				continue
			}

			metrics.EmailsSent.Inc()
			msg.Ack(false) //done processing
			log.Printf("confirmation email sent to %s", payload.Email)
		}
	}()
}

func (c *Consumer) StartNewsletterWorker() {
	msgs, err := c.channel.Consume(
		QueueNewsletter,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to start newsletter worker: %v", err)
	}

	log.Println("newsletter worker started, waiting for messages...")

	go func() {
		for msg := range msgs {
			var payload NewsletterPayload

			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				log.Printf("failed to parse newsletter message: %v", err)
				msg.Nack(false, false)
				continue
			}

			start := time.Now()

			err := c.emailProvider.Send(payload.Email, payload.Subject, payload.Body)

			duration := time.Since(start).Seconds()
			metrics.EmailProcessingDuration.Observe(duration)

			if err != nil {
				log.Printf("failed to send newsletter to %s: %v", payload.Email, err)

				metrics.EmailsFailed.Inc()
				_ = c.newsletterRepo.IncrementFailCount(payload.NewsletterID)

				msg.Nack(false, true)
				continue
			}

			metrics.EmailsSent.Inc()
			_ = c.newsletterRepo.IncrementSentCount(payload.NewsletterID)

			msg.Ack(false)
			log.Printf("newsletter sent to %s", payload.Email)
		}
	}()
}
