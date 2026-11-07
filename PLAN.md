# Notification System — Implementation Plan

## Problem

A streaming app needs to notify users (video processed, password reset, new episode available) but without a central system every team sends from its own code — causing duplicate sends, no user control over channels, no delivery tracking, and slow providers blocking requests.

**Goal:** a central platform where any service says "notify user X about event Y" and the platform decides whether, where, and how to deliver. Delivery must be reliable and effectively-once from the user's perspective.

---

## Architecture

### Pattern
DDD + Hexagonal (ports and adapters).

### Services (3 binaries)

| Binary | Responsibility |
|---|---|
| `cmd/gateway` | HTTP in, gRPC out. Auth, Idempotency-Key header, rate limiting. No domain logic. |
| `cmd/users` | Owns user preferences, quiet hours, device registry. |
| `cmd/notifier` | Core: decide, render, persist, publish, track. Runs all workers as goroutines. |

### Bounded Contexts

| Domain | Lives in | Owns |
|---|---|---|
| `notification` | notifier | Notification aggregate, Delivery child entity, idempotency, routing |
| `template` | notifier | Embedded template rendering (no DB, no repository) |
| `preference` | users | Channel opt-ins, quiet hours per user |

> No separate `delivery` domain. `Delivery` is a child entity inside the `Notification` aggregate — they are always created together in one transaction.

### Full Directory Layout

```
/
├── proto/
│   ├── notification.proto
│   └── users.proto
│
├── cmd/
│   ├── gateway/main.go
│   ├── users/main.go
│   └── notifier/main.go
│
├── internal/
│   ├── domain/
│   │   ├── notification/
│   │   │   ├── port/
│   │   │   │   ├── repository.go     ← NotificationRepository
│   │   │   │   ├── preference.go     ← PreferenceClient
│   │   │   │   ├── publisher.go      ← Publisher (outbox → RabbitMQ)
│   │   │   │   ├── consumer.go       ← Consumer (AMQP queue → worker)
│   │   │   │   └── watcher.go        ← StatusWatcher (in-memory pub/sub)
│   │   │   └── pure/
│   │   │       ├── notification.go   ← Notification aggregate root
│   │   │       ├── delivery.go       ← Delivery child entity
│   │   │       ├── channel.go        ← Channel value object (email|sms|inapp)
│   │   │       ├── status.go         ← DeliveryStatus value object
│   │   │       ├── priority.go       ← Priority + NotificationType.Priority()
│   │   │       ├── routing.go        ← RoutingKey(channel, priority) string
│   │   │       └── policy.go         ← DecideChannels(type, prefs, time.Time)
│   │   ├── template/
│   │   │   └── pure/
│   │   │       └── renderer.go       ← Render(type, channel, payload) string
│   │   └── preference/
│   │       ├── port/
│   │       │   └── repository.go     ← PreferenceRepository
│   │       └── pure/
│   │           └── preference.go     ← Preferences, QuietHours, ChannelPreference
│   │
│   ├── app/
│   │   ├── notification/
│   │   │   ├── service/
│   │   │   │   └── service.go        ← SendNotification, GetNotification, WatchStatus, UpdateDeliveryStatus
│   │   │   └── worker/
│   │   │       ├── outbox/           ← polls outbox → publishes to RabbitMQ
│   │   │       ├── email/            ← consumes q.email.high + q.email.low
│   │   │       ├── sms/              ← consumes q.sms
│   │   │       ├── inapp/            ← consumes q.inapp → publishes to amq.topic
│   │   │       └── result/           ← consumes notification.events → updates status
│   │   └── preference/
│   │       └── service/
│   │           └── service.go        ← GetPreferences, RegisterDevice
│   │
│   ├── infra/
│   │   ├── postgres/                 ← NotificationRepository impl (pgx/v5)
│   │   ├── rabbit/                   ← Publisher + Consumer impl (amqp091)
│   │   ├── stream/                   ← StatusWatcher impl (in-memory, sync.RWMutex)
│   │   └── grpc/
│   │       ├── userclient/           ← PreferenceClient impl (notifier → users)
│   │       └── gatewayclient/        ← gRPC client (gateway → notifier)
│   │
│   ├── server/
│   │   ├── http/                     ← gateway HTTP handlers
│   │   └── grpc/
│   │       ├── notification/         ← NotificationService gRPC handler (notifier)
│   │       └── users/                ← UsersService gRPC handler (users)
│   │
│   └── platform/
│       ├── config/                   ← config loading (yaml + env)
│       ├── logging/                  ← structured logger setup
│       ├── lifecycle/                ← lifecycle.Component interface (Name/Start/Stop)
│       ├── otel/                     ← OpenTelemetry tracer setup (step 8)
│       └── metrics/                  ← Prometheus metrics setup (step 8)
│
├── deploy/
│   ├── docker-compose.yml
│   └── rabbitmq.conf                 ← enables MQTT plugin on port 1883
│
├── scripts/
│   └── migrate/                      ← DB migration runner
│
├── Makefile
├── buf.gen.yaml
└── go.mod
```

