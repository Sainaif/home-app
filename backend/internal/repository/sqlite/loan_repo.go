package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// LoanRow represents a loan row in SQLite
type LoanRow struct {
	ID         string  `db:"id"`
	LenderID   string  `db:"lender_id"`
	BorrowerID string  `db:"borrower_id"`
	AmountPLN  string  `db:"amount_pln"`
	Note       *string `db:"note"`
	DueDate    *string `db:"due_date"`
	Status     string  `db:"status"`
	CreatedAt  string  `db:"created_at"`
}

// LoanRepository implements repository.LoanRepository for SQLite
type LoanRepository struct {
	db *sqlx.DB
}

// NewLoanRepository creates a new SQLite loan repository
func NewLoanRepository(db *sqlx.DB) *LoanRepository {
	return &LoanRepository{db: db}
}

// Create creates a new loan
func (r *LoanRepository) Create(ctx context.Context, loan *models.Loan) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	var dueDate *string
	if loan.DueDate != nil {
		dd := loan.DueDate.UTC().Format(time.RFC3339)
		dueDate = &dd
	}

	query := `
		INSERT INTO loans (id, lender_id, borrower_id, amount_pln, note, due_date, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		loan.LenderID,
		loan.BorrowerID,
		loan.AmountPLN,
		loan.Note,
		dueDate,
		loan.Status,
		now,
	)
	return err
}

// GetByID retrieves a loan by ID
func (r *LoanRepository) GetByID(ctx context.Context, id string) (*models.Loan, error) {
	var row LoanRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM loans WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToLoan(&row), nil
}

// Update updates an existing loan
func (r *LoanRepository) Update(ctx context.Context, loan *models.Loan) error {
	var dueDate *string
	if loan.DueDate != nil {
		dd := loan.DueDate.UTC().Format(time.RFC3339)
		dueDate = &dd
	}

	query := `
		UPDATE loans SET
			lender_id = ?, borrower_id = ?, amount_pln = ?, note = ?, due_date = ?, status = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		loan.LenderID,
		loan.BorrowerID,
		loan.AmountPLN,
		loan.Note,
		dueDate,
		loan.Status,
		loan.ID,
	)
	return err
}

// Delete deletes a loan
func (r *LoanRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM loans WHERE id = ?", id)
	return err
}

// List returns all loans
func (r *LoanRepository) List(ctx context.Context) ([]models.Loan, error) {
	var rows []LoanRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM loans ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	return rowsToLoans(rows), nil
}

// ListByLenderID returns loans by lender
func (r *LoanRepository) ListByLenderID(ctx context.Context, lenderID string) ([]models.Loan, error) {
	var rows []LoanRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM loans WHERE lender_id = ? ORDER BY created_at DESC", lenderID)
	if err != nil {
		return nil, err
	}
	return rowsToLoans(rows), nil
}

// ListByBorrowerID returns loans by borrower
func (r *LoanRepository) ListByBorrowerID(ctx context.Context, borrowerID string) ([]models.Loan, error) {
	var rows []LoanRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM loans WHERE borrower_id = ? ORDER BY created_at DESC", borrowerID)
	if err != nil {
		return nil, err
	}
	return rowsToLoans(rows), nil
}

// ListByStatus returns loans by status
func (r *LoanRepository) ListByStatus(ctx context.Context, status string) ([]models.Loan, error) {
	var rows []LoanRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM loans WHERE status = ? ORDER BY created_at DESC", status)
	if err != nil {
		return nil, err
	}
	return rowsToLoans(rows), nil
}

// ListOpenBetweenUsers returns open/partial loans where userA is the lender and userB is the borrower
func (r *LoanRepository) ListOpenBetweenUsers(ctx context.Context, lenderID, borrowerID string) ([]models.Loan, error) {
	var rows []LoanRow
	query := `
		SELECT * FROM loans
		WHERE (status = 'open' OR status = 'partial')
		AND lender_id = ? AND borrower_id = ?
		ORDER BY created_at DESC
	`
	err := r.db.SelectContext(ctx, &rows, query, lenderID, borrowerID)
	if err != nil {
		return nil, err
	}
	return rowsToLoans(rows), nil
}

