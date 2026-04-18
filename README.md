# Newsletter Backend System

A production-style newsletter backend written in Go with a split API/worker architecture, asynchronous email delivery, Redis-backed middleware, Prometheus metrics, Grafana dashboards, and k6 load-testing scripts.

The project currently supports:

- subscriber signup and confirmation
- newsletter dispatch to confirmed subscribers
- asynchronous job delivery through RabbitMQ
- PostgreSQL persistence for subscribers and send history
- Redis-backed rate limiting and idempotency
- Prometheus scraping for API and worker metrics
- Grafana dashboards for runtime and throughput visibility
- SDK clients for Go, TypeScript, and Python consumers

To make the implementation easier to follow, the sections below include small excerpts from the actual codebase so the architecture description maps directly to real files and logic.

## Getting Started

### Prerequisites

- Docker & Docker Compose (for PostgreSQL, Redis, RabbitMQ)
- Go 1.25+
- A Resend account (https://resend.com) with API key

### Email Configuration with Resend

The system uses **Resend** for email delivery. To set up email delivery:

1. **Get your Resend API key**
   - Go to [Resend API Keys](https://resend.com/api-keys)
   - Create or copy an existing API key

2. **Configure `.env`**
   
   ```bash
   # Email Provider
   EMAIL_PROVIDER=resend
   EMAIL_FROM_EMAIL=onboarding@resend.dev          # Pre-verified for testing, or use your verified domain
   EMAIL_FROM_NAME=Newsletter System               # Display name in email clients
   RESEND_API_KEY=re_your_api_key_here             # From Resend dashboard
   RESEND_BASE_URL=https://api.resend.com          # Optional, has sensible default
   RESEND_TIMEOUT=10s                              # Optional, has sensible default
   ```

3. **Important: Free Tier Limitations**
   
   If using Resend's free tier:
   - You can only send test emails to **one verified email address** on your Resend account
   - Subscribe and test with that email during development
   - To send to multiple recipients, [verify a domain](https://resend.com/domains) and update `EMAIL_FROM_EMAIL` to use that domain
   
   Example:
   ```bash
   # For development (free tier) - limited to one test recipient
   EMAIL_FROM_EMAIL=onboarding@resend.dev
   
   # For production (custom domain verified)
   EMAIL_FROM_EMAIL=noreply@yourdomain.com
   ```

4. **Start services and test the flow**
   
   ```bash
   # Start PostgreSQL, Redis, RabbitMQ using make
   make infra-up
   
   # Start API (terminal 1)
   go run ./cmd/api
   
   # Start worker (terminal 2)
   go run ./cmd/worker
   
   # Subscribe (terminal 3)
   curl -X POST http://localhost:3001/api/v1/subscribe \
     -H "Content-Type: application/json" \
     -d '{"email":"your-verified-email@example.com"}'
   
   # Get confirmation token from logs, then confirm
   curl "http://localhost:3001/api/v1/confirm?token=TOKEN_FROM_LOGS"
   
   # Send newsletter
   curl -X POST http://localhost:3001/api/v1/newsletter/send \
     -H "Content-Type: application/json" \
     -H "X-API-Key: YOUR_ADMIN_API_KEY" \
     -H "Idempotency-Key: send-001" \
     -d '{"subject":"Test Newsletter","body":"Hello from your newsletter system!"}'
   ```

## Architecture Overview

The implementation is intentionally split into synchronous and asynchronous paths:

- the API handles validation, persistence, auth, idempotency, and queue publishing
- the worker consumes queue messages and performs email sending work
- Prometheus scrapes metrics from both the API and worker processes
- Grafana visualizes health, goroutines, throughput, latency, and error rates

```mermaid
flowchart LR
    user["Client Applications"] --> sdk["SDK Layer\nGo | TypeScript | Python"]
    sdk --> api["Fiber API\ncmd/api"]
    k6["k6 Load Tests"] --> api

    api --> routes["Routes and Handlers\nsubscribe | confirm | newsletter | health | metrics"]
    routes --> middleware["Middleware\nRate limiting | API key auth | Idempotency"]
    middleware --> repos["Repository Layer"]

    repos --> pg["PostgreSQL\nsubscribers\nnewsletter_sends"]
    middleware --> redis["Redis\nrate-limit counters\nidempotency cache"]

    routes --> publisher["RabbitMQ Publisher"]
    publisher --> rabbit["RabbitMQ\nconfirmation.queue\nnewsletter.queue"]
    rabbit --> worker["Worker Process\ncmd/worker"]
    worker --> email["Email Provider\nResend / provider abstraction"]
    worker --> pg

    api --> apiMetrics["API Metrics\n/api/v1/metrics"]
    worker --> workerMetrics["Worker Metrics\n:3002/metrics"]

    apiMetrics --> prom["Prometheus"]
    workerMetrics --> prom
    prom --> grafana["Grafana Dashboards"]
```

Representative route wiring from `internal/api/router.go`:

```go
api := app.Group("/api/v1")
api.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
api.Get("/health", healthHandler.Check)

if rateLimitCfg.Enabled {
    public := api.Group("", middleware.RateLimiter(redisClient, rateLimitCfg.Limit, rateLimitCfg.Window))
    public.Post("/subscribe", subscribeHandler.Handle)
    public.Get("/confirm", confirmHandler.Handle)
} else {
    api.Post("/subscribe", subscribeHandler.Handle)
    api.Get("/confirm", confirmHandler.Handle)
}

newsletterapi := api.Group("/newsletter", middleware.APIKeyAuth(adminAPIKey))
if idempotencyCfg.Enabled {
    newsletterapi.Post("/send", middleware.Idempotency(redisClient, idempotencyCfg.TTL), newsletterHandler.HandleSend)
} else {
    newsletterapi.Post("/send", newsletterHandler.HandleSend)
}
```

## Request and Processing Flow

### Subscription Flow

1. A client calls `POST /api/v1/subscribe`.
2. The API validates the email and checks whether the subscriber already exists.
3. A confirmation token is generated and stored in PostgreSQL.
4. A confirmation email job is published to RabbitMQ.
5. The worker consumes the job and sends the email.
6. The user clicks `GET /api/v1/confirm?token=...`.
7. The API validates token age and marks the subscriber as confirmed.

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant API
    participant PostgreSQL
    participant RabbitMQ
    participant Worker
    participant Email as Email Provider

    Client->>API: POST /api/v1/subscribe
    API->>PostgreSQL: Check existing email
    API->>PostgreSQL: Insert subscriber + token
    API->>RabbitMQ: Publish confirmation job
    API-->>Client: 201 Created
    RabbitMQ-->>Worker: Deliver confirmation job
    Worker->>Email: Send confirmation email
    Client->>API: GET /api/v1/confirm?token=...
    API->>PostgreSQL: Validate token and confirm subscriber
    API-->>Client: 200 Confirmed
```

Code snapshot from `internal/api/handlers/subscribe.go`:

```go
existing, err := h.repo.FindByEmail(req.Email)
if err != nil {
    return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
        "error": "something went wrong",
    })
}

token, err := generateToken()
now := time.Now()
subscriber := &domain.Subscriber{
    ID:             uuid.New().String(),
    Email:          req.Email,
    Confirmed:      false,
    Token:          token,
    TokenExpiresAt: now.Add(24 * time.Hour),
    CreatedAt:      now,
    UpdatedAt:      now,
}

if err := h.repo.Create(subscriber); err != nil {
    return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
        "error": "failed to save subscriber",
    })
}

