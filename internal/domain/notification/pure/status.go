package pure

type DeliveryStatus string

const (
	StatusPending    DeliveryStatus = "PENDING"
	StatusQueued     DeliveryStatus = "QUEUED"
	StatusSent       DeliveryStatus = "SENT"
	StatusFailed     DeliveryStatus = "FAILED"
	StatusDead       DeliveryStatus = "DEAD"
	StatusSuppressed DeliveryStatus = "SUPPRESSED"
)

func (s DeliveryStatus) String() string { return string(s) }

func (s DeliveryStatus) IsTerminal() bool {
	return s == StatusSent || s == StatusDead || s == StatusSuppressed
}
