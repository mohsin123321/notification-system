package pure

import "time"

type DeliveryID string

type Delivery struct {
	ID             DeliveryID
	NotificationID NotificationID
	Channel        Channel
	Status         DeliveryStatus
	Attempts       int
	LastError      string
	UpdatedAt      time.Time
}

func NewDelivery(notifID NotificationID, ch Channel, status DeliveryStatus) Delivery {
	return Delivery{
		NotificationID: notifID,
		Channel:        ch,
		Status:         status,
		UpdatedAt:      time.Now(),
	}
}