_ = h.publisher.PublishConfirmation(queue.ConfirmationPayload{
    Email: subscriber.Email,
    Token: subscriber.Token,
})
```

### Newsletter Dispatch Flow

1. An authenticated admin calls `POST /api/v1/newsletter/send`.
2. The API applies API key auth and optional idempotency protection.
3. A newsletter send record is created in PostgreSQL.
4. All confirmed subscribers are fetched from PostgreSQL.
5. One RabbitMQ job is published per confirmed subscriber.
6. The worker consumes jobs and sends emails.
7. Success and failure counters are written to Prometheus metrics and persisted to PostgreSQL send counters.

```mermaid
sequenceDiagram
    autonumber
    participant Admin
    participant API
    participant Redis
    participant PostgreSQL
    participant RabbitMQ
    participant Worker
    participant Email as Email Provider

    Admin->>API: POST /api/v1/newsletter/send
    API->>Redis: Validate/store idempotency key
    API->>PostgreSQL: Create newsletter_sends record
    API->>PostgreSQL: Load confirmed subscribers
    loop For each confirmed subscriber
        API->>RabbitMQ: Publish newsletter job
    end
    API-->>Admin: 200 Dispatch started
    RabbitMQ-->>Worker: Deliver newsletter jobs
    Worker->>Email: Send email
    Worker->>PostgreSQL: Increment sent/fail counters
