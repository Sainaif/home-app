package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// ChoreRow represents a chore row in SQLite
type ChoreRow struct {
	ID                   string  `db:"id"`
	Name                 string  `db:"name"`
	Description          *string `db:"description"`
	Frequency            string  `db:"frequency"`
	CustomInterval       *int    `db:"custom_interval"`
	Difficulty           int     `db:"difficulty"`
	Priority             int     `db:"priority"`
	AssignmentMode       string  `db:"assignment_mode"`
	NotificationsEnabled int     `db:"notifications_enabled"`
	ReminderHours        *int    `db:"reminder_hours"`
	IsActive             int     `db:"is_active"`
	CreatedAt            string  `db:"created_at"`
}

// ChoreRepository implements repository.ChoreRepository for SQLite
type ChoreRepository struct {
	db *sqlx.DB
}

// NewChoreRepository creates a new SQLite chore repository
func NewChoreRepository(db *sqlx.DB) *ChoreRepository {
	return &ChoreRepository{db: db}
}

// Create creates a new chore
func (r *ChoreRepository) Create(ctx context.Context, chore *models.Chore) error {
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO chores (id, name, description, frequency, custom_interval, difficulty, priority,
			assignment_mode, notifications_enabled, reminder_hours, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		chore.ID,
		chore.Name,
		chore.Description,
		chore.Frequency,
		chore.CustomInterval,
		chore.Difficulty,
		chore.Priority,
		chore.AssignmentMode,
		boolToInt(chore.NotificationsEnabled),
		chore.ReminderHours,
		boolToInt(chore.IsActive),
		now,
	)
	return err
}

// GetByID retrieves a chore by ID
func (r *ChoreRepository) GetByID(ctx context.Context, id string) (*models.Chore, error) {
	var row ChoreRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM chores WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToChore(&row), nil
}

// Update updates an existing chore
func (r *ChoreRepository) Update(ctx context.Context, chore *models.Chore) error {
	query := `
		UPDATE chores SET
			name = ?, description = ?, frequency = ?, custom_interval = ?, difficulty = ?, priority = ?,
			assignment_mode = ?, notifications_enabled = ?, reminder_hours = ?, is_active = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		chore.Name,
		chore.Description,
		chore.Frequency,
		chore.CustomInterval,
		chore.Difficulty,
		chore.Priority,
		chore.AssignmentMode,
		boolToInt(chore.NotificationsEnabled),
		chore.ReminderHours,
		boolToInt(chore.IsActive),
		chore.ID,
	)
	return err
}

// Delete deletes a chore
func (r *ChoreRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM chores WHERE id = ?", id)
	return err
}

// List returns all chores
func (r *ChoreRepository) List(ctx context.Context) ([]models.Chore, error) {
	var rows []ChoreRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM chores ORDER BY name")
	if err != nil {
		return nil, err
	}
	return rowsToChores(rows), nil
}

// ListActive returns all active chores
func (r *ChoreRepository) ListActive(ctx context.Context) ([]models.Chore, error) {
	var rows []ChoreRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM chores WHERE is_active = 1 ORDER BY name")
	if err != nil {
		return nil, err
	}
	return rowsToChores(rows), nil
}

func rowToChore(row *ChoreRow) *models.Chore {
	chore := &models.Chore{
		ID:                   row.ID,
		Name:                 row.Name,
		Description:          row.Description,
		Frequency:            row.Frequency,
		CustomInterval:       row.CustomInterval,
		Difficulty:           row.Difficulty,
		Priority:             row.Priority,
		AssignmentMode:       row.AssignmentMode,
		NotificationsEnabled: intToBool(row.NotificationsEnabled),
		ReminderHours:        row.ReminderHours,
		IsActive:             intToBool(row.IsActive),
	}
	chore.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	return chore
}

func rowsToChores(rows []ChoreRow) []models.Chore {
	chores := make([]models.Chore, len(rows))
	for i, row := range rows {
		chores[i] = *rowToChore(&row)
	}
	return chores
}

// ChoreAssignmentRow represents a chore assignment row in SQLite
type ChoreAssignmentRow struct {
	ID             string  `db:"id"`
	ChoreID        string  `db:"chore_id"`
	AssigneeUserID string  `db:"assignee_user_id"`
	DueDate        string  `db:"due_date"`
	Status         string  `db:"status"`
	CompletedAt    *string `db:"completed_at"`
	Points         int     `db:"points"`
	IsOnTime       int     `db:"is_on_time"`
}

// ChoreAssignmentRepository implements repository.ChoreAssignmentRepository for SQLite
type ChoreAssignmentRepository struct {
	db *sqlx.DB
}

// NewChoreAssignmentRepository creates a new SQLite chore assignment repository
func NewChoreAssignmentRepository(db *sqlx.DB) *ChoreAssignmentRepository {
	return &ChoreAssignmentRepository{db: db}
}

// Create creates a new chore assignment
func (r *ChoreAssignmentRepository) Create(ctx context.Context, assignment *models.ChoreAssignment) error {
	var completedAt *string
	if assignment.CompletedAt != nil {
		ca := assignment.CompletedAt.UTC().Format(time.RFC3339)
		completedAt = &ca
	}

	query := `
		INSERT INTO chore_assignments (id, chore_id, assignee_user_id, due_date, status, completed_at, points, is_on_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		assignment.ID,
		assignment.ChoreID,
		assignment.AssigneeUserID,
		assignment.DueDate.UTC().Format(time.RFC3339),
		assignment.Status,
		completedAt,
		assignment.Points,
		boolToInt(assignment.IsOnTime),
	)
	return err
}

