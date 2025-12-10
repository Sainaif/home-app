package sqlite

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/repository"
)

// AllocationRow represents an allocation row in SQLite
type AllocationRow struct {
	ID           string `db:"id"`
	BillID       string `db:"bill_id"`
	SubjectType  string `db:"subject_type"`
	SubjectID    string `db:"subject_id"`
	AllocatedPLN string `db:"allocated_pln"`
}

// AllocationRepository implements repository.AllocationRepository for SQLite
type AllocationRepository struct {
	db *sqlx.DB
}

// NewAllocationRepository creates a new SQLite allocation repository
func NewAllocationRepository(db *sqlx.DB) *AllocationRepository {
	return &AllocationRepository{db: db}
}

// Create creates a new allocation
func (r *AllocationRepository) Create(ctx context.Context, billID, subjectType, subjectID, allocatedPLN string) error {
	id := uuid.New().String()

	query := `INSERT INTO allocations (id, bill_id, subject_type, subject_id, allocated_pln) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, id, billID, subjectType, subjectID, allocatedPLN)
	return err
}

// GetByBillID returns allocations for a bill
func (r *AllocationRepository) GetByBillID(ctx context.Context, billID string) ([]repository.Allocation, error) {
	var rows []AllocationRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM allocations WHERE bill_id = ?", billID)
	if err != nil {
		return nil, err
	}

	allocations := make([]repository.Allocation, len(rows))
	for i, row := range rows {
		allocations[i] = repository.Allocation{
			ID:           row.ID,
			BillID:       row.BillID,
			SubjectType:  row.SubjectType,
			SubjectID:    row.SubjectID,
			AllocatedPLN: row.AllocatedPLN,
		}
	}
	return allocations, nil
}

// DeleteByBillID deletes all allocations for a bill
func (r *AllocationRepository) DeleteByBillID(ctx context.Context, billID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM allocations WHERE bill_id = ?", billID)
	return err
}

// List returns all allocations
func (r *AllocationRepository) List(ctx context.Context) ([]repository.Allocation, error) {
	var rows []AllocationRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM allocations")
	if err != nil {
		return nil, err
	}

	allocations := make([]repository.Allocation, len(rows))
	for i, row := range rows {
		allocations[i] = repository.Allocation{
			ID:           row.ID,
			BillID:       row.BillID,
			SubjectType:  row.SubjectType,
			SubjectID:    row.SubjectID,
			AllocatedPLN: row.AllocatedPLN,
		}
	}
	return allocations, nil
}
