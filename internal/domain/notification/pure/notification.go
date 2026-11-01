package pure

import "time"

type NotificationID string
type UserID string

type Notification struct {
	ID             NotificationID
	UserID         UserID
	Type           NotificationType
	Payload        map[string]any
	IdempotencyKey string
	Deliveries     []Delivery
	CreatedAt      time.Time
}
