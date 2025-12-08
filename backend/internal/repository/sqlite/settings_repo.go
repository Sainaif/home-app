package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// AppSettingsRow represents app settings row in SQLite
type AppSettingsRow struct {
	ID                string `db:"id"`
	AppName           string `db:"app_name"`
	DefaultLanguage   string `db:"default_language"`
	DisableAutoDetect int    `db:"disable_auto_detect"`
	UpdatedAt         string `db:"updated_at"`
}

// AppSettingsRepository implements repository.AppSettingsRepository for SQLite
type AppSettingsRepository struct {
	db *sqlx.DB
}

// NewAppSettingsRepository creates a new SQLite app settings repository
func NewAppSettingsRepository(db *sqlx.DB) *AppSettingsRepository {
	return &AppSettingsRepository{db: db}
}

// Get retrieves the app settings singleton
func (r *AppSettingsRepository) Get(ctx context.Context) (*models.AppSettings, error) {
	var row AppSettingsRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM app_settings WHERE id = 'singleton'")
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToAppSettings(&row), nil
}

// Upsert creates or updates app settings
func (r *AppSettingsRepository) Upsert(ctx context.Context, settings *models.AppSettings) error {
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO app_settings (id, app_name, default_language, disable_auto_detect, updated_at)
		VALUES ('singleton', ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			app_name = excluded.app_name,
			default_language = excluded.default_language,
			disable_auto_detect = excluded.disable_auto_detect,
			updated_at = excluded.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		settings.AppName,
		settings.DefaultLanguage,
		boolToInt(settings.DisableAutoDetect),
		now,
	)
	return err
}

func rowToAppSettings(row *AppSettingsRow) *models.AppSettings {
	settings := &models.AppSettings{
		ID:                row.ID,
		AppName:           row.AppName,
		DefaultLanguage:   row.DefaultLanguage,
		DisableAutoDetect: intToBool(row.DisableAutoDetect),
	}
	settings.UpdatedAt, _ = time.Parse(time.RFC3339, row.UpdatedAt)
	return settings
}