```

Code snapshot from `internal/api/handlers/newsletter.go`:

```go
newsletter := &domain.NewsletterSend{
    ID:        uuid.New().String(),
    Subject:   req.Subject,
    Body:      req.Body,
    Status:    domain.StatusPending,
    CreatedAt: now,
    UpdatedAt: now,
}

if err := h.newsletterRepo.Create(newsletter); err != nil {
    return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
        "error": "failed to create newsletter",
    })
}

subscribers, err := h.subscriberRepo.FindAllConfirmed()
if err != nil {
    return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
        "error": "failed to fetch subscribers",
    })
}

for _, sub := range subscribers {
    _ = h.publisher.PublishNewsletter(queue.NewsletterPayload{
        NewsletterID: newsletter.ID,
        Email:        sub.Email,
        Subject:      req.Subject,
        Body:         req.Body,
    })
}
```

## Project Structure

```text
.
├── cmd/
│   ├── api/             # API process entrypoint
│   ├── seed/            # data/bootstrap utilities
│   └── worker/          # worker process entrypoint and worker metrics server
├── deploy/
│   ├── docker-compose.yml
│   ├── grafana/
│   │   └── dashboards/  # checked-in Grafana dashboard screenshots
│   └── prometheus.yml
├── internal/
│   ├── api/
│   │   ├── handlers/    # HTTP handlers
│   │   └── middleware/  # rate limit, auth, idempotency
│   ├── config/          # env-driven configuration
│   ├── database/        # PostgreSQL connection setup
│   ├── domain/          # domain entities
│   ├── email/           # provider abstraction and implementations
│   ├── metrics/         # Prometheus metrics registration
│   ├── queue/           # RabbitMQ connection, publisher, consumer
│   ├── redis/           # Redis client wiring
│   └── repository/      # repository interfaces and PostgreSQL implementations
├── migrations/          # schema creation scripts
├── sdk/
│   ├── go/
│   ├── python/
│   └── typescript/
├── tests/
│   └── k6/              # load tests
├── Makefile
└── README.md
```

## Folder-by-Folder Breakdown

### `cmd/`

- `cmd/api/main.go`
  Starts the Fiber API, loads config, connects PostgreSQL, Redis, and RabbitMQ, registers Prometheus metrics, and exposes the HTTP server.
- `cmd/worker/main.go`
  Starts the worker, exposes the worker metrics endpoint on `WORKER_METRICS_PORT` defaulting to `3002`, connects RabbitMQ/PostgreSQL, and processes confirmation/newsletter jobs.
- `cmd/seed/main.go`
  Inserts demo confirmed subscribers into PostgreSQL for local testing.

### `internal/api/`

- `router.go`
  Registers:
  - `POST /api/v1/subscribe`
  - `GET /api/v1/confirm`
  - `POST /api/v1/newsletter/send`
  - `GET /api/v1/health`
  - `GET /api/v1/metrics`
- `handlers/subscribe.go`
  Creates unconfirmed subscribers and publishes confirmation jobs.
- `handlers/confirm.go`
  Confirms a subscription using a token and expiry validation.
- `handlers/newsletter.go`
  Creates newsletter send records, fetches confirmed subscribers, and publishes jobs.
- `handlers/health.go`
  Returns dependency health for PostgreSQL, RabbitMQ, and Redis.

### `internal/api/middleware/`

- `auth.go`
  Protects newsletter dispatch with an admin API key.
- `ratelimit.go`
  Uses Redis counters and TTL windows to limit public traffic.
- `idempotency.go`
  Uses Redis locks and cached responses to safely replay repeated newsletter-send requests with the same idempotency key.

Representative snippet from `internal/api/middleware/idempotency.go`:

```go
key := c.Get(idempotencyHeader)
fingerprint := requestFingerprint(c)
lockKey := "idempotency:lock:" + key
responseKey := "idempotency:response:" + key

