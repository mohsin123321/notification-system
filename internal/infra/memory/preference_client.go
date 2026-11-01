package memory

import (
	"context"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

// PreferenceClient is an in-memory implementation of notification/port.PreferenceClient.
// It returns default preferences (all channels enabled, no quiet hours) for any user.
// Used when running the notifier without a real users-service connection.
type PreferenceClient struct{}

// NewPreferenceClient creates a new in-memory PreferenceClient.
func NewPreferenceClient() *PreferenceClient {
	return &PreferenceClient{}
}

// GetPreferences returns default preferences with all channels enabled.
func (c *PreferenceClient) GetPreferences(_ context.Context, _ pure.UserID) (*pure.Preferences, error) {
	return &pure.Preferences{
		Channels: []pure.ChannelPref{
			{Channel: pure.ChannelEmail, Enabled: true},
			{Channel: pure.ChannelSMS, Enabled: true},
			{Channel: pure.ChannelInApp, Enabled: true},
		},
		QuietHours: pure.QuietHours{Enabled: false},
	}, nil
}
