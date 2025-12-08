package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// SessionRow represents a session row in SQLite
type SessionRow struct {
	ID           string `db:"id"`
	UserID       string `db:"user_id"`
	RefreshToken string `db:"refresh_token"`
	Name         string `db:"name"`
	IPAddress    string `db:"ip_address"`
	UserAgent    string `db:"user_agent"`
	CreatedAt    string `db:"created_at"`
	LastUsedAt   string `db:"last_used_at"`
	ExpiresAt    string `db:"expires_at"`
}

// SessionRepository implements repository.SessionRepository for SQLite
type SessionRepository struct {
	db *sqlx.DB
}

// NewSessionRepository creates a new SQLite session repository
func NewSessionRepository(db *sqlx.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(ctx context.Context, session *models.Session) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO sessions (id, user_id, refresh_token, name, ip_address, user_agent, created_at, last_used_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		id,
		session.UserID,
		session.RefreshToken,
		session.Name,
		session.IPAddress,
		session.UserAgent,
		now,
		now,
		session.ExpiresAt.UTC().Format(time.RFC3339),
	)
	return err
}

// GetByID retrieves a session by ID
func (r *SessionRepository) GetByID(ctx context.Context, id string) (*models.Session, error) {
	var row SessionRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM sessions WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToSession(&row), nil
}

// GetByRefreshToken retrieves a session by refresh token hash
func (r *SessionRepository) GetByRefreshToken(ctx context.Context, tokenHash string) (*models.Session, error) {
	var row SessionRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM sessions WHERE refresh_token = ?", tokenHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToSession(&row), nil
}

// Update updates an existing session
func (r *SessionRepository) Update(ctx context.Context, session *models.Session) error {
	query := `
		UPDATE sessions SET
			refresh_token = ?,
			name = ?,
			last_used_at = ?,
			expires_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		session.RefreshToken,
		session.Name,
		session.LastUsedAt.UTC().Format(time.RFC3339),
		session.ExpiresAt.UTC().Format(time.RFC3339),
		session.ID,
	)
	return err
}

// Delete deletes a session
func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", id)
	return err
}

// DeleteByUserID deletes all sessions for a user
func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// DeleteExpired deletes all expired sessions
func (r *SessionRepository) DeleteExpired(ctx context.Context) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < ?", now)
	return err
}

// ListByUserID returns all sessions for a user
func (r *SessionRepository) ListByUserID(ctx context.Context, userID string) ([]models.Session, error) {
	var rows []SessionRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM sessions WHERE user_id = ? ORDER BY last_used_at DESC", userID)
	if err != nil {
		return nil, err
	}
	return rowsToSessions(rows), nil
}

func rowToSession(row *SessionRow) *models.Session {
	session := &models.Session{
		ID:           row.ID,
		UserID:       row.UserID,
		RefreshToken: row.RefreshToken,
		Name:         row.Name,
		IPAddress:    row.IPAddress,
		UserAgent:    row.UserAgent,
	}
	session.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	session.LastUsedAt, _ = time.Parse(time.RFC3339, row.LastUsedAt)
	session.ExpiresAt, _ = time.Parse(time.RFC3339, row.ExpiresAt)
	return session
}

func rowsToSessions(rows []SessionRow) []models.Session {
	sessions := make([]models.Session, len(rows))
	for i, row := range rows {
		sessions[i] = *rowToSession(&row)
	}
	return sessions
}
