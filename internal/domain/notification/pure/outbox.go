package pure

import "time"

type OutboxID string

type OutboxMessage struct {
	ID          OutboxID
	RoutingKey  string
	Body        []byte
	PublishedAt *time.Time
}

type DeliveryMessage struct {
	NotificationID NotificationID
	DeliveryID     DeliveryID
	UserID         UserID
	Type           NotificationType
	Channel        Channel
	Payload        map[string]any
	Attempt        int
}

type StatusEvent struct {
	NotificationID NotificationID
	DeliveryID     DeliveryID
	Channel        Channel
	Status         DeliveryStatus
	Error          string
	UpdatedAt      time.Time
}
