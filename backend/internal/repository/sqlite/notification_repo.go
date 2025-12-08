package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// NotificationRow represents a notification row in SQLite
type NotificationRow struct {
	ID           string  `db:"id"`
	Channel      string  `db:"channel"`
	TemplateID   string  `db:"template_id"`
	ScheduledFor string  `db:"scheduled_for"`
	SentAt       *string `db:"sent_at"`
	Status       string  `db:"status"`
	Read         int     `db:"read"`
	UserID       *string `db:"user_id"`
	Title        string  `db:"title"`
	Body         string  `db:"body"`
}

// NotificationRepository implements repository.NotificationRepository for SQLite
type NotificationRepository struct {
	db *sqlx.DB
}

// NewNotificationRepository creates a new SQLite notification repository
func NewNotificationRepository(db *sqlx.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create creates a new notification
func (r *NotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	id := uuid.New().String()

	var sentAt *string
	if notification.SentAt != nil {
		sa := notification.SentAt.UTC().Format(time.RFC3339)
		sentAt = &sa
	}

	query := `
		INSERT INTO notifications (id, channel, template_id, scheduled_for, sent_at, status, read, user_id, title, body)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		notification.Channel,
		notification.TemplateID,
		notification.ScheduledFor.UTC().Format(time.RFC3339),
		sentAt,
		notification.Status,
		boolToInt(notification.Read),
		notification.UserID,
		notification.Title,
		notification.Body,
	)
	return err
}

// GetByID retrieves a notification by ID
func (r *NotificationRepository) GetByID(ctx context.Context, id string) (*models.Notification, error) {
	var row NotificationRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM notifications WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToNotification(&row), nil
}

// Update updates an existing notification
func (r *NotificationRepository) Update(ctx context.Context, notification *models.Notification) error {
	var sentAt *string
	if notification.SentAt != nil {
		sa := notification.SentAt.UTC().Format(time.RFC3339)
		sentAt = &sa
	}

	query := `
		UPDATE notifications SET
			channel = ?, template_id = ?, scheduled_for = ?, sent_at = ?, status = ?, read = ?, user_id = ?, title = ?, body = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		notification.Channel,
		notification.TemplateID,
		notification.ScheduledFor.UTC().Format(time.RFC3339),
		sentAt,
		notification.Status,
		boolToInt(notification.Read),
		notification.UserID,
		notification.Title,
		notification.Body,
		notification.ID,
	)
	return err
}

// Delete deletes a notification
func (r *NotificationRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM notifications WHERE id = ?", id)
	return err
}

// List returns all notifications
func (r *NotificationRepository) List(ctx context.Context) ([]models.Notification, error) {
	var rows []NotificationRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM notifications ORDER BY scheduled_for DESC")
	if err != nil {
		return nil, err
	}
	return rowsToNotifications(rows), nil
}

// ListByUserID returns notifications for a user
func (r *NotificationRepository) ListByUserID(ctx context.Context, userID string, limit int) ([]models.Notification, error) {
	var rows []NotificationRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM notifications WHERE user_id = ? ORDER BY scheduled_for DESC LIMIT ?", userID, limit)
	if err != nil {
		return nil, err
	}
	return rowsToNotifications(rows), nil
}

// ListUnreadByUserID returns unread notifications for a user
func (r *NotificationRepository) ListUnreadByUserID(ctx context.Context, userID string) ([]models.Notification, error) {
	var rows []NotificationRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM notifications WHERE user_id = ? AND read = 0 ORDER BY scheduled_for DESC", userID)
	if err != nil {
		return nil, err
	}
	return rowsToNotifications(rows), nil
}

// MarkAsRead marks a notification as read
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE notifications SET read = 1 WHERE id = ?", id)
	return err
}

// MarkAllAsReadForUser marks all notifications as read for a user
func (r *NotificationRepository) MarkAllAsReadForUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE notifications SET read = 1 WHERE user_id = ?", userID)
	return err
}

func rowToNotification(row *NotificationRow) *models.Notification {
	notification := &models.Notification{
		ID:         row.ID,
		Channel:    row.Channel,
		TemplateID: row.TemplateID,
		Status:     row.Status,
		Read:       intToBool(row.Read),
		UserID:     row.UserID,
		Title:      row.Title,
		Body:       row.Body,
	}
	notification.ScheduledFor, _ = time.Parse(time.RFC3339, row.ScheduledFor)

	if row.SentAt != nil {
		t, _ := time.Parse(time.RFC3339, *row.SentAt)
		notification.SentAt = &t
	}

	return notification
}

func rowsToNotifications(rows []NotificationRow) []models.Notification {
	notifications := make([]models.Notification, len(rows))
	for i, row := range rows {
		notifications[i] = *rowToNotification(&row)
	}
	return notifications
}

