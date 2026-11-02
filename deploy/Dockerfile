# Single Dockerfile for all Go services.
# Pass --build-arg SERVICE=<gateway|users|notifier|device-sim> to select the binary.
ARG SERVICE=notifier

# ── Stage 1: build ───────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder
ARG SERVICE
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /bin/service ./cmd/${SERVICE}

# ── Stage 2: runtime ─────────────────────────────────────────────────────────
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/service /bin/service
COPY deploy/config.docker.yaml ./config.local.yaml
EXPOSE 8080 9090 9091
ENTRYPOINT ["/bin/service"]
