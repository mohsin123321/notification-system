package pure

import "time"

type ChannelDecision struct {
	Channel Channel
	Status  DeliveryStatus
	Reason  string
}

// DecideChannels returns one decision per channel based on user preferences and quiet hours.
// Critical notification types bypass quiet hours.
func DecideChannels(t NotificationType, prefs Preferences, now time.Time) []ChannelDecision {
	decisions := make([]ChannelDecision, 0, len(AllChannels()))

	inQuietHours := !t.IsCritical() && prefs.QuietHours.IsActive(now)

	for _, ch := range AllChannels() {
		pref, ok := prefs.Channel(ch)
		if !ok || !pref.Enabled {
			decisions = append(decisions, ChannelDecision{
				Channel: ch,
				Status:  StatusSuppressed,
				Reason:  "channel disabled by user",
			})
			continue
		}

		if inQuietHours {
			decisions = append(decisions, ChannelDecision{
				Channel: ch,
				Status:  StatusSuppressed,
				Reason:  "quiet hours",
			})
			continue
		}

		decisions = append(decisions, ChannelDecision{
			Channel: ch,
			Status:  StatusPending,
		})
	}

	return decisions
}
