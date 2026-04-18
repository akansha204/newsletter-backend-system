package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	apiPrefix      = "/api/v1"
	defaultTimeout = 10 * time.Second
	envBaseURL     = "NEWSLETTER_BASE_URL"
	envAPIKey      = "NEWSLETTER_API_KEY"
)

var ErrMissingBaseURL = errors.New("sdk: base URL is required")
var ErrInvalidBaseURL = errors.New("sdk: base URL must be an absolute URL")
var ErrMissingAPIKey = errors.New("sdk: API key is required for newsletter sends")

type Config struct {
	BaseURL       string
	APIKey        string
	HTTPClient    *http.Client
	DefaultHeader http.Header
}

type Client struct {
	baseURL       string
	apiKey        string
	httpClient    *http.Client
	defaultHeader http.Header
}

type SubscribeRequest struct {
	Email string `json:"email"`
}

type NewsletterSendRequest struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type SendNewsletterOptions struct {
	APIKey         string
	IdempotencyKey string
}

type MessageResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type NewsletterSendResponse struct {
	Message string `json:"message"`
	Total   int    `json:"total"`
}

type HealthCheck struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type HealthResponse struct {
	Status string                 `json:"status"`
	Checks map[string]HealthCheck `json:"checks"`
}

type APIError struct {
	StatusCode int
	Message    string
	Body       []byte
}

func (e *APIError) Error() string {
	return fmt.Sprintf("sdk: request failed with status %d: %s", e.StatusCode, e.Message)
}

func NewClient(config Config) (*Client, error) {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	baseURL, err := resolveBaseURL(config.BaseURL)
	if err != nil {
		return nil, err
	}

	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv(envAPIKey))
	}

	return &Client{
		baseURL:       baseURL,
		apiKey:        apiKey,
		httpClient:    httpClient,
		defaultHeader: cloneHeader(config.DefaultHeader),
	}, nil
}

func NewClientFromEnv() (*Client, error) {
	return NewClient(Config{})
}

func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

func (c *Client) Subscribe(ctx context.Context, email string) (*MessageResponse, error) {
	if strings.TrimSpace(email) == "" {
		return nil, errors.New("sdk: email is required")
	}

	var out MessageResponse
	if err := c.doJSON(ctx, http.MethodPost, apiPrefix+"/subscribe", SubscribeRequest{Email: email}, nil, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *Client) Confirm(ctx context.Context, token string) (*MessageResponse, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("sdk: token is required")
	}

	query := url.Values{}
	query.Set("token", token)

	var out MessageResponse
	if err := c.doJSON(ctx, http.MethodGet, apiPrefix+"/confirm?"+query.Encode(), nil, nil, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *Client) SendNewsletter(ctx context.Context, request NewsletterSendRequest, options *SendNewsletterOptions) (*NewsletterSendResponse, error) {
	if strings.TrimSpace(request.Subject) == "" {
		return nil, errors.New("sdk: subject is required")
	}
	if strings.TrimSpace(request.Body) == "" {
		return nil, errors.New("sdk: body is required")
	}

	headers := make(http.Header)
	apiKey := c.apiKey
	if options != nil {
		if options.APIKey != "" {
			apiKey = options.APIKey
		}
		if options.IdempotencyKey != "" {
			headers.Set("Idempotency-Key", options.IdempotencyKey)
		}
	}

	if strings.TrimSpace(apiKey) == "" {
		return nil, ErrMissingAPIKey
	}

	headers.Set("X-API-Key", apiKey)

	var out NewsletterSendResponse
	if err := c.doJSON(ctx, http.MethodPost, apiPrefix+"/newsletter/send", request, headers, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var out HealthResponse
	if err := c.doJSON(ctx, http.MethodGet, apiPrefix+"/health", nil, nil, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *Client) Metrics(ctx context.Context) (string, error) {
	body, err := c.do(ctx, http.MethodGet, apiPrefix+"/metrics", nil, nil)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, headers http.Header, out any) error {
	body, err := c.do(ctx, method, path, payload, headers)
	if err != nil {
		return err
	}

	if out == nil || len(bytes.TrimSpace(body)) == 0 {
		return nil
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("sdk: failed to decode JSON response: %w", err)
	}

	return nil
}

func (c *Client) do(ctx context.Context, method, path string, payload any, headers http.Header) ([]byte, error) {
	requestBody, err := encodeJSON(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, requestBody)
	if err != nil {
		return nil, fmt.Errorf("sdk: failed to build request: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, values := range c.defaultHeader {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	for key, values := range headers {
		req.Header.Del(key)
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sdk: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sdk: failed to read response body: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, newAPIError(resp.StatusCode, body)
	}

	return body, nil
}

func encodeJSON(payload any) (io.Reader, error) {
	if payload == nil {
		return nil, nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("sdk: failed to encode JSON payload: %w", err)
	}

	return bytes.NewReader(body), nil
}

func newAPIError(statusCode int, body []byte) error {
	message := http.StatusText(statusCode)

	var response MessageResponse
	if err := json.Unmarshal(body, &response); err == nil {
		switch {
		case response.Error != "":
			message = response.Error
		case response.Message != "":
			message = response.Message
		}
	} else if text := strings.TrimSpace(string(body)); text != "" {
		message = text
	}

	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Body:       append([]byte(nil), body...),
	}
}

func cloneHeader(header http.Header) http.Header {
	if header == nil {
		return make(http.Header)
	}

	cloned := make(http.Header, len(header))
	for key, values := range header {
		copied := make([]string, len(values))
		copy(copied, values)
		cloned[key] = copied
	}

	return cloned
}

func resolveBaseURL(baseURL string) (string, error) {
	resolved := strings.TrimSpace(baseURL)
	if resolved == "" {
		resolved = strings.TrimSpace(os.Getenv(envBaseURL))
	}
	if resolved == "" {
		return "", ErrMissingBaseURL
	}

	resolved = strings.TrimRight(resolved, "/")
	resolved = strings.TrimSuffix(resolved, apiPrefix)
	resolved = strings.TrimRight(resolved, "/")
	parsed, err := url.Parse(resolved)
	if err != nil {
		return "", fmt.Errorf("sdk: invalid base URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", ErrInvalidBaseURL
	}

	return resolved, nil
}