---

## Key Design Decisions

### Aggregate
`Notification` is the aggregate root. `Delivery` is a child entity owned by the aggregate. Both are created in one DB transaction. `Payload` is `map[string]any` — the domain does not care about its contents, only the template renderer does.

### Ports (driven — domain calls infra)
| Port | Interface | Concrete impl |
|---|---|---|
| `NotificationRepository` | `domain/notification/port/` | `infra/postgres/` |
| `PreferenceClient` | `domain/notification/port/` | `infra/grpc/userclient/` |
| `Publisher` | `domain/notification/port/` | `infra/rabbit/` |
| `Consumer` | `domain/notification/port/` | `infra/rabbit/` |
| `StatusWatcher` | `domain/notification/port/` | `infra/stream/` |

### Quiet Hours + Suppression
Pure free function in `pure/policy.go`:
```go
func DecideChannels(t NotificationType, prefs Preferences, now time.Time) []ChannelDecision
```
Zero dependencies. Fully unit testable without mocks. Critical types (password_reset) bypass quiet hours.

### Idempotency
- `IdempotencyKey` value object in `pure/`
- Service checks `GetByIdempotencyKey()` — if found, returns existing notification immediately
- DB `UNIQUE` constraint on `idempotency_key` as safety net for races

### Outbox Pattern
The service calls `repo.Save()`. The repository atomically writes notification + deliveries + outbox rows in one transaction. The service never writes outbox rows directly.

The outbox worker polls `PendingOutbox()`, publishes to RabbitMQ with publisher confirms, then calls `MarkOutboxPublished()`. A crash between publish and mark causes a duplicate publish — safe because workers are idempotent.

### Templates
Embedded Go files (`embed.FS`). Pure function: `Render(type, channel, payload)`. No DB. No repository. Workers call it before sending.

### Priority
`NotificationType.Priority()` returns `high` or `low`. `RoutingKey(channel, priority)` builds the AMQP routing key (e.g. `notify.email.high`). Stored in the outbox row at creation time — RabbitMQ routes it, no business logic in the broker.

### Workers
All workers implement `lifecycle.Component` (`Name/Start/Stop`) — same pattern as carfax subscription workers. Each holds a reference to the service, injects infra via port interfaces.

### Status Stream
The result worker and the gRPC stream handler are two goroutines in the same process. `StatusWatcher` (in-memory: `map[NotificationID][]chan StatusEvent` + `sync.RWMutex`) bridges them. RabbitMQ carries events from workers to the notifier process; in-memory bridges them to the gRPC handler goroutine.

---

## RabbitMQ Topology

```
Exchange "notifications" (topic, durable)
  notify.email.high  →  q.email.high
  notify.email.low   →  q.email.low
  notify.sms.*       →  q.sms
  notify.inapp.*     →  q.inapp

Retry (email example):
  q.email.retry  TTL tiers: 5s → 30s → 2min
  dead-letters back to "notifications" exchange
  after N attempts → q.email.dlq → delivery status DEAD

Exchange "notification.events" (topic, durable)
  result events from workers → result worker → UpdateDeliveryStatus
```

Settings: manual acks, prefetch=10, persistent messages, durable queues.

---

## MQTT Design

- Devices connect with stable client ID, `cleanSession=false`, QoS 1
- Topic `users/{userId}/notifications` — QoS 1, carries messages
- Topic `users/{userId}/badge` — retained, holds unread count
- Last Will on `devices/{id}/status` marks device offline on unclean disconnect
- **Key trick:** inapp-worker has no MQTT client. It publishes to `amq.topic` exchange via AMQP with routing key `users.{userId}.notifications`. The RabbitMQ MQTT plugin delivers it to subscribers on `users/{userId}/notifications`.

---

## gRPC Contracts

