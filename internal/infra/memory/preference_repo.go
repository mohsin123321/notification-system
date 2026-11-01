package memory

import (
	"context"

	preferencepure "github.com/mohsin123321/notification-system/internal/domain/preference/pure"
)

// PreferenceRepo is an in-memory implementation of preference/port.PreferenceRepository.
// It returns default preferences (all channels enabled) for any userID.
// Used by the users-service stub.
type PreferenceRepo struct{}

// NewPreferenceRepo creates a new in-memory PreferenceRepo.
func NewPreferenceRepo() *PreferenceRepo {
	return &PreferenceRepo{}
}

// GetByUserID returns default preferences for the given user.
func (r *PreferenceRepo) GetByUserID(_ context.Context, userID string) (*preferencepure.Preferences, error) {
	return &preferencepure.Preferences{
		UserID: userID,
		ChannelPrefs: []preferencepure.ChannelPreference{
			{Channel: "email", Enabled: true},
			{Channel: "sms", Enabled: true},
			{Channel: "inapp", Enabled: true},
		},
		QuietHours: preferencepure.QuietHours{Enabled: false},
	}, nil
}

// Save is a no-op stub — in-memory store does not persist.
func (r *PreferenceRepo) Save(_ context.Context, _ *preferencepure.Preferences) error {
	return nil
}
