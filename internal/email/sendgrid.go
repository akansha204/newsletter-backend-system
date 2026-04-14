package email

import (
	"fmt"
	"log"
)

type SendGridProvider struct {
	apiKey string
}

func NewSendGridProvider(apiKey string) *SendGridProvider {
	return &SendGridProvider{apiKey: apiKey}
}

func (s *SendGridProvider) Send(to, subject, body string) error {
	log.Printf("[EMAIL] to=%s subject=%s body=%s", to, subject, body)
	fmt.Printf("\n Sending email to: %s\n   Subject: %s\n   Body: %s\n\n", to, subject, body)
	return nil
}
