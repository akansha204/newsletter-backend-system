package email

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewResendProviderValidation(t *testing.T) {
	t.Parallel()

	_, err := NewResendProvider(ResendConfig{
		FromEmail: "sender@example.com",
	})
	if err == nil || !strings.Contains(err.Error(), ErrMissingResendAPIKey.Error()) {
		t.Fatalf("expected missing API key error, got %v", err)
	}

	_, err = NewResendProvider(ResendConfig{
		APIKey: "test-key",
	})
	if err == nil || !strings.Contains(err.Error(), ErrMissingResendFromEmail.Error()) {
		t.Fatalf("expected missing from email error, got %v", err)
	}

	_, err = NewResendProvider(ResendConfig{
		APIKey:    "test-key",
		FromEmail: "sender@example.com",
		BaseURL:   "not-a-url",
	})
	if err == nil || !strings.Contains(err.Error(), ErrInvalidResendBaseURL.Error()) {
		t.Fatalf("expected invalid base URL error, got %v", err)
	}
}

func TestResendProviderSend(t *testing.T) {
	t.Parallel()

	type requestPayload struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
	}

	var (
		gotAuthorization string
		gotAccept        string
		gotContentType   string
		gotPath          string
		gotPayload       requestPayload
		decodeErr        error
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")
		gotPath = r.URL.Path

		decodeErr = json.NewDecoder(r.Body).Decode(&gotPayload)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email_123"}`))
	}))
	defer server.Close()

	provider, err := NewResendProvider(ResendConfig{
		APIKey:     "test-key",
		FromEmail:  "sender@example.com",
		FromName:   "Newsletter Team",
		BaseURL:    server.URL,
		Timeout:    time.Second,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("failed to build provider: %v", err)
	}

	err = provider.Send("reader@example.com", "Weekly Update", "Hello from the worker")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if decodeErr != nil {
		t.Fatalf("failed to decode request body: %v", decodeErr)
	}

	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("unexpected authorization header: %q", gotAuthorization)
	}
	if gotAccept != "application/json" {
		t.Fatalf("unexpected accept header: %q", gotAccept)
	}
	if gotContentType != "application/json" {
		t.Fatalf("unexpected content type: %q", gotContentType)
	}
	if gotPath != "/emails" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
	if gotPayload.From != "\"Newsletter Team\" <sender@example.com>" {
		t.Fatalf("unexpected from field: %q", gotPayload.From)
	}
	if len(gotPayload.To) != 1 || gotPayload.To[0] != "reader@example.com" {
		t.Fatalf("unexpected recipient payload: %+v", gotPayload.To)
	}
	if gotPayload.Subject != "Weekly Update" {
		t.Fatalf("unexpected subject: %q", gotPayload.Subject)
	}
	if gotPayload.Text != "Hello from the worker" {
		t.Fatalf("unexpected text payload: %q", gotPayload.Text)
	}
}

func TestResendProviderSendReturnsHelpfulError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"The from address is not on a verified domain."}`))
	}))
	defer server.Close()

	provider, err := NewResendProvider(ResendConfig{
		APIKey:     "test-key",
		FromEmail:  "sender@example.com",
		BaseURL:    server.URL,
		Timeout:    time.Second,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("failed to build provider: %v", err)
	}

	err = provider.Send("reader@example.com", "Weekly Update", "Hello from the worker")
	if err == nil {
		t.Fatal("expected send error")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected status code in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "verified domain") {
		t.Fatalf("expected resend error message, got %v", err)
	}
}
