package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	notificationport "github.com/mohsin123321/notification-system/internal/domain/notification/port"
	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

// ErrNotFound is returned when a notification is not found.
var ErrNotFound = errors.New("notification not found")

// Service implements notification business logic.
type Service struct {
	repo    notificationport.NotificationRepository
	prefs   notificationport.PreferenceClient
	pub     notificationport.Publisher     // may be nil in early steps
	watcher notificationport.StatusWatcher // may be nil in early steps
}

// New creates a new notification Service.
func New(
	repo notificationport.NotificationRepository,
	prefs notificationport.PreferenceClient,
	pub notificationport.Publisher,
	watcher notificationport.StatusWatcher,
) *Service {
	return &Service{repo: repo, prefs: prefs, pub: pub, watcher: watcher}
}

// SendRequest holds the data required to send a notification.
type SendRequest struct {
	UserID         pure.UserID
	Type           pure.NotificationType
	IdempotencyKey string
	Payload        map[string]any
}

// Send creates a notification, decides channels, and persists everything.
func (s *Service) Send(ctx context.Context, req SendRequest) (*pure.Notification, error) {
	// 1. Check idempotency.
	if req.IdempotencyKey != "" {
		existing, err := s.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
		if err == nil {
			return existing, nil
		}
	}

	// 2. Fetch preferences.
	prefs, err := s.prefs.GetPreferences(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	// 3. Decide channels.
	decisions := pure.DecideChannels(req.Type, *prefs, time.Now())

	// 4. Build notification.
	notif := &pure.Notification{
		ID:             pure.NotificationID(uuid.New().String()),
		UserID:         req.UserID,
		Type:           req.Type,
		Payload:        req.Payload,
		IdempotencyKey: req.IdempotencyKey,
		CreatedAt:      time.Now(),
	}
	for _, d := range decisions {
		notif.Deliveries = append(notif.Deliveries, pure.Delivery{
			ID:             pure.DeliveryID(uuid.New().String()),
			NotificationID: notif.ID,
			Channel:        d.Channel,
			Status:         d.Status,
			UpdatedAt:      time.Now(),
		})
	}

	// 5. Persist (repo builds outbox internally).
	if err := s.repo.Save(ctx, notif); err != nil {
		return nil, err
	}
	return notif, nil
}

// Get retrieves a notification by ID.
func (s *Service) Get(ctx context.Context, id pure.NotificationID) (*pure.Notification, error) {
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return n, nil
}

// UpdateDeliveryStatus persists a delivery status change and notifies watchers.
func (s *Service) UpdateDeliveryStatus(ctx context.Context, event pure.StatusEvent) error {
	d := &pure.Delivery{
		ID:             event.DeliveryID,
		NotificationID: event.NotificationID,
		Channel:        event.Channel,
		Status:         event.Status,
		LastError:      event.Error,
		UpdatedAt:      event.UpdatedAt,
	}
	if err := s.repo.UpdateDeliveryStatus(ctx, d); err != nil {
		return err
	}
	if s.watcher != nil {
		_ = s.watcher.Notify(ctx, event)
	}
	return nil
}

// WatchStatus returns a channel that receives status events for a notification.
func (s *Service) WatchStatus(ctx context.Context, id pure.NotificationID) (<-chan pure.StatusEvent, error) {
	if s.watcher == nil {
		ch := make(chan pure.StatusEvent)
		close(ch)
		return ch, nil
	}
	return s.watcher.Subscribe(ctx, id)
}

// UnwatchStatus removes a status subscription.
func (s *Service) UnwatchStatus(ctx context.Context, id pure.NotificationID, ch <-chan pure.StatusEvent) {
	if s.watcher != nil {
		s.watcher.Unsubscribe(ctx, id, ch)
	}
}
