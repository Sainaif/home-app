package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// SupplySettingsRow represents supply settings row in SQLite
type SupplySettingsRow struct {
	ID                    string `db:"id"`
	WeeklyContributionPLN string `db:"weekly_contribution_pln"`
	ContributionDay       string `db:"contribution_day"`
	CurrentBudgetPLN      string `db:"current_budget_pln"`
	LastContributionAt    string `db:"last_contribution_at"`
	IsActive              int    `db:"is_active"`
	CreatedAt             string `db:"created_at"`
	UpdatedAt             string `db:"updated_at"`
}

// SupplySettingsRepository implements repository.SupplySettingsRepository for SQLite
type SupplySettingsRepository struct {
	db *sqlx.DB
}

// NewSupplySettingsRepository creates a new SQLite supply settings repository
func NewSupplySettingsRepository(db *sqlx.DB) *SupplySettingsRepository {
	return &SupplySettingsRepository{db: db}
}

// Get retrieves the supply settings singleton
func (r *SupplySettingsRepository) Get(ctx context.Context) (*models.SupplySettings, error) {
	var row SupplySettingsRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM supply_settings WHERE id = 'singleton'")
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToSupplySettings(&row), nil
}

// Upsert creates or updates supply settings
func (r *SupplySettingsRepository) Upsert(ctx context.Context, settings *models.SupplySettings) error {
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO supply_settings (id, weekly_contribution_pln, contribution_day, current_budget_pln,
			last_contribution_at, is_active, created_at, updated_at)
		VALUES ('singleton', ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			weekly_contribution_pln = excluded.weekly_contribution_pln,
			contribution_day = excluded.contribution_day,
			current_budget_pln = excluded.current_budget_pln,
			last_contribution_at = excluded.last_contribution_at,
			is_active = excluded.is_active,
			updated_at = excluded.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		settings.WeeklyContributionPLN,
		settings.ContributionDay,
		settings.CurrentBudgetPLN,
		settings.LastContributionAt.UTC().Format(time.RFC3339),
		boolToInt(settings.IsActive),
		now,
		now,
	)
	return err
}

func rowToSupplySettings(row *SupplySettingsRow) *models.SupplySettings {
	settings := &models.SupplySettings{
		ID:                    row.ID,
		WeeklyContributionPLN: row.WeeklyContributionPLN,
		ContributionDay:       row.ContributionDay,
		CurrentBudgetPLN:      row.CurrentBudgetPLN,
		IsActive:              intToBool(row.IsActive),
	}
	settings.LastContributionAt, _ = time.Parse(time.RFC3339, row.LastContributionAt)
	settings.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	settings.UpdatedAt, _ = time.Parse(time.RFC3339, row.UpdatedAt)
	return settings
}

// SupplyItemRow represents a supply item row in SQLite
type SupplyItemRow struct {
	ID                    string  `db:"id"`
	Name                  string  `db:"name"`
	Category              string  `db:"category"`
	CurrentQuantity       int     `db:"current_quantity"`
	MinQuantity           int     `db:"min_quantity"`
	Unit                  string  `db:"unit"`
	Priority              int     `db:"priority"`
	AddedByUserID         string  `db:"added_by_user_id"`
	AddedAt               string  `db:"added_at"`
	LastRestockedAt       *string `db:"last_restocked_at"`
	LastRestockedByUserID *string `db:"last_restocked_by_user_id"`
	LastRestockAmountPLN  *string `db:"last_restock_amount_pln"`
	NeedsRefund           int     `db:"needs_refund"`
	Notes                 *string `db:"notes"`
}

// SupplyItemRepository implements repository.SupplyItemRepository for SQLite
type SupplyItemRepository struct {
	db *sqlx.DB
}

// NewSupplyItemRepository creates a new SQLite supply item repository
func NewSupplyItemRepository(db *sqlx.DB) *SupplyItemRepository {
	return &SupplyItemRepository{db: db}
}

