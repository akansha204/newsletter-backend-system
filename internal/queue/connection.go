package queue

import (
	"context"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection struct {
	conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewConnection(url string) *Connection {
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open rabbitmq channel: %v", err)
	}

	log.Println("rabbitmq connected successfully")
	return &Connection{conn: conn, Channel: channel}
}

func (c *Connection) HealthCheck(_ context.Context) error {
	if c == nil || c.conn == nil || c.Channel == nil {
		return amqp.ErrClosed
	}
	if c.conn.IsClosed() || c.Channel.IsClosed() {
		return amqp.ErrClosed
	}
	return nil
}

func (c *Connection) Close() {
	if c.Channel != nil {
		c.Channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
