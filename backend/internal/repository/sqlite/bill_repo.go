package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// BillRow represents a bill row in SQLite
type BillRow struct {
	ID                  string  `db:"id"`
	Type                string  `db:"type"`
	CustomType          *string `db:"custom_type"`
	AllocationType      *string `db:"allocation_type"`
	PeriodStart         string  `db:"period_start"`
	PeriodEnd           string  `db:"period_end"`
	PaymentDeadline     *string `db:"payment_deadline"`
	TotalAmountPLN      string  `db:"total_amount_pln"`
	TotalUnits          *string `db:"total_units"`
	Notes               *string `db:"notes"`
	Status              string  `db:"status"`
	ReopenedAt          *string `db:"reopened_at"`
	ReopenReason        *string `db:"reopen_reason"`
	ReopenedBy          *string `db:"reopened_by"`
	RecurringTemplateID *string `db:"recurring_template_id"`
	CreatedAt           string  `db:"created_at"`
}

// BillRepository implements repository.BillRepository for SQLite
type BillRepository struct {
	db *sqlx.DB
}

// NewBillRepository creates a new SQLite bill repository
func NewBillRepository(db *sqlx.DB) *BillRepository {
	return &BillRepository{db: db}
}

// Create creates a new bill
func (r *BillRepository) Create(ctx context.Context, bill *models.Bill) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	var reopenedAt *string
	if bill.ReopenedAt != nil {
		ra := bill.ReopenedAt.UTC().Format(time.RFC3339)
		reopenedAt = &ra
	}

	var paymentDeadline *string
	if bill.PaymentDeadline != nil {
		pd := bill.PaymentDeadline.UTC().Format(time.RFC3339)
		paymentDeadline = &pd
	}

	var totalUnits *string
	if bill.TotalUnits != "" {
		totalUnits = &bill.TotalUnits
	}

	query := `
		INSERT INTO bills (id, type, custom_type, allocation_type, period_start, period_end, payment_deadline,
			total_amount_pln, total_units, notes, status, reopened_at, reopen_reason, reopened_by, recurring_template_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		bill.Type,
		bill.CustomType,
		bill.AllocationType,
		bill.PeriodStart.UTC().Format(time.RFC3339),
		bill.PeriodEnd.UTC().Format(time.RFC3339),
		paymentDeadline,
		bill.TotalAmountPLN,
		totalUnits,
		bill.Notes,
		bill.Status,
		reopenedAt,
		bill.ReopenReason,
		bill.ReopenedBy,
		bill.RecurringTemplateID,
		now,
	)
	return err
}

// GetByID retrieves a bill by ID
func (r *BillRepository) GetByID(ctx context.Context, id string) (*models.Bill, error) {
	var row BillRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM bills WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToBill(&row), nil
}

// Update updates an existing bill
func (r *BillRepository) Update(ctx context.Context, bill *models.Bill) error {
	var reopenedAt *string
	if bill.ReopenedAt != nil {
		ra := bill.ReopenedAt.UTC().Format(time.RFC3339)
		reopenedAt = &ra
	}

	var paymentDeadline *string
	if bill.PaymentDeadline != nil {
		pd := bill.PaymentDeadline.UTC().Format(time.RFC3339)
		paymentDeadline = &pd
	}

	var totalUnits *string
	if bill.TotalUnits != "" {
		totalUnits = &bill.TotalUnits
	}

	query := `
		UPDATE bills SET
			type = ?, custom_type = ?, allocation_type = ?, period_start = ?, period_end = ?, payment_deadline = ?,
			total_amount_pln = ?, total_units = ?, notes = ?, status = ?, reopened_at = ?, reopen_reason = ?,
			reopened_by = ?, recurring_template_id = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		bill.Type,
		bill.CustomType,
		bill.AllocationType,
		bill.PeriodStart.UTC().Format(time.RFC3339),
		bill.PeriodEnd.UTC().Format(time.RFC3339),
		paymentDeadline,
		bill.TotalAmountPLN,
		totalUnits,
		bill.Notes,
		bill.Status,
		reopenedAt,
		bill.ReopenReason,
		bill.ReopenedBy,
		bill.RecurringTemplateID,
		bill.ID,
	)
	return err
}

