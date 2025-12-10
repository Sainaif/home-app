package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// SentReminderRow represents a sent reminder row in SQLite
type SentReminderRow struct {
	ID           string `db:"id"`
	UserID       string `db:"user_id"`
	ResourceType string `db:"resource_type"`
	ResourceID   string `db:"resource_id"`
	ReminderType string `db:"reminder_type"`
	SentAt       string `db:"sent_at"`
}

// SentReminderRepository implements repository.SentReminderRepository for SQLite
type SentReminderRepository struct {
	db *sqlx.DB
}

// NewSentReminderRepository creates a new SQLite sent reminder repository
func NewSentReminderRepository(db *sqlx.DB) *SentReminderRepository {
	return &SentReminderRepository{db: db}
}

// Create creates a new sent reminder record
func (r *SentReminderRepository) Create(ctx context.Context, reminder *models.SentReminder) error {
	if reminder.ID == "" {
		reminder.ID = uuid.New().String()
	}
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO sent_reminders (id, user_id, resource_type, resource_id, reminder_type, sent_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		reminder.ID,
		reminder.UserID,
		reminder.ResourceType,
		reminder.ResourceID,
		reminder.ReminderType,
		now,
	)
	return err
}

// Exists checks if a reminder already exists for the given parameters
func (r *SentReminderRepository) Exists(ctx context.Context, userID, resourceType, resourceID, reminderType string) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM sent_reminders
		WHERE user_id = ? AND resource_type = ? AND resource_id = ? AND reminder_type = ?
	`
	err := r.db.GetContext(ctx, &count, query, userID, resourceType, resourceID, reminderType)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountRecentByUserID counts reminders sent to a user since a given time
func (r *SentReminderRepository) CountRecentByUserID(ctx context.Context, userID string, since time.Time) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM sent_reminders
		WHERE user_id = ? AND sent_at >= ?
	`
	err := r.db.GetContext(ctx, &count, query, userID, since.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteOlderThan deletes reminders sent before the given time
func (r *SentReminderRepository) DeleteOlderThan(ctx context.Context, before time.Time) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM sent_reminders WHERE sent_at < ?", before.UTC().Format(time.RFC3339))
	return err
}

// List returns all sent reminders
func (r *SentReminderRepository) List(ctx context.Context) ([]models.SentReminder, error) {
	var rows []SentReminderRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM sent_reminders ORDER BY sent_at DESC")
	if err != nil {
		return nil, err
	}
	return rowsToSentReminders(rows), nil
}

// GetByID retrieves a sent reminder by ID
func (r *SentReminderRepository) GetByID(ctx context.Context, id string) (*models.SentReminder, error) {
	var row SentReminderRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM sent_reminders WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToSentReminder(&row), nil
}

func rowToSentReminder(row *SentReminderRow) *models.SentReminder {
	reminder := &models.SentReminder{
		ID:           row.ID,
		UserID:       row.UserID,
		ResourceType: row.ResourceType,
		ResourceID:   row.ResourceID,
		ReminderType: row.ReminderType,
	}
	reminder.SentAt, _ = time.Parse(time.RFC3339, row.SentAt)
	return reminder
}

func rowsToSentReminders(rows []SentReminderRow) []models.SentReminder {
	reminders := make([]models.SentReminder, len(rows))
	for i, row := range rows {
		reminders[i] = *rowToSentReminder(&row)
	}
	return reminders
}
