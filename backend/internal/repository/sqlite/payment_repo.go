package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// PaymentRow represents a payment row in SQLite
type PaymentRow struct {
	ID          string  `db:"id"`
	BillID      string  `db:"bill_id"`
	PayerUserID string  `db:"payer_user_id"`
	AmountPLN   string  `db:"amount_pln"`
	PaidAt      string  `db:"paid_at"`
	Method      *string `db:"method"`
	Reference   *string `db:"reference"`
}

// PaymentRepository implements repository.PaymentRepository for SQLite
type PaymentRepository struct {
	db *sqlx.DB
}

// NewPaymentRepository creates a new SQLite payment repository
func NewPaymentRepository(db *sqlx.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create creates a new payment
func (r *PaymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	id := uuid.New().String()

	query := `
		INSERT INTO payments (id, bill_id, payer_user_id, amount_pln, paid_at, method, reference)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		payment.BillID,
		payment.PayerUserID,
		payment.AmountPLN,
		payment.PaidAt.UTC().Format(time.RFC3339),
		payment.Method,
		payment.Reference,
	)
	return err
}

// GetByID retrieves a payment by ID
func (r *PaymentRepository) GetByID(ctx context.Context, id string) (*models.Payment, error) {
	var row PaymentRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM payments WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToPayment(&row), nil
}

// Delete deletes a payment
func (r *PaymentRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM payments WHERE id = ?", id)
	return err
}

// List returns all payments
func (r *PaymentRepository) List(ctx context.Context) ([]models.Payment, error) {
	var rows []PaymentRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM payments ORDER BY paid_at DESC")
	if err != nil {
		return nil, err
	}
	return rowsToPayments(rows), nil
}

// ListByBillID returns payments for a bill
func (r *PaymentRepository) ListByBillID(ctx context.Context, billID string) ([]models.Payment, error) {
	var rows []PaymentRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM payments WHERE bill_id = ? ORDER BY paid_at DESC", billID)
	if err != nil {
		return nil, err
	}
	return rowsToPayments(rows), nil
}

// ListByPayerID returns payments by a payer
func (r *PaymentRepository) ListByPayerID(ctx context.Context, payerID string) ([]models.Payment, error) {
	var rows []PaymentRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM payments WHERE payer_user_id = ? ORDER BY paid_at DESC", payerID)
	if err != nil {
		return nil, err
	}
	return rowsToPayments(rows), nil
}

// SumByBillID returns the sum of payments for a bill
func (r *PaymentRepository) SumByBillID(ctx context.Context, billID string) (string, error) {
	var sum sql.NullString
	err := r.db.GetContext(ctx, &sum, "SELECT COALESCE(SUM(CAST(amount_pln AS REAL)), 0) FROM payments WHERE bill_id = ?", billID)
	if err != nil {
		return "0", err
	}
	if !sum.Valid || sum.String == "" {
		return "0", nil
	}
	return sum.String, nil
}

func rowToPayment(row *PaymentRow) *models.Payment {
	payment := &models.Payment{
		ID:          row.ID,
		BillID:      row.BillID,
		PayerUserID: row.PayerUserID,
		AmountPLN:   row.AmountPLN,
		Method:      row.Method,
		Reference:   row.Reference,
	}

	payment.PaidAt, _ = time.Parse(time.RFC3339, row.PaidAt)

	return payment
}

func rowsToPayments(rows []PaymentRow) []models.Payment {
	payments := make([]models.Payment, len(rows))
	for i, row := range rows {
		payments[i] = *rowToPayment(&row)
	}
	return payments
}
