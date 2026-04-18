package email

import (
	"fmt"
	"strings"

	"github.com/akansh204/newsletter-backend-system/internal/config"
)

type Provider interface {
	Send(to, subject, body string) error
}

type Message struct {
	To      string
	Subject string
	Body    string
}

func NewProvider(cfg config.EmailConfig) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "resend":
		return NewResendProvider(ResendConfig{
			APIKey:    cfg.ResendAPIKey,
			FromEmail: cfg.FromEmail,
			FromName:  cfg.FromName,
			BaseURL:   cfg.ResendBaseURL,
			Timeout:   cfg.ResendTimeout,
		})
	case "ses":
		return nil, fmt.Errorf("email provider ses is not implemented")
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", cfg.Provider)
	}
}