func rowToLoan(row *LoanRow) *models.Loan {
	loan := &models.Loan{
		ID:         row.ID,
		LenderID:   row.LenderID,
		BorrowerID: row.BorrowerID,
		AmountPLN:  row.AmountPLN,
		Note:       row.Note,
		Status:     row.Status,
	}

	loan.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)

	if row.DueDate != nil {
		t, _ := time.Parse(time.RFC3339, *row.DueDate)
		loan.DueDate = &t
	}

	return loan
}

func rowsToLoans(rows []LoanRow) []models.Loan {
	loans := make([]models.Loan, len(rows))
	for i, row := range rows {
		loans[i] = *rowToLoan(&row)
	}
	return loans
}

// LoanPaymentRow represents a loan payment row in SQLite
type LoanPaymentRow struct {
	ID        string  `db:"id"`
	LoanID    string  `db:"loan_id"`
	AmountPLN string  `db:"amount_pln"`
	PaidAt    string  `db:"paid_at"`
	Note      *string `db:"note"`
}

// LoanPaymentRepository implements repository.LoanPaymentRepository for SQLite
type LoanPaymentRepository struct {
	db *sqlx.DB
}

// NewLoanPaymentRepository creates a new SQLite loan payment repository
func NewLoanPaymentRepository(db *sqlx.DB) *LoanPaymentRepository {
	return &LoanPaymentRepository{db: db}
}

// Create creates a new loan payment
func (r *LoanPaymentRepository) Create(ctx context.Context, payment *models.LoanPayment) error {
	id := uuid.New().String()

	query := `INSERT INTO loan_payments (id, loan_id, amount_pln, paid_at, note) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		id,
		payment.LoanID,
		payment.AmountPLN,
		payment.PaidAt.UTC().Format(time.RFC3339),
		payment.Note,
	)
	return err
}

// GetByID retrieves a loan payment by ID
func (r *LoanPaymentRepository) GetByID(ctx context.Context, id string) (*models.LoanPayment, error) {
	var row LoanPaymentRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM loan_payments WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToLoanPayment(&row), nil
}

// Delete deletes a loan payment
func (r *LoanPaymentRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM loan_payments WHERE id = ?", id)
	return err
}

// List returns all loan payments
func (r *LoanPaymentRepository) List(ctx context.Context) ([]models.LoanPayment, error) {
	var rows []LoanPaymentRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM loan_payments ORDER BY paid_at DESC")
	if err != nil {
		return nil, err
	}
	return rowsToLoanPayments(rows), nil
}

// ListByLoanID returns payments for a loan
func (r *LoanPaymentRepository) ListByLoanID(ctx context.Context, loanID string) ([]models.LoanPayment, error) {
	var rows []LoanPaymentRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM loan_payments WHERE loan_id = ? ORDER BY paid_at DESC", loanID)
	if err != nil {
		return nil, err
	}
	return rowsToLoanPayments(rows), nil
}

// SumByLoanID returns the sum of payments for a loan
func (r *LoanPaymentRepository) SumByLoanID(ctx context.Context, loanID string) (string, error) {
	var sum sql.NullString
	err := r.db.GetContext(ctx, &sum, "SELECT COALESCE(SUM(CAST(amount_pln AS REAL)), 0) FROM loan_payments WHERE loan_id = ?", loanID)
	if err != nil {
		return "0", err
	}
	if !sum.Valid || sum.String == "" {
		return "0", nil
	}
	return sum.String, nil
}

func rowToLoanPayment(row *LoanPaymentRow) *models.LoanPayment {
	payment := &models.LoanPayment{
		ID:        row.ID,
		LoanID:    row.LoanID,
		AmountPLN: row.AmountPLN,
		Note:      row.Note,
	}

	payment.PaidAt, _ = time.Parse(time.RFC3339, row.PaidAt)

	return payment
}

func rowsToLoanPayments(rows []LoanPaymentRow) []models.LoanPayment {
	payments := make([]models.LoanPayment, len(rows))
	for i, row := range rows {
		payments[i] = *rowToLoanPayment(&row)
	}
	return payments
}
