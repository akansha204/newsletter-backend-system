package queue

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	QueueConfirmation = "confirmation.queue"
	QueueNewsletter   = "newsletter.queue"
)

type ConfirmationPayload struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type NewsletterPayload struct {
	Email   string `json:"email"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type Publisher struct {
	channel *amqp.Channel
}

func NewPublisher(conn *Connection) *Publisher {
	_, err := conn.Channel.QueueDeclare(
		QueueConfirmation,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("failed to declare confirmation queue: %v", err)
	}

	_, err = conn.Channel.QueueDeclare(
		QueueNewsletter,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to declare newsletter queue: %v", err)
	}

	log.Println("queues declared successfully")
	return &Publisher{channel: conn.Channel}
}

func (p *Publisher) PublishConfirmation(payload ConfirmationPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.channel.Publish(
		"",                // exchange — empty means default
		QueueConfirmation, // routing key — which queue to send to
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // message survives rabbitmq restart
		},
	)
}

func (p *Publisher) PublishNewsletter(payload NewsletterPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.channel.Publish(
		"",
		QueueNewsletter,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}
