package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// UserRow represents a user row in SQLite
type UserRow struct {
	ID                 string  `db:"id"`
	Email              string  `db:"email"`
	Username           *string `db:"username"`
	Name               string  `db:"name"`
	PasswordHash       string  `db:"password_hash"`
	Role               string  `db:"role"`
	GroupID            *string `db:"group_id"`
	IsActive           int     `db:"is_active"`
	MustChangePassword int     `db:"must_change_password"`
	TOTPSecret         *string `db:"totp_secret"`
	CreatedAt          string  `db:"created_at"`
}

// UserRepository implements repository.UserRepository for SQLite
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new SQLite user repository
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	var username *string
	if user.Username != "" {
		username = &user.Username
	}

	var totpSecret *string
	if user.TOTPSecret != "" {
		totpSecret = &user.TOTPSecret
	}

	query := `
		INSERT INTO users (id, email, username, name, password_hash, role, group_id, is_active, must_change_password, totp_secret, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		strings.ToLower(user.Email),
		username,
		user.Name,
		user.PasswordHash,
		user.Role,
		user.GroupID,
		boolToInt(user.IsActive),
		boolToInt(user.MustChangePassword),
		totpSecret,
		now,
	)

	return err
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var row UserRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM users WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToUser(&row), nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var row UserRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM users WHERE LOWER(email) = LOWER(?)", email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToUser(&row), nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var row UserRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM users WHERE LOWER(username) = LOWER(?)", username)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToUser(&row), nil
}

// GetByEmailOrUsername retrieves a user by email or username
func (r *UserRepository) GetByEmailOrUsername(ctx context.Context, identifier string) (*models.User, error) {
	var row UserRow
	err := r.db.GetContext(ctx, &row,
		"SELECT * FROM users WHERE LOWER(email) = LOWER(?) OR LOWER(username) = LOWER(?)",
		identifier, identifier)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToUser(&row), nil
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	var username *string
	if user.Username != "" {
		username = &user.Username
	}

	var totpSecret *string
	if user.TOTPSecret != "" {
		totpSecret = &user.TOTPSecret
	}

	query := `
		UPDATE users SET
			email = ?,
			username = ?,
			name = ?,
			password_hash = ?,
			role = ?,
			group_id = ?,
			is_active = ?,
			must_change_password = ?,
			totp_secret = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		strings.ToLower(user.Email),
		username,
		user.Name,
		user.PasswordHash,
		user.Role,
		user.GroupID,
		boolToInt(user.IsActive),
		boolToInt(user.MustChangePassword),
		totpSecret,
		user.ID,
	)

	return err
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	return err
}

// List returns all users
func (r *UserRepository) List(ctx context.Context) ([]models.User, error) {
	var rows []UserRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM users ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	return rowsToUsers(rows), nil
}

// ListActive returns all active users
func (r *UserRepository) ListActive(ctx context.Context) ([]models.User, error) {
	var rows []UserRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM users WHERE is_active = 1 ORDER BY name")
	if err != nil {
		return nil, err
	}
	return rowsToUsers(rows), nil
}

// ListByGroupID returns users in a specific group
func (r *UserRepository) ListByGroupID(ctx context.Context, groupID string) ([]models.User, error) {
	var rows []UserRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM users WHERE group_id = ?", groupID)
	if err != nil {
		return nil, err
	}
	return rowsToUsers(rows), nil
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET password_hash = ? WHERE id = ?", passwordHash, id)
	return err
}

// SetMustChangePassword sets the must_change_password flag
func (r *UserRepository) SetMustChangePassword(ctx context.Context, id string, must bool) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET must_change_password = ? WHERE id = ?", boolToInt(must), id)
	return err
}

// UpdateTOTPSecret updates a user's TOTP secret
func (r *UserRepository) UpdateTOTPSecret(ctx context.Context, id, secret string) error {
	var totpSecret *string
	if secret != "" {
		totpSecret = &secret
	}
	_, err := r.db.ExecContext(ctx, "UPDATE users SET totp_secret = ? WHERE id = ?", totpSecret, id)
	return err
}

// Helper functions

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}

func rowToUser(row *UserRow) *models.User {
	user := &models.User{
		ID:                 row.ID,
		Email:              row.Email,
		Name:               row.Name,
		PasswordHash:       row.PasswordHash,
		Role:               row.Role,
		GroupID:            row.GroupID,
		IsActive:           intToBool(row.IsActive),
		MustChangePassword: intToBool(row.MustChangePassword),
	}

	// Handle optional fields
	if row.Username != nil {
		user.Username = *row.Username
	}
	if row.TOTPSecret != nil {
		user.TOTPSecret = *row.TOTPSecret
	}

	// Parse CreatedAt
	user.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)

	return user
}

func rowsToUsers(rows []UserRow) []models.User {
	users := make([]models.User, len(rows))
	for i, row := range rows {
		users[i] = *rowToUser(&row)
	}
	return users
}