replayed, err := replayCachedResponse(ctx, c, rdb, responseKey, fingerprint)
if replayed {
    return nil
}

acquired, err := rdb.SetNX(ctx, lockKey, fingerprint, ttl).Result()
if !acquired {
    return c.Status(fiber.StatusConflict).JSON(fiber.Map{
        "error": "request with this idempotency key is already in progress",
    })
}
```

### `internal/config/`

- Centralizes runtime configuration from `.env`
- Includes ports, database settings, Redis, RabbitMQ, rate limiting, idempotency, email provider config, admin key, and worker metrics port

Representative snippet from `internal/config/config.go`:

```go
cfg := &Config{
    App: AppConfig{
        Port: getEnvOptional("APP_PORT", "3001"),
        Env:  getEnvOptional("APP_ENV", "development"),
    },
    DB: DBConfig{
        Host:     getEnvRequired("DB_HOST"),
        Port:     getEnvRequired("DB_PORT"),
        User:     getEnvRequired("DB_USER"),
        Password: getEnvRequired("DB_PASSWORD"),
        Name:     getEnvRequired("DB_NAME"),
        SSLMode:  getEnvOptional("DB_SSLMODE", "disable"),
    },
    RabbitMQ: RabbitMQConfig{
        URL: getEnvRequired("RABBITMQ_URL"),
    },
    Email: EmailConfig{
        Provider:      getEnvOptional("EMAIL_PROVIDER", "resend"),
        FromEmail:     getEnvOptional("EMAIL_FROM_EMAIL", ""),
        FromName:      getEnvOptional("EMAIL_FROM_NAME", ""),
        ResendAPIKey:  getEnvOptional("RESEND_API_KEY", ""),
        ResendBaseURL: getEnvOptional("RESEND_BASE_URL", ""),
        ResendTimeout: getEnvOptionalDuration("RESEND_TIMEOUT", 10*time.Second),
    },
}
```

### `internal/database/`

- Creates the SQL connection used by the repository layer

### `internal/domain/`

- `subscriber.go`
  Defines subscriber state including confirmation token and token expiry
- `newsletter.go`
  Defines newsletter send state and counters

### `internal/email/`

- `provider.go`
  Email provider abstraction and provider selection
- `resend.go`
  Production Resend email API implementation used by the worker path
- `ses.go`
  Alternative provider direction for AWS SES

### `internal/metrics/`

- Registers:
  - `emails_sent_total`
  - `emails_failed_total`
  - `email_processing_duration_seconds`
- API and worker each expose their own process-level Prometheus metrics

Representative snippet from `internal/metrics/prometheus.go`:

```go
var (
    EmailsSent = prometheus.NewCounter(
        prometheus.CounterOpts{Name: "emails_sent_total", Help: "Total number of emails sent successfully"},
    )
    EmailsFailed = prometheus.NewCounter(
        prometheus.CounterOpts{Name: "emails_failed_total", Help: "Total number of failed email sends"},
    )
    EmailProcessingDuration = prometheus.NewHistogram(
        prometheus.HistogramOpts{Name: "email_processing_duration_seconds", Help: "Time taken to process email sending"},
    )
)
```

### `internal/queue/`

- `connection.go`
  RabbitMQ connection and health integration
- `publisher.go`
  Queue declaration and job publishing
- `consumer.go`
  Queue declaration, worker consumers, send processing, and metric increments

Representative snippet from `internal/queue/consumer.go`:

```go
start := time.Now()
err := c.emailProvider.Send(payload.Email, payload.Subject, payload.Body)
duration := time.Since(start).Seconds()
metrics.EmailProcessingDuration.Observe(duration)