```proto
service NotificationService {
  rpc SendNotification(SendRequest) returns (SendResponse);              // unary
  rpc GetNotification(GetRequest) returns (Notification);               // unary
  rpc WatchNotificationStatus(WatchRequest) returns (stream StatusEvent); // server streaming
}

service UsersService {
  rpc GetPreferences(GetPrefsRequest) returns (Preferences);
  rpc RegisterDevice(RegisterDeviceRequest) returns (Device);
}
```

gRPC → HTTP status code mapping in gateway:

| gRPC | HTTP |
|---|---|
| `InvalidArgument` | 400 |
| `NotFound` | 404 |
| `AlreadyExists` | 409 |
| `DeadlineExceeded` | 504 |
| `Unavailable` | 503 |

---

## Database Schema

```sql
notifications(
  id               UUID PRIMARY KEY,
  user_id          UUID NOT NULL,
  type             VARCHAR NOT NULL,
  payload          JSONB,
  idempotency_key  VARCHAR UNIQUE NOT NULL,
  created_at       TIMESTAMPTZ DEFAULT now()
)

deliveries(
  id               UUID PRIMARY KEY,
  notification_id  UUID REFERENCES notifications(id),
  channel          VARCHAR NOT NULL,
  status           VARCHAR NOT NULL DEFAULT 'PENDING',
  attempts         INT DEFAULT 0,
  last_error       TEXT,
  updated_at       TIMESTAMPTZ DEFAULT now(),
  UNIQUE (notification_id, channel)
)

outbox(
  id           UUID PRIMARY KEY,
  routing_key  VARCHAR NOT NULL,
  body         JSONB NOT NULL,
  published_at TIMESTAMPTZ NULL
)
```

Delivery status flow: `PENDING → QUEUED → SENT | FAILED | DEAD | SUPPRESSED`

---

## Makefile

```makefile
# Infrastructure
infra-up:        docker compose up -d
infra-down:      docker compose down
infra-reset:     docker compose down -v && docker compose up -d

# Proto generation
proto:           buf generate

# Wire (DI)
wire-notifier:   cd cmd/notifier && wire
wire-users:      cd cmd/users && wire

# Development
dev-gateway:     go run ./cmd/gateway
dev-users:       go run ./cmd/users
dev-notifier:    go run ./cmd/notifier

# Build
build:           go build -o bin/... ./cmd/...

# Database
migrate:         go run ./scripts/migrate

# Tests
test:            go test ./...
test-race:       go test -race ./...

# Linting
lint:            golangci-lint run ./...
lint-fix:        golangci-lint run --fix ./...
```

---

## Build Steps

### Step 1 — Foundation
Repo layout, `go.mod`, `docker-compose.yml` (Postgres 16 + RabbitMQ 3.13 with MQTT plugin), `deploy/rabbitmq.conf`, `Makefile`, proto files, `buf.gen.yaml`, `platform/lifecycle`.

**Done when:** `make infra-up` brings everything up healthy. `make proto` generates Go code. `make dev-notifier` starts without panicking.

---

### Step 2 — Domain Model
All `pure/` structs and free functions. All port interfaces. Zero infra imports anywhere in `domain/`.

**Done when:** every pure function has a passing unit test. `go build ./internal/domain/...` has zero external imports outside stdlib.

**Unit tests:**
- `DecideChannels` — enabled channel, suppressed channel, critical type bypasses quiet hours, all channels suppressed
- `NotificationType.Priority()` — transactional returns high, marketing returns low
- `RoutingKey(channel, priority)` — correct AMQP routing key format
- `Render(type, channel, payload)` — template renders expected output, missing key returns error

---

### Step 3 — Sync Path (in-memory)
users-service with in-memory preferences. notification-service with in-memory repository. api-gateway HTTP → gRPC. No database, no queue.

**Done when:** `POST /v1/notifications` returns 202 with a channel decision (active or suppressed) per channel.

---

### Step 4 — Persistence + Idempotency
Wire `infra/postgres/` repository and `infra/rabbit/` publisher into `cmd/notifier`. GORM AutoMigrate creates tables on startup. Outbox rows written atomically inside `Save()`. RabbitMQ publisher wired so the outbox worker can publish in Step 5.

**Done when:** a repeated `POST` with the same `Idempotency-Key` header returns the same notification ID. DB rows exist in `notifications`, `deliveries`, and `outbox` with correct statuses.

**Unit tests:** idempotency check in service — mock repo returns existing notification on `GetByIdempotencyKey`, assert service returns it without calling `Save`.

Run `make test` to verify all service tests pass before marking this step done.

---

