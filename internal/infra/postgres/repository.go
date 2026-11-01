package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// Repository implements notification/port.NotificationRepository using PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a Repository connected to the given DSN.
func New(ctx context.Context, dsn string) (*Repository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Repository{pool: pool}, nil
}

// Close shuts down the connection pool.
func (r *Repository) Close() {
	r.pool.Close()
}

// Save persists a notification with its deliveries and outbox messages in a single transaction.
func (r *Repository) Save(ctx context.Context, n *pure.Notification) error {
	if n.ID == "" {
		n.ID = pure.NotificationID(uuid.New().String())
	}
	for i := range n.Deliveries {
		if n.Deliveries[i].ID == "" {
			n.Deliveries[i].ID = pure.DeliveryID(uuid.New().String())
		}
	}

	payloadBytes, err := json.Marshal(n.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Insert notification.
	_, err = tx.Exec(ctx, `
		INSERT INTO notifications (id, user_id, type, payload, idempotency_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (idempotency_key) DO NOTHING
	`, string(n.ID), string(n.UserID), string(n.Type), payloadBytes, n.IdempotencyKey, n.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}

	// Insert deliveries.
	for _, d := range n.Deliveries {
		_, err = tx.Exec(ctx, `
			INSERT INTO deliveries (id, notification_id, channel, status, attempts, last_error, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, string(d.ID), string(d.NotificationID), string(d.Channel), string(d.Status), d.Attempts, d.LastError, d.UpdatedAt)
		if err != nil {
			return fmt.Errorf("insert delivery: %w", err)
		}

		// Build outbox message for non-suppressed deliveries.
		if d.Status == pure.StatusSuppressed {
			continue
		}

		msg := pure.DeliveryMessage{
			NotificationID: n.ID,
			DeliveryID:     d.ID,
			UserID:         n.UserID,
			Type:           n.Type,
			Channel:        d.Channel,
			Payload:        n.Payload,
			Attempt:        0,
		}
		body, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("marshal delivery message: %w", err)
		}

		outboxID := uuid.New().String()
		_, err = tx.Exec(ctx, `
			INSERT INTO outbox (id, routing_key, body)
			VALUES ($1, $2, $3)
		`, outboxID, pure.RoutingKey(d.Channel, n.Type.Priority()), body)
		if err != nil {
			return fmt.Errorf("insert outbox: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// GetByID returns a notification with all its deliveries.
func (r *Repository) GetByID(ctx context.Context, id pure.NotificationID) (*pure.Notification, error) {
	return r.queryOne(ctx, `
		SELECT n.id, n.user_id, n.type, n.payload, n.idempotency_key, n.created_at,
		       d.id, d.notification_id, d.channel, d.status, d.attempts, d.last_error, d.updated_at
		FROM notifications n
		LEFT JOIN deliveries d ON d.notification_id = n.id
		WHERE n.id = $1
	`, string(id))
}

// GetByIdempotencyKey returns a notification by its idempotency key.
func (r *Repository) GetByIdempotencyKey(ctx context.Context, key string) (*pure.Notification, error) {
	return r.queryOne(ctx, `
		SELECT n.id, n.user_id, n.type, n.payload, n.idempotency_key, n.created_at,
		       d.id, d.notification_id, d.channel, d.status, d.attempts, d.last_error, d.updated_at
		FROM notifications n
		LEFT JOIN deliveries d ON d.notification_id = n.id
		WHERE n.idempotency_key = $1
	`, key)
}

// queryOne executes a notification+deliveries query and assembles the result.
func (r *Repository) queryOne(ctx context.Context, sql string, arg any) (*pure.Notification, error) {
	rows, err := r.pool.Query(ctx, sql, arg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var n *pure.Notification
	for rows.Next() {
		var (
			nID, nUserID, nType, nIdempKey string
			nPayloadBytes                  []byte
			nCreatedAt                     time.Time
			dID, dNotifID, dChannel, dStatus, dLastError *string
			dAttempts                                     *int
			dUpdatedAt                                    *time.Time
		)
		if err := rows.Scan(
			&nID, &nUserID, &nType, &nPayloadBytes, &nIdempKey, &nCreatedAt,
			&dID, &dNotifID, &dChannel, &dStatus, &dAttempts, &dLastError, &dUpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if n == nil {
			var payload map[string]any
			if err := json.Unmarshal(nPayloadBytes, &payload); err != nil {
				return nil, fmt.Errorf("unmarshal payload: %w", err)
			}
			n = &pure.Notification{
				ID:             pure.NotificationID(nID),
				UserID:         pure.UserID(nUserID),
				Type:           pure.NotificationType(nType),
				Payload:        payload,
				IdempotencyKey: nIdempKey,
				CreatedAt:      nCreatedAt,
			}
		}
		if dID != nil {
			n.Deliveries = append(n.Deliveries, pure.Delivery{
				ID:             pure.DeliveryID(*dID),
				NotificationID: pure.NotificationID(*dNotifID),
				Channel:        pure.Channel(*dChannel),
				Status:         pure.DeliveryStatus(*dStatus),
				Attempts:       *dAttempts,
				LastError:      *dLastError,
				UpdatedAt:      *dUpdatedAt,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	if n == nil {
		return nil, ErrNotFound
	}
	return n, nil
}

// UpdateDeliveryStatus updates a delivery's status and increments the attempt counter.
func (r *Repository) UpdateDeliveryStatus(ctx context.Context, d *pure.Delivery) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE deliveries
		SET status = $1, attempts = attempts + 1, last_error = $2, updated_at = now()
		WHERE id = $3
	`, string(d.Status), d.LastError, string(d.ID))
	if err != nil {
		return fmt.Errorf("update delivery: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PendingOutbox returns up to limit unpublished outbox messages, locking them for processing.
func (r *Repository) PendingOutbox(ctx context.Context, limit int) ([]pure.OutboxMessage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, routing_key, body
		FROM outbox
		WHERE published_at IS NULL
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("pending outbox: %w", err)
	}
	defer rows.Close()

	msgs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (pure.OutboxMessage, error) {
		var m pure.OutboxMessage
		var id string
		err := row.Scan(&id, &m.RoutingKey, &m.Body)
		m.ID = pure.OutboxID(id)
		return m, err
	})
	if err != nil {
		return nil, fmt.Errorf("collect outbox rows: %w", err)
	}
	return msgs, nil
}

// MarkOutboxPublished stamps an outbox message as published.
func (r *Repository) MarkOutboxPublished(ctx context.Context, id pure.OutboxID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE outbox SET published_at = now() WHERE id = $1
	`, string(id))
	return err
}