if err != nil {
    metrics.EmailsFailed.Inc()
    _ = c.newsletterRepo.IncrementFailCount(payload.NewsletterID)
    msg.Nack(false, true)
    return
}

metrics.EmailsSent.Inc()
_ = c.newsletterRepo.IncrementSentCount(payload.NewsletterID)
msg.Ack(false)
```

### `internal/repository/`

- `interfaces.go`
  Repository interfaces used by handlers and workers
- `postgres/`
  PostgreSQL implementations for subscribers and newsletter sends

### `internal/redis/`

- Creates the Redis client used by rate limiting and idempotency middleware

### `migrations/`

- `001_create_subscribers.sql`
  Subscriber table
- `002_create_newsletter_sends.sql`
  Newsletter send tracking table

### `sdk/`

These folders are part of the planned client-consumption layer and are included in the architecture so the project can grow into a reusable platform:

- `sdk/go/client.go`
- `sdk/typescript/client.ts`
- `sdk/typescript/index.ts`
- `sdk/python/client.py`
- `sdk/python/__init__.py`

The SDKs now implement the core API surface for external consumers:

- subscribe a new email
- confirm a subscription token
- trigger newsletter dispatch
- fetch health status
- fetch raw Prometheus metrics

### `tests/k6/`

- `subscribe_load.js`
  Load profile for public subscription traffic
- `newsletter_send.js`
  Load profile for admin-triggered newsletter send traffic including deliberate idempotency collisions

Representative snippet from `tests/k6/subscribe_load.js`:

```javascript
export const options = {
  scenarios: {
    subscribe_load: {
      executor: 'ramping-vus',
      startVUs: 5,
      stages: [
        { duration: '30s', target: 50 },
        { duration: '1m', target: 100 },
        { duration: '30s', target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.5'],
    http_req_duration: ['p(95)<500'],
  },
};
```

## API Surface

### Public Endpoints

- `POST /api/v1/subscribe`
- `GET /api/v1/confirm?token=...`
- `GET /api/v1/health`
- `GET /api/v1/metrics`

### Protected Endpoint

- `POST /api/v1/newsletter/send`

Required headers:

- `X-API-Key: <admin key>`
- `Idempotency-Key: <unique request key>` when idempotency is enabled

## SDK Usage

Each SDK wraps the same set of endpoints:

- `subscribe`
- `confirm`
- `sendNewsletter` / `send_newsletter`
- `health`
- `metrics`

Shared configuration rules:

- nothing in the SDKs is tied to a hardcoded deployment URL
- pass the API base URL explicitly or provide `NEWSLETTER_BASE_URL`
- the SDK accepts either the service origin such as `https://newsletter-api.example.com` or a base URL that already ends with `/api/v1`
- provide `NEWSLETTER_API_KEY` or a per-client/per-request API key only when you need `sendNewsletter`
- `subscribe`, `confirm`, `health`, and `metrics` do not require an admin key
- in the browser, the TypeScript SDK can use the current site origin automatically when the API is served from the same origin

### Go SDK

Import path:

```bash
go get github.com/akansh204/newsletter-backend-system
```

Quick start for public subscription:

```go
package main

import (
    "context"
    "fmt"
    "log"

    newsletter "github.com/akansh204/newsletter-backend-system/sdk/go"
)

func main() {
    client, err := newsletter.NewClient(newsletter.Config{
        BaseURL: "https://newsletter-api.example.com",
    })
    if err != nil {
        log.Fatal(err)
    }

    resp, err := client.Subscribe(context.Background(), "reader@example.com")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Message)
}
```

Environment-based setup:

```bash
export NEWSLETTER_BASE_URL=https://newsletter-api.example.com
export NEWSLETTER_API_KEY=super-secret-admin-key
```

```go
client, err := newsletter.NewClientFromEnv()
if err != nil {
    log.Fatal(err)
}
```

Admin newsletter dispatch:

```go
sendResp, err := client.SendNewsletter(
    context.Background(),
    newsletter.NewsletterSendRequest{
        Subject: "Weekly Update",
        Body:    "Hello from the Go SDK",
    },
    &newsletter.SendNewsletterOptions{
        IdempotencyKey: "dispatch-001",
    },
)
if err != nil {
    log.Fatal(err)
}

fmt.Println(sendResp.Total)
```

Operational endpoints:

```go
health, err := client.Health(context.Background())
metrics, err := client.Metrics(context.Background())
```

### Python SDK

Import path:

If you are using the SDK directly from this repository, add the repo root to `PYTHONPATH` or vendor the `sdk/python` directory into your app, then import from `sdk.python`.

Quick start for public subscription:

```python
from sdk.python import NewsletterClient

client = NewsletterClient(
    base_url="https://newsletter-api.example.com",
)

response = client.subscribe("reader@example.com")
print(response.message)
```

Environment-based setup:

```bash
export NEWSLETTER_BASE_URL=https://newsletter-api.example.com
export NEWSLETTER_API_KEY=super-secret-admin-key
```

```python
from sdk.python import NewsletterClient

client = NewsletterClient.from_env()
```

Admin newsletter dispatch:

```python
send_response = client.send_newsletter(
    "Weekly Update",
    "Hello from the Python SDK",
    idempotency_key="dispatch-001",
)
print(send_response.total)
```

Operational endpoints:

```python
health = client.health()
metrics = client.metrics()
```

Error handling:

```python
from sdk.python import APIError, NewsletterClient

client = NewsletterClient(base_url="https://newsletter-api.example.com")

try:
    client.subscribe("reader@example.com")
except APIError as exc:
    print(exc.status_code, exc.message)
```

### TypeScript SDK

Import path:

If you are consuming the SDK directly from this repository, import from the `sdk/typescript` directory.

Browser quick start for a newsletter signup form:

```ts
import NewsletterClient from "./sdk/typescript";

const client = new NewsletterClient({
  baseUrl: "https://newsletter-api.example.com",
});

const form = document.querySelector<HTMLFormElement>("#newsletter-form");
const emailInput = document.querySelector<HTMLInputElement>("#newsletter-email");

form?.addEventListener("submit", async (event) => {
  event.preventDefault();

  const email = emailInput?.value ?? "";
  const response = await client.subscribe(email);
  console.log(response.message);
});
```

Same-origin browser setup:

If your site and newsletter API are served from the same origin, you can omit `baseUrl` entirely:

```ts
const client = new NewsletterClient();
```

Node or server-side setup with environment variables:

```bash
export NEWSLETTER_BASE_URL=https://newsletter-api.example.com
export NEWSLETTER_API_KEY=super-secret-admin-key
```

```ts
import NewsletterClient from "./sdk/typescript";

const client = NewsletterClient.fromEnv();
```

Admin newsletter dispatch:

```ts
const sendResponse = await client.sendNewsletter(
  {
    subject: "Weekly Update",
    body: "Hello from the TypeScript SDK",
  },
  {
    idempotencyKey: "dispatch-001",
  }
);

console.log(sendResponse.total);
```

Operational endpoints and error handling:

```ts
import NewsletterClient, { APIError } from "./sdk/typescript";

try {
  const health = await client.health();
  const metrics = await client.metrics();
  console.log(health.status, metrics.length);
} catch (error) {
  if (error instanceof APIError) {
    console.error(error.statusCode, error.message);
  }
}
```

## Local Infrastructure

The Docker Compose stack under [deploy/docker-compose.yml](deploy/docker-compose.yml) provides:

- PostgreSQL
- Redis
- RabbitMQ with management UI
- Prometheus
- Grafana

The Go API and worker run outside the container stack in the current setup, while Prometheus scrapes them through `host.docker.internal`.

## Running the Project

### 1. Start infrastructure

```bash
make infra-up
```

### 2. Apply migrations

```bash
make migrate
```

### 3. Start the API

```bash
go run ./cmd/api
```

### 4. Start the worker

```bash
go run ./cmd/worker
```

### 5. Useful URLs

- API health: `http://localhost:3001/api/v1/health`
- API metrics: `http://localhost:3001/api/v1/metrics`
- Worker metrics: `http://localhost:3002/metrics`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`
- RabbitMQ UI: `http://localhost:15672`

## Environment Variables

Important configuration values from `.env`:

```env
APP_PORT=3001
DB_HOST=localhost
DB_PORT=5432
DB_USER=newsletter
DB_PASSWORD=newsletter_pass
DB_NAME=newsletter_db
REDIS_HOST=localhost
REDIS_PORT=6379
RABBITMQ_URL=amqp://newsletter:newsletter_pass@localhost:5672/
RATE_LIMIT_ENABLED=true
RATE_LIMIT_MAX_REQUESTS=5
RATE_LIMIT_WINDOW=1m
IDEMPOTENCY_ENABLED=true
IDEMPOTENCY_TTL=10m
EMAIL_PROVIDER=resend
EMAIL_FROM_EMAIL=no-reply@example.com
EMAIL_FROM_NAME=Newsletter Team
RESEND_API_KEY=<your-resend-api-key>
RESEND_BASE_URL=https://api.resend.com
RESEND_TIMEOUT=10s
ADMIN_API_KEY=<your-admin-key>
WORKER_METRICS_PORT=3002
```

For a working Resend setup, make sure the `EMAIL_FROM_EMAIL` value belongs to a verified domain in your Resend account. For quick local testing, Resend also offers `onboarding@resend.dev` with testing limitations.

## Observability

### Metrics Endpoints

- API metrics: `/api/v1/metrics`
- Worker metrics: `:3002/metrics`

### Prometheus Jobs

- `newsletter-api`
- `newsletter-worker`
- `prometheus`

### Dashboard Signals

The current Grafana dashboard focuses on:

- API scrape health
- worker scrape health
- API goroutine count
- emails processed per second
- email failures per second
- average email processing time
- total emails sent
- total emails failed

### Dashboard Snapshots

The following Grafana screenshots are checked into the repository under `deploy/grafana/dashboards/` so the benchmark and observability sections stay reproducible across machines and pushes.

Health, scrape status, and goroutine behavior:

![Grafana screenshot showing scrape health and API goroutine trends](<deploy/grafana/dashboards/Screenshot from 2026-04-18 00-09-22.png>)

Worker processing rate, failure rate, and average processing latency:

![Grafana screenshot showing emails processed per second, failures per second, and average email processing time](<deploy/grafana/dashboards/Screenshot from 2026-04-18 00-09-40.png>)

Overall throughput and cumulative send/failure totals:

![Grafana screenshot showing email throughput plus total emails sent and failed](<deploy/grafana/dashboards/Screenshot from 2026-04-18 00-09-54.png>)

## Load Testing and Benchmarks

The repository already includes two k6 scenarios. The numbers below combine:

- the exact traffic profile encoded in the k6 scripts
- the observed worker/API metrics from the checked-in Grafana dashboard screenshots above

### k6 Scenario 1: Subscription Load

Source: [tests/k6/subscribe_load.js](tests/k6/subscribe_load.js)

Profile:

- executor: `ramping-vus`
- start VUs: `5`
- stage 1: `30s` ramp to `50` VUs
- stage 2: `1m` ramp to `100` VUs
- stage 3: `30s` ramp down to `0`
- per-iteration sleep: `0.2s`
- threshold: `http_req_failed < 0.5`
- threshold: `p(95) http_req_duration < 500ms`

Approximate intent:

- stress the public subscribe path
- exercise validation, PostgreSQL writes, token generation, and confirmation queue publishing
- validate rate limiting behavior when enabled

Relevant k6 excerpt:

```javascript
const res = http.post(
  `${BASE_URL}/api/v1/subscribe`,
  JSON.stringify({ email }),
  {
    headers: { 'Content-Type': 'application/json' },
  }
);

check(res, {
  'status is 201 or 429': (r) => r.status === 201 || r.status === 429,
});
```

### k6 Scenario 2: Newsletter Dispatch Load

Source: [tests/k6/newsletter_send.js](tests/k6/newsletter_send.js)

Profile:

- executor: `ramping-vus`
- start VUs: `5`
- stage 1: `20s` ramp to `20` VUs
- stage 2: `40s` ramp to `50` VUs
- stage 3: `20s` ramp down to `0`
- per-iteration sleep: `0.5s`
- threshold: `http_req_failed < 0.3`
- threshold: `p(95) http_req_duration < 1000ms`
- every fifth request reuses the same idempotency key to intentionally trigger collisions

Approximate intent:

- stress newsletter dispatch creation
- exercise API key auth and idempotency middleware
- fan out work into RabbitMQ for worker-side processing

Relevant k6 excerpt:

```javascript
const idempotencyKey =
  __ITER % 5 === 0
    ? 'fixed-key-test'
    : `${__VU}-${__ITER}-${Date.now()}`;

const res = http.post(`${BASE_URL}/api/v1/newsletter/send`, payload, {
  headers: {
    'Content-Type': 'application/json',
    'X-API-Key': API_KEY,
    'Idempotency-Key': idempotencyKey,
  },
});
```

### Observed Benchmark Snapshot

Based on the checked-in Grafana dashboard images under `deploy/grafana/dashboards/`, the latest observed run shows:

| Metric | Observed Value |
| --- | --- |
| Worker scrape health | `UP` |
| API scrape health | `UP` |
| API goroutines | baseline around `12-15`, spike near `56` |
| Emails processed/sec | sustained around `500-800 ops/s`, peak near `1.1K ops/s` |
| Emails/sec throughput | sustained around `500-800 ops/s`, peak near `1.1K ops/s` |
| Email failures/sec | `0 ops/s` during the captured run |
| Average email processing time | roughly `30-40 microseconds` in the captured window |
| Total emails failed | `0` |
| Total emails sent | `421229` |

Interpretation:

- the worker path stayed healthy under the observed load
- the send pipeline showed high queue-consumption throughput with no visible send failures in the captured dashboard window
- the API remained healthy while goroutines increased sharply during the busiest period
- the benchmark run appears to validate the current asynchronous design for bursty newsletter traffic

Because the Grafana screenshots represent a captured time window rather than a raw k6 result export, treat the values above as observed benchmark results for that run, not fixed upper bounds for the system.

## Current Design Strengths

- clean split between request acceptance and async email processing
- clear repository abstraction between handlers/workers and persistence
- Redis-backed middleware for operational controls
- queue-backed fan-out that scales better than synchronous delivery
- Prometheus and Grafana already wired into the runtime path
- usable SDKs included for Go, TypeScript, and Python consumers

## Current Limitations and Next Steps

- provider-specific email integrations are still lightweight
- no checked-in Grafana dashboard JSON is currently documented in this repo workflow
- benchmark results are currently documented from dashboard captures rather than exported test reports
- additional integration tests around queue processing and idempotency replay would strengthen confidence
- SDK source is checked in, but packaging and release automation are not set up yet

## Next Enhancements

1. Package and publish the Go, TypeScript, and Python SDKs for easier external adoption.
2. Check in Grafana dashboard JSON under `deploy/grafana/dashboards` for reproducible monitoring.
3. Export raw k6 summaries to JSON/CSV and version benchmark reports under `tests/benchmarks/`.
4. Add dead-letter queues and retry policies for failed email jobs.
5. Expand worker concurrency controls and delivery tracing.

## Notes

- The project uses Prometheus scraping from Docker containers into host-based Go processes through `host.docker.internal`.
- Worker metrics are intentionally exposed separately from API metrics because the worker owns the email processing counters.
- The SDK layer is included in the architecture diagram because it is already present in the repository structure and is a natural next extension of the platform.