### Step 5 — Async Delivery
`infra/rabbit/` publisher + consumer. Outbox worker. Email + sms workers with fake providers (log to stdout, configurable failure rate).

**Done when:** a notification progresses `PENDING → QUEUED → SENT` visible in DB. Log shows "sent email to {address}".

**Unit tests:**
- Outbox worker — mock `PendingOutbox` returns rows, mock `Publisher` called once per row, mock `MarkOutboxPublished` called after publish
- Email worker — fake provider with 0% failure rate, assert result event published with `SENT` status
- SMS worker — same pattern as email worker

---

### Step 6 — Reliability
Retry tiers, DLQ. Result events on `notification.events` exchange. Result worker updates delivery status. `WatchNotificationStatus` gRPC stream. `infra/stream/` in-memory StatusWatcher.

**Done when:** a provider with 100% failure rate ends in `DEAD` after retries. The gRPC status stream shows every transition in order.

**Unit tests:**
- `StatusWatcher` — subscribe, publish event, assert subscriber receives it; unsubscribe, assert no further events; concurrent subscribers under `go test -race`
- Result worker — mock repo `UpdateDeliveryStatus` called with correct status on `SENT` event and `FAILED` event

---

### Step 7 — In-App + MQTT
inapp-worker (AMQP → amq.topic). Device simulator: 100 goroutines, MQTT QoS 1, `cleanSession=false`, retained badge, Last Will, random offline/online.

**Done when:** an offline device goroutine receives all queued messages on reconnect. Badge count arrives immediately on a fresh connection.

**Unit tests:**
- Inapp worker — assert AMQP publish uses correct routing key format `users.{userId}.notifications`
- Device simulator — goroutine connects, receives message, disconnects cleanly

---

### Step 8 — Hardening
Priority queues (`q.email.high` / `q.email.low`). gRPC interceptors (logging, request ID). Graceful shutdown via lifecycle.Manager (SIGTERM/SIGINT). OpenTelemetry tracing across HTTP → gRPC → AMQP → worker. Prometheus metrics (queue depth, send latency, failures).

**Done when:** one request shows as a single trace across gateway → notifier → users → email worker. `/metrics` returns queue depth and send latency.

**Unit tests:**
- Logging interceptor — assert request ID is set in outgoing context
- `lifecycle.Manager` — start order, stop reverse order, stop called on all components even if one errors

---

### Step 9 — Integration Test Suite
Full coverage using `testcontainers-go` (Postgres + RabbitMQ). All integration tests from every step live here — nothing in steps 1–8 requires a real container.

**Repository (Postgres):**
- `Save()` writes notification + deliveries + outbox in one transaction
- `GetByIdempotencyKey` returns existing row on duplicate key
- Concurrent `Save` with same key — only one row created

**Async delivery (Postgres + RabbitMQ):**
- Full end-to-end `SendNotification` → `SENT` delivery row
- Outbox relay crash test: publisher killed after publish but before mark — redelivery does not double-send

**Reliability:**
- Provider at 100% failure rate → delivery reaches `DEAD` after N attempts, message lands in DLQ
- Client streams `WatchNotificationStatus` → every status transition arrives in order

**MQTT (RabbitMQ with MQTT plugin):**
- Disconnect device, send 5 notifications, reconnect → all 5 arrive in order
- Retained badge: fresh connect receives current unread count without a new message being sent

**Hardening:**
- Graceful shutdown: SIGTERM while 10 in-flight messages → zero message loss
- Priority: send 100 marketing + 10 password_reset concurrently → password_reset deliveries complete first

**Failure scenarios (from README table):**

| Scenario | Test |
|---|---|
| Client retries request | Same notification ID returned, delivery count stays 1 |
| Worker crashes mid-send | Redelivered message, no duplicate delivery |
| Email provider errors | Status reaches `DEAD` after retries, DLQ has the message |
| RabbitMQ down during send | Outbox holds rows, relay publishes on broker return |
| users-service slow | gRPC deadline fires, API returns 503 |
| Device offline | Messages queued, all delivered on reconnect |
| 2 outbox relays running | Each outbox row published exactly once |

**Done when:** `make test` is green. `make test-race` is green. All scenarios pass against real containers.

---

## Libraries

```
google.golang.org/grpc
github.com/rabbitmq/amqp091-go
github.com/eclipse/paho.mqtt.golang
github.com/jackc/pgx/v5
github.com/testcontainers/testcontainers-go
go.opentelemetry.io/otel
github.com/prometheus/client_golang
github.com/google/wire
buf.build/gen/go (proto generation)
```
