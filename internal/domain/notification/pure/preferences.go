package pure

import "time"

// Preferences holds the data the notification domain needs to make channel decisions.
// It is populated from the users-service and passed into DecideChannels.
type Preferences struct {
	Channels   []ChannelPref
	QuietHours QuietHours
	Email      string
	Phone      string
}

type ChannelPref struct {
	Channel Channel
	Enabled bool
}

type QuietHours struct {
	Enabled   bool
	StartTime string
	EndTime   string
	Timezone  string
}

// Channel returns the preference for a given channel, if one exists.
func (p Preferences) Channel(ch Channel) (ChannelPref, bool) {
	for _, c := range p.Channels {
		if c.Channel == ch {
			return c, true
		}
	}
	return ChannelPref{}, false
}

// IsActive returns true when quiet hours are enabled and now falls within the window.
func (q QuietHours) IsActive(now time.Time) bool {
	if !q.Enabled {
		return false
	}

	loc, err := time.LoadLocation(q.Timezone)
	if err != nil {
		loc = time.UTC
	}

	local := now.In(loc)
	start, err := time.ParseInLocation("15:04", q.StartTime, loc)
	if err != nil {
		return false
	}
	end, err := time.ParseInLocation("15:04", q.EndTime, loc)
	if err != nil {
		return false
	}

	start = time.Date(local.Year(), local.Month(), local.Day(), start.Hour(), start.Minute(), 0, 0, loc)
	end = time.Date(local.Year(), local.Month(), local.Day(), end.Hour(), end.Minute(), 0, 0, loc)

	if end.Before(start) {
		return local.After(start) || local.Before(end)
	}
	return local.After(start) && local.Before(end)
}
