package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// RecurringBillTemplateRow represents a recurring bill template row in SQLite
type RecurringBillTemplateRow struct {
	ID              string  `db:"id"`
	CustomType      string  `db:"custom_type"`
	Frequency       string  `db:"frequency"`
	Amount          string  `db:"amount"`
	DayOfMonth      int     `db:"day_of_month"`
	StartDate       string  `db:"start_date"`
	Notes           *string `db:"notes"`
	IsActive        int     `db:"is_active"`
	CurrentBillID   *string `db:"current_bill_id"`
	NextDueDate     string  `db:"next_due_date"`
	LastGeneratedAt *string `db:"last_generated_at"`
	CreatedAt       string  `db:"created_at"`
	UpdatedAt       string  `db:"updated_at"`
}

// RecurringBillTemplateRepository implements repository.RecurringBillTemplateRepository for SQLite
type RecurringBillTemplateRepository struct {
	db *sqlx.DB
}

// NewRecurringBillTemplateRepository creates a new SQLite recurring bill template repository
func NewRecurringBillTemplateRepository(db *sqlx.DB) *RecurringBillTemplateRepository {
	return &RecurringBillTemplateRepository{db: db}
}

// Create creates a new recurring bill template
func (r *RecurringBillTemplateRepository) Create(ctx context.Context, template *models.RecurringBillTemplate) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	var lastGeneratedAt *string
	if template.LastGeneratedAt != nil {
		lga := template.LastGeneratedAt.UTC().Format(time.RFC3339)
		lastGeneratedAt = &lga
	}

	query := `
		INSERT INTO recurring_bill_templates (id, custom_type, frequency, amount, day_of_month, start_date, notes,
			is_active, current_bill_id, next_due_date, last_generated_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		template.CustomType,
		template.Frequency,
		template.Amount,
		template.DayOfMonth,
		template.StartDate.UTC().Format(time.RFC3339),
		template.Notes,
		boolToInt(template.IsActive),
		template.CurrentBillID,
		template.NextDueDate.UTC().Format(time.RFC3339),
		lastGeneratedAt,
		now,
		now,
	)
	return err
}

// GetByID retrieves a recurring bill template by ID
func (r *RecurringBillTemplateRepository) GetByID(ctx context.Context, id string) (*models.RecurringBillTemplate, error) {
	var row RecurringBillTemplateRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM recurring_bill_templates WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToRecurringBillTemplate(&row), nil
}

// Update updates an existing recurring bill template
func (r *RecurringBillTemplateRepository) Update(ctx context.Context, template *models.RecurringBillTemplate) error {
	now := time.Now().UTC().Format(time.RFC3339)

	var lastGeneratedAt *string
	if template.LastGeneratedAt != nil {
		lga := template.LastGeneratedAt.UTC().Format(time.RFC3339)
		lastGeneratedAt = &lga
	}

	query := `
		UPDATE recurring_bill_templates SET
			custom_type = ?, frequency = ?, amount = ?, day_of_month = ?, start_date = ?, notes = ?,
			is_active = ?, current_bill_id = ?, next_due_date = ?, last_generated_at = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		template.CustomType,
		template.Frequency,
		template.Amount,
		template.DayOfMonth,
		template.StartDate.UTC().Format(time.RFC3339),
		template.Notes,
		boolToInt(template.IsActive),
		template.CurrentBillID,
		template.NextDueDate.UTC().Format(time.RFC3339),
		lastGeneratedAt,
		now,
		template.ID,
	)
	return err
}

// Delete deletes a recurring bill template
func (r *RecurringBillTemplateRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM recurring_bill_templates WHERE id = ?", id)
	return err
}

// List returns all recurring bill templates
func (r *RecurringBillTemplateRepository) List(ctx context.Context) ([]models.RecurringBillTemplate, error) {
	var rows []RecurringBillTemplateRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM recurring_bill_templates ORDER BY custom_type")
	if err != nil {
		return nil, err
	}
	return rowsToRecurringBillTemplates(rows), nil
}

// ListActive returns all active recurring bill templates
func (r *RecurringBillTemplateRepository) ListActive(ctx context.Context) ([]models.RecurringBillTemplate, error) {
	var rows []RecurringBillTemplateRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM recurring_bill_templates WHERE is_active = 1 ORDER BY custom_type")
	if err != nil {
		return nil, err
	}
	return rowsToRecurringBillTemplates(rows), nil
}