// Create creates a new supply item
func (r *SupplyItemRepository) Create(ctx context.Context, item *models.SupplyItem) error {
	id := uuid.New().String()

	query := `
		INSERT INTO supply_items (id, name, category, current_quantity, min_quantity, unit, priority,
			added_by_user_id, added_at, last_restocked_at, last_restocked_by_user_id, last_restock_amount_pln, needs_refund, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var lastRestockedAt *string
	if item.LastRestockedAt != nil {
		lra := item.LastRestockedAt.UTC().Format(time.RFC3339)
		lastRestockedAt = &lra
	}

	_, err := r.db.ExecContext(ctx, query,
		id,
		item.Name,
		item.Category,
		item.CurrentQuantity,
		item.MinQuantity,
		item.Unit,
		item.Priority,
		item.AddedByUserID,
		item.AddedAt.UTC().Format(time.RFC3339),
		lastRestockedAt,
		item.LastRestockedByUserID,
		item.LastRestockAmountPLN,
		boolToInt(item.NeedsRefund),
		item.Notes,
	)
	return err
}

// GetByID retrieves a supply item by ID
func (r *SupplyItemRepository) GetByID(ctx context.Context, id string) (*models.SupplyItem, error) {
	var row SupplyItemRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM supply_items WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToSupplyItem(&row), nil
}

// Update updates an existing supply item
func (r *SupplyItemRepository) Update(ctx context.Context, item *models.SupplyItem) error {
	var lastRestockedAt *string
	if item.LastRestockedAt != nil {
		lra := item.LastRestockedAt.UTC().Format(time.RFC3339)
		lastRestockedAt = &lra
	}

	query := `
		UPDATE supply_items SET
			name = ?, category = ?, current_quantity = ?, min_quantity = ?, unit = ?, priority = ?,
			last_restocked_at = ?, last_restocked_by_user_id = ?, last_restock_amount_pln = ?, needs_refund = ?, notes = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		item.Name,
		item.Category,
		item.CurrentQuantity,
		item.MinQuantity,
		item.Unit,
		item.Priority,
		lastRestockedAt,
		item.LastRestockedByUserID,
		item.LastRestockAmountPLN,
		boolToInt(item.NeedsRefund),
		item.Notes,
		item.ID,
	)
	return err
}

// Delete deletes a supply item
func (r *SupplyItemRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM supply_items WHERE id = ?", id)
	return err
}

// List returns all supply items
func (r *SupplyItemRepository) List(ctx context.Context) ([]models.SupplyItem, error) {
	var rows []SupplyItemRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM supply_items ORDER BY name")
	if err != nil {
		return nil, err
	}
	return rowsToSupplyItems(rows), nil
}

// ListByCategory returns supply items by category
func (r *SupplyItemRepository) ListByCategory(ctx context.Context, category string) ([]models.SupplyItem, error) {
	var rows []SupplyItemRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM supply_items WHERE category = ? ORDER BY name", category)
	if err != nil {
		return nil, err
	}
	return rowsToSupplyItems(rows), nil
}

// ListLowStock returns items with low stock
func (r *SupplyItemRepository) ListLowStock(ctx context.Context) ([]models.SupplyItem, error) {
	var rows []SupplyItemRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM supply_items WHERE current_quantity <= min_quantity ORDER BY priority DESC, name")
	if err != nil {
		return nil, err
	}
	return rowsToSupplyItems(rows), nil
}

func rowToSupplyItem(row *SupplyItemRow) *models.SupplyItem {
	item := &models.SupplyItem{
		ID:                    row.ID,
		Name:                  row.Name,
		Category:              row.Category,
		CurrentQuantity:       row.CurrentQuantity,
		MinQuantity:           row.MinQuantity,
		Unit:                  row.Unit,
		Priority:              row.Priority,
		AddedByUserID:         row.AddedByUserID,
		LastRestockedByUserID: row.LastRestockedByUserID,
		LastRestockAmountPLN:  row.LastRestockAmountPLN,
		NeedsRefund:           intToBool(row.NeedsRefund),
		Notes:                 row.Notes,
	}
	item.AddedAt, _ = time.Parse(time.RFC3339, row.AddedAt)

	if row.LastRestockedAt != nil {
		t, _ := time.Parse(time.RFC3339, *row.LastRestockedAt)
		item.LastRestockedAt = &t
	}

	return item
}

