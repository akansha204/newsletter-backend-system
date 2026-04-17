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
- starter SDK placeholders for Go, TypeScript, and Python clients

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
    worker --> email["Email Provider\ncurrent stub / provider abstraction"]
    worker --> pg

    api --> apiMetrics["API Metrics\n/api/v1/metrics"]
    worker --> workerMetrics["Worker Metrics\n:3002/metrics"]

    apiMetrics --> prom["Prometheus"]
    workerMetrics --> prom
    prom --> grafana["Grafana Dashboards"]
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

### Newsletter Dispatch Flow

1. An authenticated admin calls `POST /api/v1/newsletter/send`.
2. The API applies API key auth and optional idempotency protection.
3. A newsletter send record is created in PostgreSQL.
4. All confirmed subscribers are fetched from PostgreSQL.
5. One RabbitMQ job is published per confirmed subscriber.
6. The worker consumes jobs and sends emails.
7. Success and failure counters are written to Prometheus metrics and persisted to PostgreSQL send counters.

## Project Structure

```text
.
├── cmd/
│   ├── api/             # API process entrypoint
│   ├── seed/            # data/bootstrap utilities
│   └── worker/          # worker process entrypoint and worker metrics server
├── deploy/
│   ├── docker-compose.yml
│   ├── prometheus.yml
│   └── grafana/         # Grafana provisioning/dashboard area
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
│   ├── integration/
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
  Reserved for bootstrap or seed workflows.

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

### `internal/config/`

- Centralizes runtime configuration from `.env`
- Includes ports, database settings, Redis, RabbitMQ, rate limiting, idempotency, email provider config, admin key, and worker metrics port

### `internal/database/`

- Creates the SQL connection used by the repository layer

### `internal/domain/`

- `subscriber.go`
  Defines subscriber state including confirmation token and token expiry
- `newsletter.go`
  Defines newsletter send state and counters

### `internal/email/`

- `provider.go`
  Email provider abstraction
- `sendgrid.go`
  Current stubbed provider implementation used by the worker path
- `ses.go`
  Alternative provider direction for AWS SES

### `internal/metrics/`

- Registers:
  - `emails_sent_total`
  - `emails_failed_total`
  - `email_processing_duration_seconds`
- API and worker each expose their own process-level Prometheus metrics

### `internal/queue/`

- `connection.go`
  RabbitMQ connection and health integration
- `publisher.go`
  Queue declaration and job publishing
- `consumer.go`
  Queue declaration, worker consumers, send processing, and metric increments

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
- `sdk/python/client.py`

At the moment they are placeholders, but they define the intended extension point for external consumers of the API.

### `tests/k6/`

- `subscribe_load.js`
  Load profile for public subscription traffic
- `newsletter_send.js`
  Load profile for admin-triggered newsletter send traffic including deliberate idempotency collisions

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

## Local Infrastructure

The Docker Compose stack under [deploy/docker-compose.yml](/home/akansha-roy/newsletter-system/deploy/docker-compose.yml) provides:

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
RATE_LIMIT_ENABLED=false
RATE_LIMIT_MAX_REQUESTS=10000
RATE_LIMIT_WINDOW=1m
IDEMPOTENCY_ENABLED=true
IDEMPOTENCY_TTL=10m
ADMIN_API_KEY=<your-admin-key>
WORKER_METRICS_PORT=3002
```

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

## Load Testing and Benchmarks

The repository already includes two k6 scenarios. The numbers below combine:

- the exact traffic profile encoded in the k6 scripts
- the observed worker/API metrics from the attached Grafana snapshots

### k6 Scenario 1: Subscription Load

Source: [tests/k6/subscribe_load.js](/home/akansha-roy/newsletter-system/tests/k6/subscribe_load.js)

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

### k6 Scenario 2: Newsletter Dispatch Load

Source: [tests/k6/newsletter_send.js](/home/akansha-roy/newsletter-system/tests/k6/newsletter_send.js)

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

### Observed Benchmark Snapshot

Based on the attached Grafana dashboard images, the latest observed run shows:

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
- SDK directories already reserved for future external integration

## Current Limitations and Next Steps

- SDK implementations are placeholders and not yet feature-complete
- provider-specific email integrations are still lightweight
- no checked-in Grafana dashboard JSON is currently documented in this repo workflow
- benchmark results are currently documented from dashboard captures rather than exported test reports
- additional integration tests around queue processing and idempotency replay would strengthen confidence

## Suggested Next Enhancements

1. Add fully implemented Go, TypeScript, and Python SDKs for subscription and newsletter operations.
2. Check in Grafana dashboard JSON under `deploy/grafana/dashboards` for reproducible monitoring.
3. Export raw k6 summaries to JSON/CSV and version benchmark reports under `tests/benchmarks/`.
4. Add dead-letter queues and retry policies for failed email jobs.
5. Expand worker concurrency controls and delivery tracing.

## Notes

- The project uses Prometheus scraping from Docker containers into host-based Go processes through `host.docker.internal`.
- Worker metrics are intentionally exposed separately from API metrics because the worker owns the email processing counters.
- The SDK layer is included in the architecture diagram because it is already present in the repository structure and is a natural next extension of the platform.