// ListDueBefore returns templates due before the given date
func (r *RecurringBillTemplateRepository) ListDueBefore(ctx context.Context, date time.Time) ([]models.RecurringBillTemplate, error) {
	var rows []RecurringBillTemplateRow
	err := r.db.SelectContext(ctx, &rows,
		"SELECT * FROM recurring_bill_templates WHERE is_active = 1 AND next_due_date <= ? ORDER BY next_due_date",
		date.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return rowsToRecurringBillTemplates(rows), nil
}

func rowToRecurringBillTemplate(row *RecurringBillTemplateRow) *models.RecurringBillTemplate {
	template := &models.RecurringBillTemplate{
		ID:            row.ID,
		CustomType:    row.CustomType,
		Frequency:     row.Frequency,
		Amount:        row.Amount,
		DayOfMonth:    row.DayOfMonth,
		Notes:         row.Notes,
		IsActive:      intToBool(row.IsActive),
		CurrentBillID: row.CurrentBillID,
	}

	template.StartDate, _ = time.Parse(time.RFC3339, row.StartDate)
	template.NextDueDate, _ = time.Parse(time.RFC3339, row.NextDueDate)
	template.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	template.UpdatedAt, _ = time.Parse(time.RFC3339, row.UpdatedAt)

	if row.LastGeneratedAt != nil {
		t, _ := time.Parse(time.RFC3339, *row.LastGeneratedAt)
		template.LastGeneratedAt = &t
	}

	return template
}

func rowsToRecurringBillTemplates(rows []RecurringBillTemplateRow) []models.RecurringBillTemplate {
	templates := make([]models.RecurringBillTemplate, len(rows))
	for i, row := range rows {
		templates[i] = *rowToRecurringBillTemplate(&row)
	}
	return templates
}

// RecurringBillAllocationRow represents a recurring bill allocation row in SQLite
type RecurringBillAllocationRow struct {
	ID                  string   `db:"id"`
	TemplateID          string   `db:"template_id"`
	SubjectType         string   `db:"subject_type"`
	SubjectID           string   `db:"subject_id"`
	AllocationType      string   `db:"allocation_type"`
	Percentage          *float64 `db:"percentage"`
	FractionNumerator   *int     `db:"fraction_numerator"`
	FractionDenominator *int     `db:"fraction_denominator"`
	FixedAmount         *string  `db:"fixed_amount"`
}

// RecurringBillAllocationRepository implements repository.RecurringBillAllocationRepository for SQLite
type RecurringBillAllocationRepository struct {
	db *sqlx.DB
}

// NewRecurringBillAllocationRepository creates a new SQLite recurring bill allocation repository
func NewRecurringBillAllocationRepository(db *sqlx.DB) *RecurringBillAllocationRepository {
	return &RecurringBillAllocationRepository{db: db}
}

// Create creates a new recurring bill allocation
func (r *RecurringBillAllocationRepository) Create(ctx context.Context, templateID string, alloc *models.RecurringBillAllocation) error {
	id := uuid.New().String()

	query := `
		INSERT INTO recurring_bill_allocations (id, template_id, subject_type, subject_id, allocation_type,
			percentage, fraction_numerator, fraction_denominator, fixed_amount)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		templateID,
		alloc.SubjectType,
		alloc.SubjectID,
		alloc.AllocationType,
		alloc.Percentage,
		alloc.FractionNum,
		alloc.FractionDenom,
		alloc.FixedAmount,
	)
	return err
}

// GetByTemplateID returns allocations for a template
func (r *RecurringBillAllocationRepository) GetByTemplateID(ctx context.Context, templateID string) ([]models.RecurringBillAllocation, error) {
	var rows []RecurringBillAllocationRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM recurring_bill_allocations WHERE template_id = ?", templateID)
	if err != nil {
		return nil, err
	}
	return rowsToRecurringBillAllocations(rows), nil
}

// DeleteByTemplateID deletes all allocations for a template
func (r *RecurringBillAllocationRepository) DeleteByTemplateID(ctx context.Context, templateID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM recurring_bill_allocations WHERE template_id = ?", templateID)
	return err
}

// ReplaceForTemplate replaces all allocations for a template
func (r *RecurringBillAllocationRepository) ReplaceForTemplate(ctx context.Context, templateID string, allocs []models.RecurringBillAllocation) error {
	// Delete existing
	if err := r.DeleteByTemplateID(ctx, templateID); err != nil {
		return err
	}

	// Insert new
	for _, alloc := range allocs {
		if err := r.Create(ctx, templateID, &alloc); err != nil {
			return err
		}
	}

	return nil
}

func rowsToRecurringBillAllocations(rows []RecurringBillAllocationRow) []models.RecurringBillAllocation {
	allocs := make([]models.RecurringBillAllocation, len(rows))
	for i, row := range rows {
		alloc := models.RecurringBillAllocation{
			SubjectType:    row.SubjectType,
			SubjectID:      row.SubjectID,
			AllocationType: row.AllocationType,
			Percentage:     row.Percentage,
			FractionNum:    row.FractionNumerator,
			FractionDenom:  row.FractionDenominator,
			FixedAmount:    row.FixedAmount,
		}
		allocs[i] = alloc
	}
	return allocs
}
