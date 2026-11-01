package pure

import "time"

type SendResult struct {
	NotificationID string
	Decisions      []ChannelDecision
}

type ChannelDecision struct {
	Channel string
	Status  string
	Reason  string
}

type NotificationDetail struct {
	NotificationID string
	UserID         string
	Type           string
	CreatedAt      time.Time
	Deliveries     []DeliveryDetail
}

type DeliveryDetail struct {
	Channel   string
	Status    string
	Attempts  int
	LastError string
	UpdatedAt time.Time
}
