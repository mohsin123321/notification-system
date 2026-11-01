package pure

type Priority string

const (
	PriorityHigh Priority = "high"
	PriorityLow  Priority = "low"
)

type NotificationType string

const (
	TypePasswordReset  NotificationType = "password_reset"
	TypeVideoProcessed NotificationType = "video_processed"
	TypeNewEpisode     NotificationType = "new_episode"
	TypeMarketing      NotificationType = "marketing"
)

func (t NotificationType) Priority() Priority {
	switch t {
	case TypePasswordReset, TypeVideoProcessed:
		return PriorityHigh
	default:
		return PriorityLow
	}
}

func (t NotificationType) IsCritical() bool {
	return t == TypePasswordReset
}
