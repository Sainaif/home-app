package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
)

type NotificationPreferenceService struct {
	notificationPreferences repository.NotificationPreferenceRepository
}

func NewNotificationPreferenceService(notificationPreferences repository.NotificationPreferenceRepository) *NotificationPreferenceService {
	return &NotificationPreferenceService{notificationPreferences: notificationPreferences}
}

func (s *NotificationPreferenceService) GetPreferences(ctx context.Context, userID string) (*models.NotificationPreference, error) {
	preferences, err := s.notificationPreferences.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if preferences == nil {
		return s.createDefaultPreferences(ctx, userID)
	}
	return preferences, nil
}

func (s *NotificationPreferenceService) UpdatePreferences(ctx context.Context, userID string, preferences map[string]bool, allEnabled bool) (*models.NotificationPreference, error) {
	pref := &models.NotificationPreference{
		ID:          uuid.New().String(),
		UserID:      userID,
		Preferences: preferences,
		AllEnabled:  allEnabled,
		UpdatedAt:   time.Now(),
	}

	if err := s.notificationPreferences.Upsert(ctx, pref); err != nil {
		return nil, err
	}

	// Fetch the updated preferences
	return s.notificationPreferences.GetByUserID(ctx, userID)
}

func (s *NotificationPreferenceService) createDefaultPreferences(ctx context.Context, userID string) (*models.NotificationPreference, error) {
	defaultPreferences := &models.NotificationPreference{
		ID:     uuid.New().String(),
		UserID: userID,
		Preferences: map[string]bool{
			"bill":   true,
			"chore":  true,
			"supply": true,
			"loan":   true,
		},
		AllEnabled: true,
		UpdatedAt:  time.Now(),
	}

	if err := s.notificationPreferences.Upsert(ctx, defaultPreferences); err != nil {
		return nil, err
	}

	return defaultPreferences, nil
}
