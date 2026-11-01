package postgres

import "time"

func init() {
	RegisterModels(&Notification{}, &Delivery{}, &Outbox{})
}

type Notification struct {
	ID             string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID         string    `gorm:"column:user_id;type:uuid;not null;index"`
	Type           string    `gorm:"column:type;type:varchar(64);not null"`
	Payload        []byte    `gorm:"column:payload;type:jsonb"`
	IdempotencyKey string    `gorm:"column:idempotency_key;type:varchar(255);uniqueIndex;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (Notification) TableName() string { return "notifications" }

type Delivery struct {
	ID             string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	NotificationID string    `gorm:"column:notification_id;type:uuid;not null;uniqueIndex:idx_delivery_unique"`
	Channel        string    `gorm:"column:channel;type:varchar(32);not null;uniqueIndex:idx_delivery_unique"`
	Status         string    `gorm:"column:status;type:varchar(32);not null;default:'PENDING'"`
	Attempts       int       `gorm:"column:attempts;default:0"`
	LastError      string    `gorm:"column:last_error;type:text"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Delivery) TableName() string { return "deliveries" }

type Outbox struct {
	ID          string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	RoutingKey  string     `gorm:"column:routing_key;type:varchar(128);not null"`
	Body        []byte     `gorm:"column:body;type:jsonb;not null"`
	PublishedAt *time.Time `gorm:"column:published_at"`
}

func (Outbox) TableName() string { return "outbox" }
