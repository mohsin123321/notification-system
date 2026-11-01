package service

import (
	"context"

	preferenceport "github.com/mohsin123321/notification-system/internal/domain/preference/port"
	preferencepure "github.com/mohsin123321/notification-system/internal/domain/preference/pure"
)

// Service handles preference-related business logic.
type Service struct {
	repo preferenceport.PreferenceRepository
}

// New creates a new preference Service.
func New(repo preferenceport.PreferenceRepository) *Service {
	return &Service{repo: repo}
}

// GetPreferences fetches preferences for the given user.
func (s *Service) GetPreferences(ctx context.Context, userID string) (*preferencepure.Preferences, error) {
	return s.repo.GetByUserID(ctx, userID)
}

// RegisterDevice stubs device registration (to be implemented in Step 7).
func (s *Service) RegisterDevice(_ context.Context, _, _ string) error {
	return nil
}
