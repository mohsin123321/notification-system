# Notification System

A multi-channel notification platform built with Go microservices. Any service sends *"notify user X about event Y"* and the platform decides whether, where, and how to deliver — reliably and exactly once from the user's perspective.

Built as a deep-dive into Go microservices patterns: gRPC for synchronous calls, RabbitMQ (AMQP) for async reliable delivery, and MQTT for real-time device push.

## What it does

- Sends notifications to users via **email**, **SMS**, and **in-app** (MQTT) channels
- Respects per-user channel preferences and **quiet hours** (critical types like password reset bypass them)
- Tracks delivery status per channel with **streaming updates** over gRPC
- **Idempotent**: repeat requests return the same notification — nothing is sent twice
- **Outbox pattern**: a broker outage cannot lose messages
- **Priority routing**: transactional notifications get dedicated queues, marketing blasts can't starve password resets

## Architecture

```
Client ──HTTP──► api-gateway ──gRPC──► notification-service ──gRPC──► users-service
                                              │
                                         Postgres
                                    (notifications, deliveries, outbox)
                                              │
                                        outbox worker
                                              │ AMQP
                                    exchange "notifications"
                               ┌──────────┬──────────┐
                           q.email     q.sms      q.inapp
                               │                      │
                          email-worker           inapp-worker ──AMQP──► amq.topic
                                                                              │
                                                                         MQTT plugin
                                                                              │
                                                                    device-simulator
```

Three binaries:

| Service | Role |
|---|---|
| `cmd/gateway` | HTTP in, gRPC out. Auth, idempotency key header, rate limiting. |
| `cmd/users` | User preferences, quiet hours, device registry. |
| `cmd/notifier` | Core logic: decide, render, persist, publish, track. Runs all workers. |

## Tech stack

| Component | Technology |
|---|---|
| Language | Go 1.26 |
| Service communication | gRPC + Protocol Buffers (buf) |
| Async messaging | RabbitMQ 3.13 (AMQP) |
| In-app push | MQTT via RabbitMQ MQTT plugin |
| Database | PostgreSQL 16 (GORM + pgx/v5 driver) |
| Observability | OpenTelemetry + Prometheus |

## Getting started

### Prerequisites

- Docker & Docker Compose

That's it. No local Go install required to run the system.

### Run everything

```bash
make up
```

Builds and starts all six services in Docker:

| Service | Port | Role |
|---|---|---|
| gateway | 8080 | HTTP entry point |
| notifier | 9090 | Core gRPC service |
| users | 9091 | Preferences gRPC service |
| postgres | 5433 | Notification storage |
| rabbitmq | 5672 / 1883 / 15672 | Message broker + MQTT |
| device-sim | — | 100 simulated MQTT devices |

RabbitMQ management UI is at `http://localhost:15672` (user: `notify`, pass: `notify`).

### Stop everything

```bash
make down
```

### Watch logs

```bash
make logs
```

---

## Try it

**1. Start the system:**
```bash
make up
```

**2. Send a notification** (wait ~10s for services to be ready):
```bash
curl -s -X POST http://localhost:8080/v1/notifications \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-1","type":"new_episode","payload":{"title":"Episode 5"}}' | jq
```

**Response:**
```json
{
  "notification_id": "a3f1c2d4-...",
  "decisions": [
    { "channel": "email",  "status": "PENDING" },
    { "channel": "inapp",  "status": "PENDING" }
  ]
}
```

The gateway accepted the request, the notifier fetched user-1's preferences, decided email + in-app are enabled, and returned immediately.

**3. Check the notifier logs:**
```bash
docker compose -f deploy/docker-compose.yml logs notifier
```

You'll see the channel routing decision logged per request.

**4. Send the same request again** (same `Idempotency-Key`):
```bash
KEY=$(uuidgen)

curl -s -X POST http://localhost:8080/v1/notifications \
  -H "Idempotency-Key: $KEY" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-1","type":"new_episode","payload":{"title":"Episode 5"}}' | jq .notification_id

# Second call — returns the same ID, nothing is sent twice
curl -s -X POST http://localhost:8080/v1/notifications \
  -H "Idempotency-Key: $KEY" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-1","type":"new_episode","payload":{"title":"Episode 5"}}' | jq .notification_id
```

Both calls return the **same** notification ID — idempotency working.

> **Coming in Steps 5–7:** email worker logs (`sent email to user-1`), delivery status changing `PENDING → SENT`, and device-sim receiving the in-app message over MQTT.

---

## Development (without Docker)

### Prerequisites

- Go 1.26+
- Docker & Docker Compose (for infra only)
- [buf](https://buf.build/docs/installation) (proto regeneration only)

```bash
make infra-up        # start Postgres + RabbitMQ only

make dev-users       # users-service on :9091
make dev-notifier    # notifier on :9090
make dev-gateway     # HTTP gateway on :8080
```

### Regenerate proto

```bash
make proto
```

### Run tests

```bash
make test
make test-race
```

## Build plan

This project is actively being built. Current progress:

- [x] Step 1 — Foundation: repo layout, protos, docker-compose, lifecycle
- [x] Step 2 — Domain model: pure types, port interfaces, policy functions
- [x] Step 3 — Sync path: gateway → notifier → users over gRPC (in-memory)
- [x] Step 4 — Persistence + idempotency: GORM models, Postgres repository, outbox pattern (wiring in progress)
- [ ] Step 5 — Async delivery: RabbitMQ outbox relay, email + SMS workers
- [ ] Step 6 — Reliability: retry tiers, DLQ, result events, status stream
- [ ] Step 7 — In-app + MQTT: inapp-worker, device simulator (100 goroutines)
- [ ] Step 8 — Hardening: priorities, gRPC interceptors, graceful shutdown, OTel tracing, Prometheus metrics
- [ ] Step 9 — Integration test suite: all 7 failure scenarios with testcontainers (Postgres + RabbitMQ)

See [PLAN.md](PLAN.md) for the full architecture and design decisions.

## Failure scenarios (the real point)

| Scenario | Expected behavior |
|---|---|
| Client retries the request | Same notification ID, nothing sent twice |
| Worker crashes mid-send | Message redelivered; unique delivery row prevents double send |
| Email provider errors | Backoff retries → DLQ → status `DEAD` |
| RabbitMQ down during send | Outbox holds rows; relay publishes on broker return |
| users-service slow | gRPC deadline fires; API returns 503 instead of hanging |
| Device offline | MQTT queues message; delivered on reconnect |
| Marketing blast of 100k | Transactional queue stays fast |

## License

MIT
