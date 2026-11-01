package memory

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// Repository is an in-memory implementation of notification/port.NotificationRepository.
type Repository struct {
	mu             sync.RWMutex
	notifications  map[pure.NotificationID]*pure.Notification
	byIdempotency  map[string]pure.NotificationID
	outbox         map[pure.OutboxID]*pure.OutboxMessage
}

// NewRepository creates an empty in-memory Repository.
func NewRepository() *Repository {
	return &Repository{
		notifications: make(map[pure.NotificationID]*pure.Notification),
		byIdempotency: make(map[string]pure.NotificationID),
		outbox:        make(map[pure.OutboxID]*pure.OutboxMessage),
	}
}

// Save stores the notification and creates outbox messages for non-suppressed deliveries.
func (r *Repository) Save(_ context.Context, n *pure.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if n.ID == "" {
		n.ID = pure.NotificationID(uuid.New().String())
	}
	for i := range n.Deliveries {
		if n.Deliveries[i].ID == "" {
			n.Deliveries[i].ID = pure.DeliveryID(uuid.New().String())
		}
	}

	// Deep-copy to avoid external mutation.
	clone := copyNotification(n)
	r.notifications[clone.ID] = clone

	if n.IdempotencyKey != "" {
		r.byIdempotency[n.IdempotencyKey] = clone.ID
	}

	// Build outbox messages for pending deliveries.
	for _, d := range clone.Deliveries {
		if d.Status == pure.StatusSuppressed {
			continue
		}
		msg := pure.DeliveryMessage{
			NotificationID: clone.ID,
			DeliveryID:     d.ID,
			UserID:         clone.UserID,
			Type:           clone.Type,
			Channel:        d.Channel,
			Payload:        clone.Payload,
			Attempt:        0,
		}
		body, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		outboxID := pure.OutboxID(uuid.New().String())
		r.outbox[outboxID] = &pure.OutboxMessage{
			ID:         outboxID,
			RoutingKey: pure.RoutingKey(d.Channel, clone.Type.Priority()),
			Body:       body,
		}
	}
	return nil
}

// GetByID returns a notification by its ID.
func (r *Repository) GetByID(_ context.Context, id pure.NotificationID) (*pure.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.notifications[id]
	if !ok {
		return nil, ErrNotFound
	}
	clone := copyNotification(n)
	return clone, nil
}

// GetByIdempotencyKey returns a notification by its idempotency key.
func (r *Repository) GetByIdempotencyKey(_ context.Context, key string) (*pure.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byIdempotency[key]
	if !ok {
		return nil, ErrNotFound
	}
	n, ok := r.notifications[id]
	if !ok {
		return nil, ErrNotFound
	}
	return copyNotification(n), nil
}

// UpdateDeliveryStatus finds the delivery inside its notification and updates it.
func (r *Repository) UpdateDeliveryStatus(_ context.Context, d *pure.Delivery) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.notifications[d.NotificationID]
	if !ok {
		return ErrNotFound
	}
	for i := range n.Deliveries {
		if n.Deliveries[i].ID == d.ID {
			n.Deliveries[i].Status = d.Status
			n.Deliveries[i].LastError = d.LastError
			n.Deliveries[i].Attempts++
			n.Deliveries[i].UpdatedAt = d.UpdatedAt
			return nil
		}
	}
	return ErrNotFound
}

// PendingOutbox returns up to limit outbox messages that have not yet been published.
func (r *Repository) PendingOutbox(_ context.Context, limit int) ([]pure.OutboxMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var msgs []pure.OutboxMessage
	for _, m := range r.outbox {
		if m.PublishedAt == nil {
			msgs = append(msgs, *m)
		}
		if len(msgs) >= limit {
			break
		}
	}
	return msgs, nil
}

// MarkOutboxPublished marks an outbox message as published.
func (r *Repository) MarkOutboxPublished(_ context.Context, id pure.OutboxID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.outbox[id]
	if !ok {
		return ErrNotFound
	}
	now := time.Now()
	m.PublishedAt = &now
	return nil
}

// copyNotification returns a shallow clone of n (with a copied Deliveries slice).
func copyNotification(n *pure.Notification) *pure.Notification {
	clone := *n
	clone.Deliveries = make([]pure.Delivery, len(n.Deliveries))
	copy(clone.Deliveries, n.Deliveries)
	return &clone
}