func rowsToSupplyItems(rows []SupplyItemRow) []models.SupplyItem {
	items := make([]models.SupplyItem, len(rows))
	for i, row := range rows {
		items[i] = *rowToSupplyItem(&row)
	}
	return items
}

// SupplyContributionRow represents a supply contribution row in SQLite
type SupplyContributionRow struct {
	ID          string  `db:"id"`
	UserID      string  `db:"user_id"`
	AmountPLN   string  `db:"amount_pln"`
	PeriodStart string  `db:"period_start"`
	PeriodEnd   string  `db:"period_end"`
	Type        string  `db:"type"`
	Notes       *string `db:"notes"`
	CreatedAt   string  `db:"created_at"`
}

// SupplyContributionRepository implements repository.SupplyContributionRepository for SQLite
type SupplyContributionRepository struct {
	db *sqlx.DB
}

// NewSupplyContributionRepository creates a new SQLite supply contribution repository
func NewSupplyContributionRepository(db *sqlx.DB) *SupplyContributionRepository {
	return &SupplyContributionRepository{db: db}
}

// Create creates a new supply contribution
func (r *SupplyContributionRepository) Create(ctx context.Context, contribution *models.SupplyContribution) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO supply_contributions (id, user_id, amount_pln, period_start, period_end, type, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		contribution.UserID,
		contribution.AmountPLN,
		contribution.PeriodStart.UTC().Format(time.RFC3339),
		contribution.PeriodEnd.UTC().Format(time.RFC3339),
		contribution.Type,
		contribution.Notes,
		now,
	)
	return err
}

// GetByID retrieves a supply contribution by ID
func (r *SupplyContributionRepository) GetByID(ctx context.Context, id string) (*models.SupplyContribution, error) {
	var row SupplyContributionRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM supply_contributions WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToSupplyContribution(&row), nil
}

// Delete deletes a supply contribution
func (r *SupplyContributionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM supply_contributions WHERE id = ?", id)
	return err
}

// List returns all supply contributions
func (r *SupplyContributionRepository) List(ctx context.Context) ([]models.SupplyContribution, error) {
	var rows []SupplyContributionRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM supply_contributions ORDER BY period_start DESC")
	if err != nil {
		return nil, err
	}
	return rowsToSupplyContributions(rows), nil
}

// ListByUserID returns contributions by user
func (r *SupplyContributionRepository) ListByUserID(ctx context.Context, userID string) ([]models.SupplyContribution, error) {
	var rows []SupplyContributionRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM supply_contributions WHERE user_id = ? ORDER BY period_start DESC", userID)
	if err != nil {
		return nil, err
	}
	return rowsToSupplyContributions(rows), nil
}

