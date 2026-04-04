# Webhook Delivery System

A reliable webhook delivery service designed for distributed systems that must deliver events to external endpoints safely and consistently.

The system guarantees reliable delivery, retries, idempotency, and observability, even when networks fail, receivers are slow, or services crash. It is built to handle high concurrency and failure scenarios common in payment platforms, SaaS integrations, and event-driven architectures.

---

# Design Principles 

* Reliability First - The system assumes that failures are normal and designs around them.

* Deterministic Retries - Event payloads are immutable after creation to ensure consistent retry behavior.

* Idempotent Safety - Duplicate deliveries are expected and handled safely.

* Failure Isolation - Unrecoverable events are moved to a dead letter queue rather than blocking the system.

* Security by Default - Webhook requests include HMAC signatures for verification by receiving services.

* Concurrency with Consistency - Workers operate in parallel while preserving strict delivery invariants.

---

---

## What this system does

This system acts as an intermediary between event producers (external backends) and event consumers (subscribed endpoints).

```
External System                      Subscribed Endpoints
(payments, customers, etc)           (registered by clients)
        │                                      ▲
        │  POST /events                        │
        ▼                                      │
  THIS SYSTEM                                  │
  stores the event                             │
  finds subscribed webhooks  ─────────────────►│
  creates deliveries                  delivers payload
  retries on failure
```

---

## Architecture

The system is divided into 4 domain modules, each with its own entity, service, and repository:

### `event`
Receives events from external systems. Validates the event type against a known list of constants and stores it. Once created, it triggers the delivery process.

### `webhook`
Manages endpoint registrations. External clients register their URLs here, specifying which event types they want to receive, a secret for signature verification, and a maximum number of retry attempts.

### `delivery`
Represents a single attempt to deliver an event to a webhook. Created automatically when an event arrives and a matching subscribed webhook is found. Tracks status (`pending`, `success`, `failed`) and attempt count.

### `attempt`
Represents each individual HTTP call made to a webhook endpoint. Stores the response status, response body, and timestamp. A delivery can have multiple attempts depending on `MaxAttempts`.

---

## Domain relationships

```
Event (1) ──────────────► Delivery (N)
                               │
Webhook (1) ─────────────►     │
                               │
                          Attempt (N)
```

One event can generate multiple deliveries (one per subscribed webhook).  
One delivery can generate multiple attempts (retries on failure).

---

## Event types

Events are defined as constants in the `event` package. Clients can query available types via the API.

| Category     | Event                       |
|--------------|-----------------------------|
| customer     | customer.created            |
| customer     | customer.updated            |
| customer     | customer.deleted            |
| payment      | payment.created             |
| payment      | payment.completed           |
| payment      | payment.failed              |
| payment      | payment.refunded            |
| payment      | payment.cancelled           |
| subscription | subscription.created        |
| subscription | subscription.renewed        |
| subscription | subscription.cancelled      |
| subscription | subscription.past_due       |

**These events can be modified and adapted to any external backend**

---

## API

### For external systems (machine to machine)
```
POST  /events                    Publish a new event
GET   /events/types              List all available event types
```

### For webhook administration
```
POST   /webhooks                 Register a new webhook endpoint
GET    /webhooks/:id             Get webhook details
PUT    /webhooks/:id             Update webhook configuration
DELETE /webhooks/:id             Remove a webhook
```

### For observability
```
GET  /webhooks/:id/deliveries    List all deliveries for a webhook
GET  /deliveries/:id             Get a specific delivery status
GET  /deliveries/:id/attempts    List all attempts for a delivery
```

---

## Internal event flow

```
POST /events  { type, payload }
        │
        ▼
event.Service.CreateEvent()
        │  validates type, stores event
        ▼
Find all webhooks subscribed to this event type
        │
        ▼  for each webhook
delivery.Service.Create()
        │  creates delivery with status: pending
        ▼
attempt.Service  (worker / dispatcher)
        │  makes HTTP POST to webhook endpoint
        ├── 2xx  →  delivery status: success
        └── error → retry up to MaxAttempts
                      └── all failed → delivery status: failed
```

---

## Responsibilities per layer

| Layer      | Responsibility                                      |
|------------|-----------------------------------------------------|
| Handler    | Decode HTTP request, translate to domain types, write HTTP response |
| Service    | Business logic, orchestration between modules       |
| Repository | Data persistence, database queries                  |
| Entity     | Domain rules, validation, constructors              |

---

## Key design decisions

**Events are published by machines, not humans.**  
`POST /events` is called by an external backend, not a user interface.

**Deliveries and attempts are created internally.**  
No external caller creates a delivery or attempt directly. They are a result of an event arriving.

**The domain has no knowledge of HTTP or JSON.**  
All encoding/decoding happens in the handler layer. Services and entities only deal with domain types.

**Observability is the only human-facing concern.**  
Humans interact with the system only to register webhooks and monitor delivery status.