package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// PasswordResetTokenRow represents a password reset token row in SQLite
type PasswordResetTokenRow struct {
	ID               string  `db:"id"`
	UserID           string  `db:"user_id"`
	TokenHash        string  `db:"token_hash"`
	ExpiresAt        string  `db:"expires_at"`
	Used             int     `db:"used"`
	UsedAt           *string `db:"used_at"`
	CreatedAt        string  `db:"created_at"`
	CreatedByAdminID string  `db:"created_by_admin_id"`
}

// PasswordResetTokenRepository implements repository.PasswordResetTokenRepository for SQLite
type PasswordResetTokenRepository struct {
	db *sqlx.DB
}

// NewPasswordResetTokenRepository creates a new SQLite password reset token repository
func NewPasswordResetTokenRepository(db *sqlx.DB) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{db: db}
}

// Create creates a new password reset token
func (r *PasswordResetTokenRepository) Create(ctx context.Context, token *models.PasswordResetToken) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, used, created_at, created_by_admin_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt.UTC().Format(time.RFC3339),
		boolToInt(token.Used),
		now,
		token.CreatedByAdminID,
	)
	return err
}

// GetByTokenHash retrieves a token by its hash
func (r *PasswordResetTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error) {
	var row PasswordResetTokenRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM password_reset_tokens WHERE token_hash = ?", tokenHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToPasswordResetToken(&row), nil
}

// MarkUsed marks a token as used
func (r *PasswordResetTokenRepository) MarkUsed(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, "UPDATE password_reset_tokens SET used = 1, used_at = ? WHERE id = ?", now, id)
	return err
}

// DeleteByUserID deletes all tokens for a user
func (r *PasswordResetTokenRepository) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM password_reset_tokens WHERE user_id = ?", userID)
	return err
}

// DeleteExpired deletes all expired tokens
func (r *PasswordResetTokenRepository) DeleteExpired(ctx context.Context) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, "DELETE FROM password_reset_tokens WHERE expires_at < ?", now)
	return err
}

func rowToPasswordResetToken(row *PasswordResetTokenRow) *models.PasswordResetToken {
	token := &models.PasswordResetToken{
		ID:               row.ID,
		UserID:           row.UserID,
		TokenHash:        row.TokenHash,
		CreatedByAdminID: row.CreatedByAdminID,
		Used:             intToBool(row.Used),
	}
	token.ExpiresAt, _ = time.Parse(time.RFC3339, row.ExpiresAt)
	token.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)

	if row.UsedAt != nil {
		t, _ := time.Parse(time.RFC3339, *row.UsedAt)
		token.UsedAt = &t
	}

	return token
}
