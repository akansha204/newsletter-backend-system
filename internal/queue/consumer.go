package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
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
	workerWG       sync.WaitGroup
	processingWG   sync.WaitGroup
}

func NewConsumer(conn *Connection, emailProvider email.Provider, newsletterRepo repository.NewsletterRepository) *Consumer {
	return &Consumer{
		channel:        conn.Channel,
		emailProvider:  emailProvider,
		newsletterRepo: newsletterRepo,
	}
}

func (c *Consumer) StartConfirmationWorker(ctx context.Context) error {
	const consumerTag = "confirmation-worker"
	msgs, err := c.channel.Consume(
		QueueConfirmation,
		consumerTag,
		false, // auto-ack — false means we manually ack after processing
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to start confirmation worker: %w", err)
	}

	log.Println("confirmation worker started, waiting for messages...")

	c.workerWG.Add(1)
	go func() {
		defer c.workerWG.Done()

		go func() {
			<-ctx.Done()
			if err := c.channel.Cancel(consumerTag, false); err != nil {
				log.Printf("failed to cancel confirmation consumer: %v", err)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				c.processingWG.Add(1)
				c.processConfirmationMessage(msg)
				c.processingWG.Done()
			}
		}
	}()

	return nil
}

func (c *Consumer) StartNewsletterWorker(ctx context.Context) error {
	const consumerTag = "newsletter-worker"
	msgs, err := c.channel.Consume(
		QueueNewsletter,
		consumerTag,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to start newsletter worker: %w", err)
	}

	log.Println("newsletter worker started, waiting for messages...")

	c.workerWG.Add(1)
	go func() {
		defer c.workerWG.Done()

		go func() {
			<-ctx.Done()
			if err := c.channel.Cancel(consumerTag, false); err != nil {
				log.Printf("failed to cancel newsletter consumer: %v", err)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				c.processingWG.Add(1)
				c.processNewsletterMessage(msg)
				c.processingWG.Done()
			}
		}
	}()

	return nil
}

func (c *Consumer) Wait() {
	c.workerWG.Wait()
	c.processingWG.Wait()
}

func (c *Consumer) processConfirmationMessage(msg amqp.Delivery) {
	var payload ConfirmationPayload
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Printf("failed to parse confirmation message: %v", err)
		msg.Nack(false, false) // failed parse, do not requeue
		return
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
		msg.Nack(false, true) // requeue
		return
	}

	metrics.EmailsSent.Inc()
	msg.Ack(false) // done processing
	log.Printf("confirmation email sent to %s", payload.Email)
}

func (c *Consumer) processNewsletterMessage(msg amqp.Delivery) {
	var payload NewsletterPayload

	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Printf("failed to parse newsletter message: %v", err)
		msg.Nack(false, false)
		return
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
		return
	}

	metrics.EmailsSent.Inc()
	_ = c.newsletterRepo.IncrementSentCount(payload.NewsletterID)

	msg.Ack(false)
	log.Printf("newsletter sent to %s", payload.Email)
}
