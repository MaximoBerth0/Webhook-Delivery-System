# Webhook Delivery System

A reliable, production-ready webhook delivery service built for distributed systems. Guarantees event delivery with retries, idempotency, HMAC signatures, and full observability even under network failures, slow receivers, or service crashes.


---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
- [Domain Model](#domain-model)
- [API Reference](#api-reference)
- [Event Types](#event-types)
- [Getting Started](#getting-started)
- [Environment Variables](#environment-variables)
- [Running Tests](#running-tests)
- [Deployment](#deployment)

---

## Overview

This system acts as an intermediary between **event producers** (external backends) and **event consumers** (subscribed endpoints). When an event is published, the system automatically finds all registered webhooks for that event type, creates delivery records, and dispatches HTTP requests — handling retries, failures, and observability transparently.

```
External System                      Subscribed Endpoints
(payments, customers, etc.)          (registered by clients)
        │                                       ▲
        │  POST /events                         │
        ▼                                       │
  WEBHOOK DELIVERY SYSTEM                       │
  ├── validates & stores event                  │
  ├── resolves subscribed webhooks ─────────────┤
  ├── creates delivery records        delivers payload
  ├── dispatches worker pool
  └── retries with exponential back-off
```

---

## Features

- **Coordinated Workers** — A dispatcher pool manages concurrent delivery goroutines with controlled concurrency and graceful shutdown.
- **HMAC Signatures** — Every outgoing request is signed using `HMAC-SHA256` so receiving services can verify authenticity.
- **Idempotency** — Duplicate event submissions are detected and deduplicated safely without side effects.
- **Exponential Back-off** — Failed deliveries are retried with exponentially increasing delays up to a configurable `MaxAttempts` limit.
- **Dead Letter Queue** — Deliveries that exhaust all retries are moved to a DLQ instead of blocking the system.
- **Logs & Tracing** — Structured logging and distributed tracing (OpenTelemetry) across all layers.
- **Unit & Integration Tests** — Full test coverage for domain logic and end-to-end HTTP flows.
- **AWS Deployment** — Deployed on AWS (ECS + RDS Postgres) with infrastructure managed via Terraform / CDK.
- **Frontend Client** — Connected to a Vercel-hosted frontend for webhook management and delivery monitoring.

---

## Tech Stack

| Layer          | Technology                          |
|----------------|-------------------------------------|
| Language       | Go 1.25                            |
| Database       | PostgreSQL (AWS RDS)                |
| HTTP Router    | `net/http` / Chi                    |
| Workers        | Native goroutines + waitGruops        |
| Observability  | OpenTelemetry, structured JSON logs |
| Auth / Security| HMAC-SHA256 signatures              |
| Deployment     | AWS ECS (Fargate) + RDS             |
| Frontend       | React on Vercel                     |
| Testing        | `testing` pkg, `testcontainers-go`  |

---

## Architecture

The system is organized into **4 domain modules**, each fully encapsulated with its own entity, service, and repository layer.

```
┌─────────────────────────────────────────────────────────┐
│                        HTTP Layer                        │
│           (handlers decode/encode HTTP ↔ domain)         │
└────────────┬────────────┬────────────┬───────────────────┘
             │            │            │
        ┌────▼───┐  ┌─────▼──┐  ┌─────▼────┐
        │ event  │  │ webhook│  │ delivery │
        │service │  │service │  │ service  │
        └────┬───┘  └─────┬──┘  └─────┬────┘
             │            │            │
        ┌────▼────────────▼────────────▼────┐
        │           Repository Layer         │
        │         (PostgreSQL via RDS)        │
        └────────────────────────────────────┘
                          │
              ┌───────────▼───────────┐
              │     Worker Pool        │
              │  (attempt dispatcher)  │
              │  exponential back-off  │
              └───────────────────────┘
```

### Modules

| Module     | Responsibility                                                                 |
|------------|--------------------------------------------------------------------------------|
| `event`    | Validates and stores incoming events. Triggers the delivery pipeline.          |
| `webhook`  | Manages endpoint registrations, secrets, event subscriptions, and retry config.|
| `delivery` | Represents a single event-to-webhook delivery. Tracks status and attempt count.|
| `attempt`  | Records each HTTP call: response status, body, and timestamp.                  |

### Layer responsibilities

| Layer        | Responsibility                                                           |
|--------------|--------------------------------------------------------------------------|
| `handler`    | Decode HTTP requests, translate to domain types, write HTTP responses.   |
| `service`    | Business logic and orchestration between modules.                        |
| `repository` | Data persistence and database queries.                                   |
| `entity`     | Domain rules, validation, and constructors. No HTTP or JSON knowledge.   |

---

## Domain Model

```
Event (1) ──────────────► Delivery (N)
                               │
Webhook (1) ─────────────►     │
                               │
                          Attempt (N)
```

- One **event** generates one **delivery** per subscribed webhook.
- One **delivery** generates one or more **attempts** depending on `MaxAttempts`.

### Internal event flow

```
POST /events  { type, payload }
        │
        ▼
event.Service.CreateEvent()
        │  validates type, deduplicates, stores event
        ▼
Find all webhooks subscribed to this event type
        │
        ▼  (for each matching webhook)
delivery.Service.Create()
        │  creates delivery record → status: pending
        ▼
Worker Pool picks up delivery
        │  signs payload with HMAC-SHA256
        │  POST → webhook endpoint
        ├── 2xx  ──────────────────► delivery status: success
        └── error → exponential back-off
                        └── exhausted → delivery status: failed → DLQ
```

---

## API Reference

### Events (machine-to-machine)

| Method | Endpoint          | Description                  |
|--------|-------------------|------------------------------|
| `POST` | `/events`         | Publish a new event          |
| `GET`  | `/events/types`   | List all available event types |

**POST /events**
```json
{
  "type": "payment.completed",
  "payload": {
    "payment_id": "pay_123",
    "amount": 4999,
    "currency": "USD"
  }
}
```

---

### Webhooks (administration)

| Method   | Endpoint         | Description                    |
|----------|------------------|--------------------------------|
| `POST`   | `/webhooks`      | Register a new webhook endpoint |
| `GET`    | `/webhooks/:id`  | Get webhook details            |
| `PUT`    | `/webhooks/:id`  | Update webhook configuration   |
| `DELETE` | `/webhooks/:id`  | Remove a webhook               |

**POST /webhooks**
```json
{
  "url": "https://your-service.com/hooks",
  "event_types": ["payment.completed", "payment.failed"],
  "secret": "whsec_your_signing_secret",
  "max_attempts": 5
}
```

---

### Observability

| Method | Endpoint                           | Description                        |
|--------|------------------------------------|------------------------------------|
| `GET`  | `/webhooks/:id/deliveries`         | List all deliveries for a webhook  |
| `GET`  | `/deliveries/:id`                  | Get a specific delivery status     |
| `GET`  | `/deliveries/:id/attempts`         | List all attempts for a delivery   |

---

## Event Types

Events are defined as constants in the `event` package and can be adapted to any backend domain.

| Category       | Event Type                   |
|----------------|------------------------------|
| `customer`     | `customer.created`           |
| `customer`     | `customer.updated`           |
| `customer`     | `customer.deleted`           |
| `payment`      | `payment.created`            |
| `payment`      | `payment.completed`          |
| `payment`      | `payment.failed`             |
| `payment`      | `payment.refunded`           |
| `payment`      | `payment.cancelled`          |
| `subscription` | `subscription.created`       |
| `subscription` | `subscription.renewed`       |
| `subscription` | `subscription.cancelled`     |
| `subscription` | `subscription.past_due`      |

> These event types can be freely extended or replaced to match any external system's domain.

---

## Getting Started

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- PostgreSQL 15+ (or use the provided Docker Compose setup)
- AWS CLI (for deployment only)

### Installation

```bash
# 1. Clone the repository
git clone https://github.com/your-org/webhook-delivery-system.git
cd webhook-delivery-system

# 2. Copy environment variables
cp .env.example .env

# 3. Start dependencies (Postgres)
docker compose up -d

# 4. Run database migrations
go run ./cmd/migrate

# 5. Start the server
go run ./cmd/server
```

The server will be available at `http://localhost:8080`.

---

## Environment Variables

| Variable               | Description                              | Required |
|------------------------|------------------------------------------|----------|
| `DATABASE_URL`         | PostgreSQL connection string             | ✅       |
| `SERVER_PORT`          | HTTP server port (default: `8080`)       | ✅       |
| `WORKER_POOL_SIZE`     | Number of concurrent delivery workers   | ✅       |
| `HMAC_SECRET`          | Default secret for signing (dev only)    | ✅       |
| `OTEL_EXPORTER_ENDPOINT` | OpenTelemetry collector endpoint       | ❌       |
| `LOG_LEVEL`            | Log level: `debug`, `info`, `warn`, `error` | ❌    |
| `MAX_RETRY_ATTEMPTS`   | Global default for max delivery retries  | ❌       |

---

## Running Tests

```bash
# Unit tests only
go test ./...

# Unit + integration tests (requires Docker for testcontainers)
go test ./... -tags=integration

# With coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Integration tests spin up a real PostgreSQL instance via `testcontainers-go` — no manual setup required.

---

## Deployment

The system is deployed on **AWS** using **ECS Fargate** with **RDS PostgreSQL**.

```
┌─────────────────────────────────────────────┐
│                    AWS                       │
│                                             │
│   ┌──────────┐       ┌──────────────────┐  │
│   │   ECS    │──────►│   RDS Postgres   │  │
│   │ Fargate  │       │   (Multi-AZ)     │  │
│   └──────────┘       └──────────────────┘  │
│        ▲                                    │
│   ┌────┴─────┐                             │
│   │   ALB    │◄──── External Systems       │
│   └──────────┘                             │
└─────────────────────────────────────────────┘
         ▲
         │  Vercel Frontend (client dashboard)
```

### Deploy

```bash
# Build Docker image
docker build -t webhook-delivery-system .

# Push to ECR
aws ecr get-login-password | docker login --username AWS --password-stdin <ECR_URI>
docker tag webhook-delivery-system:latest <ECR_URI>:latest
docker push <ECR_URI>:latest

# Deploy to ECS (update service)
aws ecs update-service --cluster webhook-cluster --service webhook-service --force-new-deployment
```

> The Vercel frontend connects to this service via the ALB public endpoint. Configure `NEXT_PUBLIC_API_URL` in your Vercel project settings.