// ListByPeriod returns contributions within a period
func (r *SupplyContributionRepository) ListByPeriod(ctx context.Context, start, end time.Time) ([]models.SupplyContribution, error) {
	var rows []SupplyContributionRow
	err := r.db.SelectContext(ctx, &rows,
		"SELECT * FROM supply_contributions WHERE period_start >= ? AND period_end <= ? ORDER BY period_start DESC",
		start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return rowsToSupplyContributions(rows), nil
}

// SumByUserID returns total contributions by user
func (r *SupplyContributionRepository) SumByUserID(ctx context.Context, userID string) (string, error) {
	var sum sql.NullString
	err := r.db.GetContext(ctx, &sum, "SELECT COALESCE(SUM(CAST(amount_pln AS REAL)), 0) FROM supply_contributions WHERE user_id = ?", userID)
	if err != nil {
		return "0", err
	}
	if !sum.Valid || sum.String == "" {
		return "0", nil
	}
	return sum.String, nil
}

func rowToSupplyContribution(row *SupplyContributionRow) *models.SupplyContribution {
	contribution := &models.SupplyContribution{
		ID:        row.ID,
		UserID:    row.UserID,
		AmountPLN: row.AmountPLN,
		Type:      row.Type,
		Notes:     row.Notes,
	}
	contribution.PeriodStart, _ = time.Parse(time.RFC3339, row.PeriodStart)
	contribution.PeriodEnd, _ = time.Parse(time.RFC3339, row.PeriodEnd)
	contribution.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	return contribution
}

func rowsToSupplyContributions(rows []SupplyContributionRow) []models.SupplyContribution {
	contributions := make([]models.SupplyContribution, len(rows))
	for i, row := range rows {
		contributions[i] = *rowToSupplyContribution(&row)
	}
	return contributions
}

// SupplyItemHistoryRow represents a supply item history row in SQLite
type SupplyItemHistoryRow struct {
	ID            string  `db:"id"`
	SupplyItemID  string  `db:"supply_item_id"`
	UserID        string  `db:"user_id"`
	Action        string  `db:"action"`
	QuantityDelta int     `db:"quantity_delta"`
	OldQuantity   int     `db:"old_quantity"`
	NewQuantity   int     `db:"new_quantity"`
	CostPLN       *string `db:"cost_pln"`
	Notes         *string `db:"notes"`
	CreatedAt     string  `db:"created_at"`
}

// SupplyItemHistoryRepository implements repository.SupplyItemHistoryRepository for SQLite
type SupplyItemHistoryRepository struct {
	db *sqlx.DB
}

// NewSupplyItemHistoryRepository creates a new SQLite supply item history repository
func NewSupplyItemHistoryRepository(db *sqlx.DB) *SupplyItemHistoryRepository {
	return &SupplyItemHistoryRepository{db: db}
}

// Create creates a new supply item history entry
func (r *SupplyItemHistoryRepository) Create(ctx context.Context, history *models.SupplyItemHistory) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO supply_item_history (id, supply_item_id, user_id, action, quantity_delta, old_quantity, new_quantity, cost_pln, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		history.SupplyItemID,
		history.UserID,
		history.Action,
		history.QuantityDelta,
		history.OldQuantity,
		history.NewQuantity,
		history.CostPLN,
		history.Notes,
		now,
	)
	return err
}

// ListBySupplyItemID returns history for a supply item
func (r *SupplyItemHistoryRepository) ListBySupplyItemID(ctx context.Context, supplyItemID string) ([]models.SupplyItemHistory, error) {
	var rows []SupplyItemHistoryRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM supply_item_history WHERE supply_item_id = ? ORDER BY created_at DESC", supplyItemID)
	if err != nil {
		return nil, err
	}
	return rowsToSupplyItemHistories(rows), nil
}

// ListByUserID returns history by user
func (r *SupplyItemHistoryRepository) ListByUserID(ctx context.Context, userID string) ([]models.SupplyItemHistory, error) {
	var rows []SupplyItemHistoryRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM supply_item_history WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, err
	}
	return rowsToSupplyItemHistories(rows), nil
}

func rowToSupplyItemHistory(row *SupplyItemHistoryRow) *models.SupplyItemHistory {
	history := &models.SupplyItemHistory{
		ID:            row.ID,
		SupplyItemID:  row.SupplyItemID,
		UserID:        row.UserID,
		Action:        row.Action,
		QuantityDelta: row.QuantityDelta,
		OldQuantity:   row.OldQuantity,
		NewQuantity:   row.NewQuantity,
		CostPLN:       row.CostPLN,
		Notes:         row.Notes,
	}
	history.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	return history
}

func rowsToSupplyItemHistories(rows []SupplyItemHistoryRow) []models.SupplyItemHistory {
	histories := make([]models.SupplyItemHistory, len(rows))
	for i, row := range rows {
		histories[i] = *rowToSupplyItemHistory(&row)
	}
	return histories
}
