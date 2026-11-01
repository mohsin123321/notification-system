package gateway

import "time"

// SendRequest is the JSON body for POST /v1/notifications.
type SendRequest struct {
	UserID  string         `json:"user_id"`
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

// SendResponse is the JSON response for POST /v1/notifications.
type SendResponse struct {
	NotificationID string             `json:"notification_id"`
	Decisions      []DecisionResponse `json:"decisions"`
}

// DecisionResponse holds the routing outcome for one channel.
type DecisionResponse struct {
	Channel string `json:"channel"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
}

// GetResponse is the JSON response for GET /v1/notifications/{id}.
type GetResponse struct {
	NotificationID string             `json:"notification_id"`
	UserID         string             `json:"user_id"`
	Type           string             `json:"type"`
	CreatedAt      time.Time          `json:"created_at"`
	Deliveries     []DeliveryResponse `json:"deliveries"`
}

// DeliveryResponse holds the state of one delivery attempt.
type DeliveryResponse struct {
	Channel   string    `json:"channel"`
	Status    string    `json:"status"`
	Attempts  int       `json:"attempts"`
	LastError string    `json:"last_error,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}