// Delete deletes a bill
func (r *BillRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM bills WHERE id = ?", id)
	return err
}

// List returns all bills
func (r *BillRepository) List(ctx context.Context) ([]models.Bill, error) {
	var rows []BillRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM bills ORDER BY period_start DESC")
	if err != nil {
		return nil, err
	}
	return rowsToBills(rows), nil
}

// ListByStatus returns bills by status
func (r *BillRepository) ListByStatus(ctx context.Context, status string) ([]models.Bill, error) {
	var rows []BillRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM bills WHERE status = ? ORDER BY period_start DESC", status)
	if err != nil {
		return nil, err
	}
	return rowsToBills(rows), nil
}

// ListByType returns bills by type
func (r *BillRepository) ListByType(ctx context.Context, billType string) ([]models.Bill, error) {
	var rows []BillRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM bills WHERE type = ? ORDER BY period_start DESC", billType)
	if err != nil {
		return nil, err
	}
	return rowsToBills(rows), nil
}

// ListByPeriod returns bills within a period
func (r *BillRepository) ListByPeriod(ctx context.Context, start, end time.Time) ([]models.Bill, error) {
	var rows []BillRow
	err := r.db.SelectContext(ctx, &rows,
		"SELECT * FROM bills WHERE period_start >= ? AND period_end <= ? ORDER BY period_start DESC",
		start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return rowsToBills(rows), nil
}

// GetByRecurringTemplateID retrieves a bill by recurring template ID
func (r *BillRepository) GetByRecurringTemplateID(ctx context.Context, templateID string) (*models.Bill, error) {
	var row BillRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM bills WHERE recurring_template_id = ?", templateID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToBill(&row), nil
}

// ListFiltered returns bills with optional filters
func (r *BillRepository) ListFiltered(ctx context.Context, billType *string, from, to *time.Time) ([]models.Bill, error) {
	query := "SELECT * FROM bills WHERE 1=1"
	args := []interface{}{}

	if billType != nil {
		query += " AND type = ?"
		args = append(args, *billType)
	}
	if from != nil {
		query += " AND period_start >= ?"
		args = append(args, from.UTC().Format(time.RFC3339))
	}
	if to != nil {
		query += " AND period_end <= ?"
		args = append(args, to.UTC().Format(time.RFC3339))
	}

	query += " ORDER BY period_start DESC"

	var rows []BillRow
	err := r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, err
	}
	return rowsToBills(rows), nil
}

func rowToBill(row *BillRow) *models.Bill {
	bill := &models.Bill{
		ID:                  row.ID,
		Type:                row.Type,
		CustomType:          row.CustomType,
		AllocationType:      row.AllocationType,
		TotalAmountPLN:      row.TotalAmountPLN,
		Notes:               row.Notes,
		Status:              row.Status,
		ReopenReason:        row.ReopenReason,
		ReopenedBy:          row.ReopenedBy,
		RecurringTemplateID: row.RecurringTemplateID,
	}

	bill.PeriodStart, _ = time.Parse(time.RFC3339, row.PeriodStart)
	bill.PeriodEnd, _ = time.Parse(time.RFC3339, row.PeriodEnd)
	bill.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)

	if row.TotalUnits != nil {
		bill.TotalUnits = *row.TotalUnits
	}

	if row.PaymentDeadline != nil {
		t, _ := time.Parse(time.RFC3339, *row.PaymentDeadline)
		bill.PaymentDeadline = &t
	}
	if row.ReopenedAt != nil {
		t, _ := time.Parse(time.RFC3339, *row.ReopenedAt)
		bill.ReopenedAt = &t
	}

	return bill
}

func rowsToBills(rows []BillRow) []models.Bill {
	bills := make([]models.Bill, len(rows))
	for i, row := range rows {
		bills[i] = *rowToBill(&row)
	}
	return bills
}
