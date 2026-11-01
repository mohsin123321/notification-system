package port

//go:generate go run go.uber.org/mock/mockgen -source=repository.go -destination=mock/mock_repository.go -package=mock

import (
	"context"

	"github.com/mohsin123321/notification-system/internal/domain/preference/pure"
)

type PreferenceRepository interface {
	GetByUserID(ctx context.Context, userID string) (*pure.Preferences, error)
	Save(ctx context.Context, prefs *pure.Preferences) error
}
