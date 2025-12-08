package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// GroupRow represents a group row in SQLite
type GroupRow struct {
	ID        string  `db:"id"`
	Name      string  `db:"name"`
	Weight    float64 `db:"weight"`
	CreatedAt string  `db:"created_at"`
}

// GroupRepository implements repository.GroupRepository for SQLite
type GroupRepository struct {
	db *sqlx.DB
}

// NewGroupRepository creates a new SQLite group repository
func NewGroupRepository(db *sqlx.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

// Create creates a new group
func (r *GroupRepository) Create(ctx context.Context, group *models.Group) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	query := `INSERT INTO groups (id, name, weight, created_at) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, id, group.Name, group.Weight, now)
	return err
}

// GetByID retrieves a group by ID
func (r *GroupRepository) GetByID(ctx context.Context, id string) (*models.Group, error) {
	var row GroupRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM groups WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToGroup(&row), nil
}

// Update updates an existing group
func (r *GroupRepository) Update(ctx context.Context, group *models.Group) error {
	query := `UPDATE groups SET name = ?, weight = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, group.Name, group.Weight, group.ID)
	return err
}

// Delete deletes a group
func (r *GroupRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM groups WHERE id = ?", id)
	return err
}

// List returns all groups
func (r *GroupRepository) List(ctx context.Context) ([]models.Group, error) {
	var rows []GroupRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM groups ORDER BY name")
	if err != nil {
		return nil, err
	}
	return rowsToGroups(rows), nil
}

func rowToGroup(row *GroupRow) *models.Group {
	group := &models.Group{
		ID:     row.ID,
		Name:   row.Name,
		Weight: row.Weight,
	}
	group.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	return group
}

func rowsToGroups(rows []GroupRow) []models.Group {
	groups := make([]models.Group, len(rows))
	for i, row := range rows {
		groups[i] = *rowToGroup(&row)
	}
	return groups
}
