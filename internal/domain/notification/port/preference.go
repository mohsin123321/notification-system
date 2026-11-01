package port

//go:generate go run go.uber.org/mock/mockgen -source=preference.go -destination=mock/mock_preference.go -package=mock

import (
	"context"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

type PreferenceClient interface {
	GetPreferences(ctx context.Context, userID pure.UserID) (*pure.Preferences, error)
}