// GetByID retrieves a chore assignment by ID
func (r *ChoreAssignmentRepository) GetByID(ctx context.Context, id string) (*models.ChoreAssignment, error) {
	var row ChoreAssignmentRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM chore_assignments WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToChoreAssignment(&row), nil
}

// Update updates an existing chore assignment
func (r *ChoreAssignmentRepository) Update(ctx context.Context, assignment *models.ChoreAssignment) error {
	var completedAt *string
	if assignment.CompletedAt != nil {
		ca := assignment.CompletedAt.UTC().Format(time.RFC3339)
		completedAt = &ca
	}

	query := `
		UPDATE chore_assignments SET
			chore_id = ?, assignee_user_id = ?, due_date = ?, status = ?, completed_at = ?, points = ?, is_on_time = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		assignment.ChoreID,
		assignment.AssigneeUserID,
		assignment.DueDate.UTC().Format(time.RFC3339),
		assignment.Status,
		completedAt,
		assignment.Points,
		boolToInt(assignment.IsOnTime),
		assignment.ID,
	)
	return err
}

// Delete deletes a chore assignment
func (r *ChoreAssignmentRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM chore_assignments WHERE id = ?", id)
	return err
}

// ListByChoreID returns assignments for a chore
func (r *ChoreAssignmentRepository) ListByChoreID(ctx context.Context, choreID string) ([]models.ChoreAssignment, error) {
	var rows []ChoreAssignmentRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM chore_assignments WHERE chore_id = ? ORDER BY due_date DESC", choreID)
	if err != nil {
		return nil, err
	}
	return rowsToChoreAssignments(rows), nil
}

// ListByAssigneeID returns assignments for an assignee
func (r *ChoreAssignmentRepository) ListByAssigneeID(ctx context.Context, assigneeID string) ([]models.ChoreAssignment, error) {
	var rows []ChoreAssignmentRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM chore_assignments WHERE assignee_user_id = ? ORDER BY due_date DESC", assigneeID)
	if err != nil {
		return nil, err
	}
	return rowsToChoreAssignments(rows), nil
}

// ListByStatus returns assignments by status
func (r *ChoreAssignmentRepository) ListByStatus(ctx context.Context, status string) ([]models.ChoreAssignment, error) {
	var rows []ChoreAssignmentRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM chore_assignments WHERE status = ? ORDER BY due_date", status)
	if err != nil {
		return nil, err
	}
	return rowsToChoreAssignments(rows), nil
}

// ListPendingByAssignee returns pending assignments for an assignee
func (r *ChoreAssignmentRepository) ListPendingByAssignee(ctx context.Context, assigneeID string) ([]models.ChoreAssignment, error) {
	var rows []ChoreAssignmentRow
	err := r.db.SelectContext(ctx, &rows,
		"SELECT * FROM chore_assignments WHERE assignee_user_id = ? AND status IN ('pending', 'in_progress') ORDER BY due_date",
		assigneeID)
	if err != nil {
		return nil, err
	}
	return rowsToChoreAssignments(rows), nil
}

// GetLatestByChoreID returns the latest assignment for a chore
func (r *ChoreAssignmentRepository) GetLatestByChoreID(ctx context.Context, choreID string) (*models.ChoreAssignment, error) {
	var row ChoreAssignmentRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM chore_assignments WHERE chore_id = ? ORDER BY due_date DESC LIMIT 1", choreID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToChoreAssignment(&row), nil
}

// List returns all chore assignments
func (r *ChoreAssignmentRepository) List(ctx context.Context) ([]models.ChoreAssignment, error) {
	var rows []ChoreAssignmentRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM chore_assignments ORDER BY due_date DESC")
	if err != nil {
		return nil, err
	}
	return rowsToChoreAssignments(rows), nil
}

// ListFiltered returns chore assignments with optional filters
func (r *ChoreAssignmentRepository) ListFiltered(ctx context.Context, assigneeID, status *string) ([]models.ChoreAssignment, error) {
	query := "SELECT * FROM chore_assignments WHERE 1=1"
	args := []interface{}{}

	if assigneeID != nil {
		query += " AND assignee_user_id = ?"
		args = append(args, *assigneeID)
	}
	if status != nil {
		query += " AND status = ?"
		args = append(args, *status)
	}

	query += " ORDER BY due_date DESC"

	var rows []ChoreAssignmentRow
	err := r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, err
	}
	return rowsToChoreAssignments(rows), nil
}

func rowToChoreAssignment(row *ChoreAssignmentRow) *models.ChoreAssignment {
	assignment := &models.ChoreAssignment{
		ID:             row.ID,
		ChoreID:        row.ChoreID,
		AssigneeUserID: row.AssigneeUserID,
		Status:         row.Status,
		Points:         row.Points,
		IsOnTime:       intToBool(row.IsOnTime),
	}
	assignment.DueDate, _ = time.Parse(time.RFC3339, row.DueDate)
	if row.CompletedAt != nil {
		t, _ := time.Parse(time.RFC3339, *row.CompletedAt)
		assignment.CompletedAt = &t
	}
	return assignment
}

func rowsToChoreAssignments(rows []ChoreAssignmentRow) []models.ChoreAssignment {
	assignments := make([]models.ChoreAssignment, len(rows))
	for i, row := range rows {
		assignments[i] = *rowToChoreAssignment(&row)
	}
	return assignments
}

// ChoreSettingsRow represents chore settings row in SQLite
type ChoreSettingsRow struct {
	ID                    string  `db:"id"`
	DefaultAssignmentMode string  `db:"default_assignment_mode"`
	GlobalNotifications   int     `db:"global_notifications"`
	DefaultReminderHours  int     `db:"default_reminder_hours"`
	PointsEnabled         int     `db:"points_enabled"`
	PointsMultiplier      float64 `db:"points_multiplier"`
	UpdatedAt             string  `db:"updated_at"`
}

// ChoreSettingsRepository implements repository.ChoreSettingsRepository for SQLite
type ChoreSettingsRepository struct {
	db *sqlx.DB
}

// NewChoreSettingsRepository creates a new SQLite chore settings repository
func NewChoreSettingsRepository(db *sqlx.DB) *ChoreSettingsRepository {
	return &ChoreSettingsRepository{db: db}
}

// Get retrieves the chore settings singleton
func (r *ChoreSettingsRepository) Get(ctx context.Context) (*models.ChoreSettings, error) {
	var row ChoreSettingsRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM chore_settings WHERE id = 'singleton'")
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToChoreSettings(&row), nil
}

// Upsert creates or updates the chore settings
func (r *ChoreSettingsRepository) Upsert(ctx context.Context, settings *models.ChoreSettings) error {
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO chore_settings (id, default_assignment_mode, global_notifications, default_reminder_hours,
			points_enabled, points_multiplier, updated_at)
		VALUES ('singleton', ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			default_assignment_mode = excluded.default_assignment_mode,
			global_notifications = excluded.global_notifications,
			default_reminder_hours = excluded.default_reminder_hours,
			points_enabled = excluded.points_enabled,
			points_multiplier = excluded.points_multiplier,
			updated_at = excluded.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		settings.DefaultAssignmentMode,
		boolToInt(settings.GlobalNotifications),
		settings.DefaultReminderHours,
		boolToInt(settings.PointsEnabled),
		settings.PointsMultiplier,
		now,
	)
	return err
}

func rowToChoreSettings(row *ChoreSettingsRow) *models.ChoreSettings {
	settings := &models.ChoreSettings{
		ID:                    row.ID,
		DefaultAssignmentMode: row.DefaultAssignmentMode,
		GlobalNotifications:   intToBool(row.GlobalNotifications),
		DefaultReminderHours:  row.DefaultReminderHours,
		PointsEnabled:         intToBool(row.PointsEnabled),
		PointsMultiplier:      row.PointsMultiplier,
	}
	settings.UpdatedAt, _ = time.Parse(time.RFC3339, row.UpdatedAt)
	return settings
}
