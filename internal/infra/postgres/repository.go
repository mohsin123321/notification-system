package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

var ErrNotFound = errors.New("not found")

// Repository implements notification/port.NotificationRepository using PostgreSQL via GORM.
type Repository struct {
	db *gorm.DB
}

func NewRepository(pg *Postgres) *Repository {
	return &Repository{db: pg.DB()}
}

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

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO notifications (id, user_id, type, payload, idempotency_key, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT (idempotency_key) DO NOTHING
		`, string(n.ID), string(n.UserID), string(n.Type), payloadBytes, n.IdempotencyKey, n.CreatedAt).Error; err != nil {
			return fmt.Errorf("insert notification: %w", err)
		}

		for _, d := range n.Deliveries {
			if err := tx.Exec(`
				INSERT INTO deliveries (id, notification_id, channel, status, attempts, last_error, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, string(d.ID), string(d.NotificationID), string(d.Channel), string(d.Status), d.Attempts, d.LastError, d.UpdatedAt).Error; err != nil {
				return fmt.Errorf("insert delivery: %w", err)
			}

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
				return fmt.Errorf("marshal outbox message: %w", err)
			}

			if err := tx.Exec(`
				INSERT INTO outbox (id, routing_key, body)
				VALUES (?, ?, ?)
			`, uuid.New().String(), pure.RoutingKey(d.Channel, n.Type.Priority()), body).Error; err != nil {
				return fmt.Errorf("insert outbox: %w", err)
			}
		}
		return nil
	})
}

func (r *Repository) GetByID(ctx context.Context, id pure.NotificationID) (*pure.Notification, error) {
	return r.queryOne(ctx, "n.id = ?", string(id))
}

func (r *Repository) GetByIdempotencyKey(ctx context.Context, key string) (*pure.Notification, error) {
	return r.queryOne(ctx, "n.idempotency_key = ?", key)
}

func (r *Repository) queryOne(ctx context.Context, where string, arg any) (*pure.Notification, error) {
	type row struct {
		NID, NUserID, NType, NIdempKey string
		NPayload                        []byte
		NCreatedAt                      time.Time
		DID, DNotifID, DChannel, DStatus *string
		DAttempts                        *int
		DLastError                       *string
		DUpdatedAt                       *time.Time
	}

	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT n.id AS n_id, n.user_id AS n_user_id, n.type AS n_type,
		       n.payload AS n_payload, n.idempotency_key AS n_idemp_key, n.created_at AS n_created_at,
		       d.id AS d_id, d.notification_id AS d_notif_id, d.channel AS d_channel,
		       d.status AS d_status, d.attempts AS d_attempts, d.last_error AS d_last_error,
		       d.updated_at AS d_updated_at
		FROM notifications n
		LEFT JOIN deliveries d ON d.notification_id = n.id
		WHERE `+where, arg).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}

	var payload map[string]any
	if err := json.Unmarshal(rows[0].NPayload, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	n := &pure.Notification{
		ID:             pure.NotificationID(rows[0].NID),
		UserID:         pure.UserID(rows[0].NUserID),
		Type:           pure.NotificationType(rows[0].NType),
		Payload:        payload,
		IdempotencyKey: rows[0].NIdempKey,
		CreatedAt:      rows[0].NCreatedAt,
	}
	for _, row := range rows {
		if row.DID != nil {
			n.Deliveries = append(n.Deliveries, pure.Delivery{
				ID:             pure.DeliveryID(*row.DID),
				NotificationID: pure.NotificationID(*row.DNotifID),
				Channel:        pure.Channel(*row.DChannel),
				Status:         pure.DeliveryStatus(*row.DStatus),
				Attempts:       *row.DAttempts,
				LastError:      *row.DLastError,
				UpdatedAt:      *row.DUpdatedAt,
			})
		}
	}
	return n, nil
}

func (r *Repository) UpdateDeliveryStatus(ctx context.Context, d *pure.Delivery) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE deliveries SET status = ?, attempts = attempts + 1, last_error = ?, updated_at = now()
		WHERE id = ?
	`, string(d.Status), d.LastError, string(d.ID))
	if result.Error != nil {
		return fmt.Errorf("update delivery: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) PendingOutbox(ctx context.Context, limit int) ([]pure.OutboxMessage, error) {
	var results []struct {
		ID         string
		RoutingKey string
		Body       []byte
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, routing_key, body FROM outbox
		WHERE published_at IS NULL
		LIMIT ?
		FOR UPDATE SKIP LOCKED
	`, limit).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("pending outbox: %w", err)
	}

	msgs := make([]pure.OutboxMessage, len(results))
	for i, res := range results {
		msgs[i] = pure.OutboxMessage{
			ID:         pure.OutboxID(res.ID),
			RoutingKey: res.RoutingKey,
			Body:       res.Body,
		}
	}
	return msgs, nil
}

func (r *Repository) MarkOutboxPublished(ctx context.Context, id pure.OutboxID) error {
	return r.db.WithContext(ctx).Exec(
		"UPDATE outbox SET published_at = now() WHERE id = ?", string(id),
	).Error
}
