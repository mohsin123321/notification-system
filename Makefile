.PHONY: up down logs infra-up infra-down infra-reset proto \
        dev-gateway dev-users dev-notifier \
        build test test-race lint lint-fix

# ── Full stack (all services) ─────────────────────────────────────────────────
up:
	docker compose -f deploy/docker-compose.yml up --build -d

down:
	docker compose -f deploy/docker-compose.yml down

logs:
	docker compose -f deploy/docker-compose.yml logs -f

# ── Infrastructure only (Postgres + RabbitMQ) ────────────────────────────────
infra-up:
	docker compose -f deploy/docker-compose.yml up -d postgres rabbitmq

infra-down:
	docker compose -f deploy/docker-compose.yml down

infra-reset:
	docker compose -f deploy/docker-compose.yml down -v && \
	docker compose -f deploy/docker-compose.yml up -d postgres rabbitmq

# Proto generation
proto:
	buf generate

# Development
dev-gateway:
	go run ./cmd/gateway

dev-users:
	go run ./cmd/users

dev-notifier:
	go run ./cmd/notifier

# Build all binaries
build:
	go build -o bin/gateway  ./cmd/gateway
	go build -o bin/users    ./cmd/users
	go build -o bin/notifier ./cmd/notifier

# Tests
test:
	go test ./...

test-race:
	go test -race -v ./... | grep -E "RUN|PASS|FAIL"

# Linting
lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...