// NotificationPreferenceRow represents notification preferences in SQLite
type NotificationPreferenceRow struct {
	ID          string `db:"id"`
	UserID      string `db:"user_id"`
	Preferences string `db:"preferences"`
	AllEnabled  int    `db:"all_enabled"`
	UpdatedAt   string `db:"updated_at"`
}

// NotificationPreferenceRepository implements repository.NotificationPreferenceRepository for SQLite
type NotificationPreferenceRepository struct {
	db *sqlx.DB
}

// NewNotificationPreferenceRepository creates a new SQLite notification preference repository
func NewNotificationPreferenceRepository(db *sqlx.DB) *NotificationPreferenceRepository {
	return &NotificationPreferenceRepository{db: db}
}

// GetByUserID retrieves notification preferences for a user
func (r *NotificationPreferenceRepository) GetByUserID(ctx context.Context, userID string) (*models.NotificationPreference, error) {
	var row NotificationPreferenceRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM notification_preferences WHERE user_id = ?", userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToNotificationPreference(&row), nil
}

// Upsert creates or updates notification preferences
func (r *NotificationPreferenceRepository) Upsert(ctx context.Context, pref *models.NotificationPreference) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	prefsJSON, _ := json.Marshal(pref.Preferences)

	query := `
		INSERT INTO notification_preferences (id, user_id, preferences, all_enabled, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			preferences = excluded.preferences,
			all_enabled = excluded.all_enabled,
			updated_at = excluded.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		pref.UserID,
		string(prefsJSON),
		boolToInt(pref.AllEnabled),
		now,
	)
	return err
}

func rowToNotificationPreference(row *NotificationPreferenceRow) *models.NotificationPreference {
	pref := &models.NotificationPreference{
		ID:          row.ID,
		UserID:      row.UserID,
		AllEnabled:  intToBool(row.AllEnabled),
		Preferences: make(map[string]bool),
	}
	pref.UpdatedAt, _ = time.Parse(time.RFC3339, row.UpdatedAt)
	json.Unmarshal([]byte(row.Preferences), &pref.Preferences)
	return pref
}

// WebPushSubscriptionRow represents a web push subscription row in SQLite
type WebPushSubscriptionRow struct {
	ID             string  `db:"id"`
	UserID         string  `db:"user_id"`
	Endpoint       string  `db:"endpoint"`
	ExpirationTime *string `db:"expiration_time"`
	P256dh         string  `db:"p256dh"`
	Auth           string  `db:"auth"`
}

// WebPushSubscriptionRepository implements repository.WebPushSubscriptionRepository for SQLite
type WebPushSubscriptionRepository struct {
	db *sqlx.DB
}

// NewWebPushSubscriptionRepository creates a new SQLite web push subscription repository
func NewWebPushSubscriptionRepository(db *sqlx.DB) *WebPushSubscriptionRepository {
	return &WebPushSubscriptionRepository{db: db}
}

// Create creates a new web push subscription
func (r *WebPushSubscriptionRepository) Create(ctx context.Context, sub *models.WebPushSubscription) error {
	id := uuid.New().String()

	query := `
		INSERT INTO web_push_subscriptions (id, user_id, endpoint, expiration_time, p256dh, auth)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query, id, sub.UserID, sub.Endpoint, sub.ExpirationTime, sub.P256dh, sub.Auth)
	return err
}

// GetByEndpoint retrieves a subscription by endpoint
func (r *WebPushSubscriptionRepository) GetByEndpoint(ctx context.Context, endpoint string) (*models.WebPushSubscription, error) {
	var row WebPushSubscriptionRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM web_push_subscriptions WHERE endpoint = ?", endpoint)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &models.WebPushSubscription{
		ID:             row.ID,
		UserID:         row.UserID,
		Endpoint:       row.Endpoint,
		ExpirationTime: row.ExpirationTime,
		P256dh:         row.P256dh,
		Auth:           row.Auth,
	}, nil
}

// Delete deletes a subscription by endpoint
func (r *WebPushSubscriptionRepository) Delete(ctx context.Context, endpoint string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM web_push_subscriptions WHERE endpoint = ?", endpoint)
	return err
}

// ListByUserID returns subscriptions for a user
func (r *WebPushSubscriptionRepository) ListByUserID(ctx context.Context, userID string) ([]models.WebPushSubscription, error) {
	var rows []WebPushSubscriptionRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM web_push_subscriptions WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}

	subs := make([]models.WebPushSubscription, len(rows))
	for i, row := range rows {
		subs[i] = models.WebPushSubscription{
			ID:             row.ID,
			UserID:         row.UserID,
			Endpoint:       row.Endpoint,
			ExpirationTime: row.ExpirationTime,
			P256dh:         row.P256dh,
			Auth:           row.Auth,
		}
	}
	return subs, nil
}
