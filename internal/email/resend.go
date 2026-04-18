package email

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

const (
	resendDefaultBaseURL = "https://api.resend.com"
	resendDefaultTimeout = 10 * time.Second
	resendEmailsPath     = "/emails"
	resendUserAgent      = "newsletter-backend-system/resend"
)

var ErrMissingResendAPIKey = errors.New("resend API key is required")
var ErrMissingResendFromEmail = errors.New("resend from email is required")
var ErrInvalidResendBaseURL = errors.New("resend base URL must be an absolute URL")

type ResendConfig struct {
	APIKey     string
	FromEmail  string
	FromName   string
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type ResendProvider struct {
	apiKey   string
	from     string
	endpoint string
	client   *http.Client
}

type resendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
}

type resendErrorBody struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

func NewResendProvider(cfg ResendConfig) (*ResendProvider, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, ErrMissingResendAPIKey
	}

	fromAddress, err := normalizeEmailAddress(cfg.FromEmail)
	if err != nil {
		if strings.TrimSpace(cfg.FromEmail) == "" {
			return nil, ErrMissingResendFromEmail
		}
		return nil, fmt.Errorf("invalid resend from email: %w", err)
	}

	baseURL, err := normalizeResendBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	client := cfg.HTTPClient
	if client == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = resendDefaultTimeout
		}
		client = &http.Client{Timeout: timeout}
	}

	endpoint, err := url.JoinPath(baseURL, resendEmailsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build resend endpoint: %w", err)
	}

	return &ResendProvider{
		apiKey:   apiKey,
		from:     formatEmailAddress(cfg.FromName, fromAddress),
		endpoint: endpoint,
		client:   client,
	}, nil
}

func (r *ResendProvider) Send(to, subject, body string) error {
	recipient, err := normalizeEmailAddress(to)
	if err != nil {
		return fmt.Errorf("invalid recipient email: %w", err)
	}
	if strings.TrimSpace(subject) == "" {
		return errors.New("email subject is required")
	}
	if strings.TrimSpace(body) == "" {
		return errors.New("email body is required")
	}

	payload := resendEmailRequest{
		From:    r.from,
		To:      []string{recipient},
		Subject: subject,
		Text:    body,
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode resend payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, r.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to build resend request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", resendUserAgent)

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read resend response: %w", err)
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	return fmt.Errorf("resend email send failed: %s", formatResendError(resp.StatusCode, responseBody))
}

func normalizeResendBaseURL(raw string) (string, error) {
	resolved := strings.TrimSpace(raw)
	if resolved == "" {
		resolved = resendDefaultBaseURL
	}

	resolved = strings.TrimRight(resolved, "/")

	parsed, err := url.Parse(resolved)
	if err != nil {
		return "", fmt.Errorf("invalid resend base URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", ErrInvalidResendBaseURL
	}

	return resolved, nil
}

func normalizeEmailAddress(raw string) (string, error) {
	address, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	return address.Address, nil
}

func formatEmailAddress(name, email string) string {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return email
	}

	return (&mail.Address{
		Name:    trimmedName,
		Address: email,
	}).String()
}

func formatResendError(statusCode int, body []byte) string {
	message := fmt.Sprintf("status %d", statusCode)
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return message
	}

	var response resendErrorBody
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Sprintf("%s: %s", message, trimmed)
	}

	if response.Message != "" {
		return fmt.Sprintf("%s: %s", message, response.Message)
	}
	if response.Name != "" {
		return fmt.Sprintf("%s: %s", message, response.Name)
	}

	return fmt.Sprintf("%s: %s", message, trimmed)
}
