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

// PermissionRow represents a permission row in SQLite
type PermissionRow struct {
	ID          string `db:"id"`
	Name        string `db:"name"`
	Description string `db:"description"`
	Category    string `db:"category"`
}

// PermissionRepository implements repository.PermissionRepository for SQLite
type PermissionRepository struct {
	db *sqlx.DB
}

// NewPermissionRepository creates a new SQLite permission repository
func NewPermissionRepository(db *sqlx.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

// Create creates a new permission
func (r *PermissionRepository) Create(ctx context.Context, permission *models.Permission) error {
	id := uuid.New().String()

	query := `INSERT INTO permissions (id, name, description, category) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, id, permission.Name, permission.Description, permission.Category)
	return err
}

// GetByName retrieves a permission by name
func (r *PermissionRepository) GetByName(ctx context.Context, name string) (*models.Permission, error) {
	var row PermissionRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM permissions WHERE name = ?", name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToPermission(&row), nil
}

// List returns all permissions
func (r *PermissionRepository) List(ctx context.Context) ([]models.Permission, error) {
	var rows []PermissionRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM permissions ORDER BY category, name")
	if err != nil {
		return nil, err
	}
	return rowsToPermissions(rows), nil
}

// ListByCategory returns permissions by category
func (r *PermissionRepository) ListByCategory(ctx context.Context, category string) ([]models.Permission, error) {
	var rows []PermissionRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM permissions WHERE category = ? ORDER BY name", category)
	if err != nil {
		return nil, err
	}
	return rowsToPermissions(rows), nil
}

func rowToPermission(row *PermissionRow) *models.Permission {
	return &models.Permission{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Category:    row.Category,
	}
}

func rowsToPermissions(rows []PermissionRow) []models.Permission {
	permissions := make([]models.Permission, len(rows))
	for i, row := range rows {
		permissions[i] = *rowToPermission(&row)
	}
	return permissions
}

// RoleRow represents a role row in SQLite
type RoleRow struct {
	ID          string `db:"id"`
	Name        string `db:"name"`
	DisplayName string `db:"display_name"`
	IsSystem    int    `db:"is_system"`
	Permissions string `db:"permissions"`
	CreatedAt   string `db:"created_at"`
	UpdatedAt   string `db:"updated_at"`
}

// RoleRepository implements repository.RoleRepository for SQLite
type RoleRepository struct {
	db *sqlx.DB
}

// NewRoleRepository creates a new SQLite role repository
func NewRoleRepository(db *sqlx.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// Create creates a new role
func (r *RoleRepository) Create(ctx context.Context, role *models.Role) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	permsJSON, _ := json.Marshal(role.Permissions)

	query := `INSERT INTO roles (id, name, display_name, is_system, permissions, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, id, role.Name, role.DisplayName, boolToInt(role.IsSystem), string(permsJSON), now, now)
	return err
}

// GetByID retrieves a role by ID
func (r *RoleRepository) GetByID(ctx context.Context, id string) (*models.Role, error) {
	var row RoleRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM roles WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToRole(&row), nil
}

// GetByName retrieves a role by name
func (r *RoleRepository) GetByName(ctx context.Context, name string) (*models.Role, error) {
	var row RoleRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM roles WHERE name = ?", name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToRole(&row), nil
}

// Update updates an existing role
func (r *RoleRepository) Update(ctx context.Context, role *models.Role) error {
	now := time.Now().UTC().Format(time.RFC3339)
	permsJSON, _ := json.Marshal(role.Permissions)

	query := `UPDATE roles SET name = ?, display_name = ?, is_system = ?, permissions = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, role.Name, role.DisplayName, boolToInt(role.IsSystem), string(permsJSON), now, role.ID)
	return err
}

// Delete deletes a role
func (r *RoleRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM roles WHERE id = ?", id)
	return err
}

// List returns all roles
func (r *RoleRepository) List(ctx context.Context) ([]models.Role, error) {
	var rows []RoleRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM roles ORDER BY name")
	if err != nil {
		return nil, err
	}
	return rowsToRoles(rows), nil
}

func rowToRole(row *RoleRow) *models.Role {
	role := &models.Role{
		ID:          row.ID,
		Name:        row.Name,
		DisplayName: row.DisplayName,
		IsSystem:    intToBool(row.IsSystem),
	}
	role.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	role.UpdatedAt, _ = time.Parse(time.RFC3339, row.UpdatedAt)
	json.Unmarshal([]byte(row.Permissions), &role.Permissions)
	return role
}

func rowsToRoles(rows []RoleRow) []models.Role {
	roles := make([]models.Role, len(rows))
	for i, row := range rows {
		roles[i] = *rowToRole(&row)
	}
	return roles
}
