package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
)

type AppSettingsService struct {
	appSettings repository.AppSettingsRepository
}

func NewAppSettingsService(appSettings repository.AppSettingsRepository) *AppSettingsService {
	return &AppSettingsService{appSettings: appSettings}
}

// GetSettings retrieves app settings (creates default if not exists)
func (s *AppSettingsService) GetSettings(ctx context.Context) (*models.AppSettings, error) {
	settings, err := s.appSettings.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	if settings == nil {
		// Create default settings
		settings = &models.AppSettings{
			ID:                uuid.New().String(),
			AppName:           "Holy Home",
			DefaultLanguage:   "en",
			DisableAutoDetect: false,
			UpdatedAt:         time.Now(),
		}

		if err := s.appSettings.Upsert(ctx, settings); err != nil {
			return nil, fmt.Errorf("failed to create default app settings: %w", err)
		}

		return settings, nil
	}

	return settings, nil
}

// SupportedLanguages defines the list of supported locale codes
var SupportedLanguages = []string{"en", "pl"}

// IsLanguageSupported checks if a language code is supported
func IsLanguageSupported(lang string) bool {
	for _, l := range SupportedLanguages {
		if l == lang {
			return true
		}
	}
	return false
}

// UpdateSettingsInput holds the input for updating app settings
type UpdateSettingsInput struct {
	AppName           *string `json:"appName"`
	DefaultLanguage   *string `json:"defaultLanguage"`
	DisableAutoDetect *bool   `json:"disableAutoDetect"`
}

// UpdateSettings updates app settings (ADMIN only - enforced at handler)
func (s *AppSettingsService) UpdateSettings(ctx context.Context, input UpdateSettingsInput) error {
	// Get current settings
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}

	if input.AppName != nil {
		if *input.AppName == "" {
			return errors.New("app name cannot be empty")
		}
		settings.AppName = *input.AppName
	}

	if input.DefaultLanguage != nil {
		if !IsLanguageSupported(*input.DefaultLanguage) {
			return fmt.Errorf("unsupported language: %s", *input.DefaultLanguage)
		}
		settings.DefaultLanguage = *input.DefaultLanguage
	}

	if input.DisableAutoDetect != nil {
		settings.DisableAutoDetect = *input.DisableAutoDetect
	}

	settings.UpdatedAt = time.Now()

	if err := s.appSettings.Upsert(ctx, settings); err != nil {
		return fmt.Errorf("failed to update app settings: %w", err)
	}

	return nil
}
