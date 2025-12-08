package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// PasskeyCredentialRow represents a passkey credential row in SQLite
type PasskeyCredentialRow struct {
	ID              string  `db:"id"`
	UserID          string  `db:"user_id"`
	PublicKey       []byte  `db:"public_key"`
	AttestationType string  `db:"attestation_type"`
	AAGUID          []byte  `db:"aaguid"`
	SignCount       int     `db:"sign_count"`
	Name            string  `db:"name"`
	BackupEligible  int     `db:"backup_eligible"`
	BackupState     int     `db:"backup_state"`
	CreatedAt       string  `db:"created_at"`
	LastUsedAt      *string `db:"last_used_at"`
}

// PasskeyCredentialRepository implements repository.PasskeyCredentialRepository for SQLite
type PasskeyCredentialRepository struct {
	db *sqlx.DB
}

// NewPasskeyCredentialRepository creates a new SQLite passkey credential repository
func NewPasskeyCredentialRepository(db *sqlx.DB) *PasskeyCredentialRepository {
	return &PasskeyCredentialRepository{db: db}
}

// Create creates a new passkey credential
func (r *PasskeyCredentialRepository) Create(ctx context.Context, userID string, cred *models.PasskeyCredential) error {
	credIDHex := bytesToHex(cred.ID)
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO passkey_credentials (id, user_id, public_key, attestation_type, aaguid, sign_count, name, backup_eligible, backup_state, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		credIDHex,
		userID,
		cred.PublicKey,
		cred.AttestationType,
		cred.AAGUID,
		cred.SignCount,
		cred.Name,
		boolToInt(cred.BackupEligible),
		boolToInt(cred.BackupState),
		now,
	)
	return err
}

// GetByUserID retrieves all passkey credentials for a user
func (r *PasskeyCredentialRepository) GetByUserID(ctx context.Context, userID string) ([]models.PasskeyCredential, error) {
	var rows []PasskeyCredentialRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM passkey_credentials WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	return rowsToPasskeyCredentials(rows), nil
}

// GetByCredentialID retrieves a passkey credential by its credential ID
func (r *PasskeyCredentialRepository) GetByCredentialID(ctx context.Context, credID []byte) (*models.PasskeyCredential, string, error) {
	credIDHex := bytesToHex(credID)
	var row PasskeyCredentialRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM passkey_credentials WHERE id = ?", credIDHex)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return rowToPasskeyCredential(&row), row.UserID, nil
}

// UpdateSignCount updates the sign count for a credential
func (r *PasskeyCredentialRepository) UpdateSignCount(ctx context.Context, credID []byte, signCount uint32) error {
	credIDHex := bytesToHex(credID)
	_, err := r.db.ExecContext(ctx, "UPDATE passkey_credentials SET sign_count = ? WHERE id = ?", signCount, credIDHex)
	return err
}

// UpdateLastUsed updates the last used timestamp for a credential
func (r *PasskeyCredentialRepository) UpdateLastUsed(ctx context.Context, credID []byte, lastUsedAt time.Time) error {
	credIDHex := bytesToHex(credID)
	_, err := r.db.ExecContext(ctx, "UPDATE passkey_credentials SET last_used_at = ? WHERE id = ?",
		lastUsedAt.UTC().Format(time.RFC3339), credIDHex)
	return err
}

// Delete deletes a passkey credential
func (r *PasskeyCredentialRepository) Delete(ctx context.Context, userID string, credID []byte) error {
	credIDHex := bytesToHex(credID)
	_, err := r.db.ExecContext(ctx, "DELETE FROM passkey_credentials WHERE user_id = ? AND id = ?", userID, credIDHex)
	return err
}

// DeleteAllForUser deletes all passkey credentials for a user
func (r *PasskeyCredentialRepository) DeleteAllForUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM passkey_credentials WHERE user_id = ?", userID)
	return err
}

func rowToPasskeyCredential(row *PasskeyCredentialRow) *models.PasskeyCredential {
	cred := &models.PasskeyCredential{
		PublicKey:       row.PublicKey,
		AttestationType: row.AttestationType,
		AAGUID:          row.AAGUID,
		SignCount:       uint32(row.SignCount),
		Name:            row.Name,
		BackupEligible:  intToBool(row.BackupEligible),
		BackupState:     intToBool(row.BackupState),
	}

	// Convert hex ID back to bytes
	cred.ID, _ = hexToBytes(row.ID)
	cred.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	if row.LastUsedAt != nil {
		cred.LastUsedAt, _ = time.Parse(time.RFC3339, *row.LastUsedAt)
	}

	return cred
}

func rowsToPasskeyCredentials(rows []PasskeyCredentialRow) []models.PasskeyCredential {
	creds := make([]models.PasskeyCredential, len(rows))
	for i, row := range rows {
		creds[i] = *rowToPasskeyCredential(&row)
	}
	return creds
}
