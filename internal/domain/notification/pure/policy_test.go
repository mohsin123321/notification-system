package pure_test

import (
	"testing"
	"time"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

func TestDecideChannels_AllEnabled_NoQuietHours(t *testing.T) {
	prefs := pure.Preferences{
		Channels: []pure.ChannelPref{
			{Channel: pure.ChannelEmail, Enabled: true},
			{Channel: pure.ChannelSMS, Enabled: true},
			{Channel: pure.ChannelInApp, Enabled: true},
		},
	}

	decisions := pure.DecideChannels(pure.TypeNewEpisode, prefs, time.Now())

	if len(decisions) != 3 {
		t.Fatalf("expected 3 decisions, got %d", len(decisions))
	}
	for _, d := range decisions {
		if d.Status != pure.StatusPending {
			t.Errorf("channel %s: expected PENDING, got %s", d.Channel, d.Status)
		}
	}
}

func TestDecideChannels_ChannelDisabled(t *testing.T) {
	prefs := pure.Preferences{
		Channels: []pure.ChannelPref{
			{Channel: pure.ChannelEmail, Enabled: false},
			{Channel: pure.ChannelSMS, Enabled: true},
			{Channel: pure.ChannelInApp, Enabled: true},
		},
	}

	decisions := pure.DecideChannels(pure.TypeNewEpisode, prefs, time.Now())

	for _, d := range decisions {
		if d.Channel == pure.ChannelEmail && d.Status != pure.StatusSuppressed {
			t.Errorf("email should be suppressed when disabled")
		}
	}
}

func TestDecideChannels_QuietHours_NonCritical(t *testing.T) {
	now := time.Date(2024, 1, 1, 23, 0, 0, 0, time.UTC)
	prefs := pure.Preferences{
		Channels: []pure.ChannelPref{
			{Channel: pure.ChannelEmail, Enabled: true},
			{Channel: pure.ChannelSMS, Enabled: true},
			{Channel: pure.ChannelInApp, Enabled: true},
		},
		QuietHours: pure.QuietHours{
			Enabled:   true,
			StartTime: "22:00",
			EndTime:   "08:00",
			Timezone:  "UTC",
		},
	}

	decisions := pure.DecideChannels(pure.TypeMarketing, prefs, now)

	for _, d := range decisions {
		if d.Status != pure.StatusSuppressed {
			t.Errorf("channel %s should be suppressed during quiet hours", d.Channel)
		}
	}
}

func TestDecideChannels_QuietHours_Critical_Bypasses(t *testing.T) {
	now := time.Date(2024, 1, 1, 23, 0, 0, 0, time.UTC)
	prefs := pure.Preferences{
		Channels: []pure.ChannelPref{
			{Channel: pure.ChannelEmail, Enabled: true},
		},
		QuietHours: pure.QuietHours{
			Enabled:   true,
			StartTime: "22:00",
			EndTime:   "08:00",
			Timezone:  "UTC",
		},
	}

	decisions := pure.DecideChannels(pure.TypePasswordReset, prefs, now)

	for _, d := range decisions {
		if d.Channel == pure.ChannelEmail && d.Status != pure.StatusPending {
			t.Errorf("password_reset should bypass quiet hours, got %s", d.Status)
		}
	}
}

func TestRoutingKey(t *testing.T) {
	key := pure.RoutingKey(pure.ChannelEmail, pure.PriorityHigh)
	if key != "notify.email.high" {
		t.Errorf("expected notify.email.high, got %s", key)
	}
}

func TestNotificationType_Priority(t *testing.T) {
	if pure.TypePasswordReset.Priority() != pure.PriorityHigh {
		t.Error("password_reset should be high priority")
	}
	if pure.TypeMarketing.Priority() != pure.PriorityLow {
		t.Error("marketing should be low priority")
	}
}
