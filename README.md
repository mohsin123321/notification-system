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

- Go 1.26+
- Docker & Docker Compose
- [buf](https://buf.build/docs/installation) (proto generation)

### Run infrastructure

```bash
make infra-up
```

Starts Postgres 16 and RabbitMQ 3.13 with the MQTT plugin enabled on port 1883.

Database tables are created automatically via GORM AutoMigrate when the notifier service starts.

### Regenerate proto

```bash
make proto
```

### Run services

```bash
make dev-gateway    # HTTP gateway on :8080
make dev-users      # users-service gRPC on :50052
make dev-notifier   # notification-service gRPC on :50051
```

### Send a notification

```bash
curl -X POST http://localhost:8080/v1/notifications \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-1","type":"new_episode","payload":{"title":"Episode 5"}}'
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
