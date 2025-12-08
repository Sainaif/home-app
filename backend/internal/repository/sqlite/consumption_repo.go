package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
)

// ConsumptionRow represents a consumption row in SQLite
type ConsumptionRow struct {
	ID          string  `db:"id"`
	BillID      string  `db:"bill_id"`
	SubjectType string  `db:"subject_type"`
	SubjectID   string  `db:"subject_id"`
	Units       string  `db:"units"`
	MeterValue  *string `db:"meter_value"`
	RecordedAt  string  `db:"recorded_at"`
	Source      string  `db:"source"`
}

// ConsumptionRepository implements repository.ConsumptionRepository for SQLite
type ConsumptionRepository struct {
	db *sqlx.DB
}

// NewConsumptionRepository creates a new SQLite consumption repository
func NewConsumptionRepository(db *sqlx.DB) *ConsumptionRepository {
	return &ConsumptionRepository{db: db}
}

// Create creates a new consumption
func (r *ConsumptionRepository) Create(ctx context.Context, consumption *models.Consumption) error {
	id := uuid.New().String()

	query := `
		INSERT INTO consumptions (id, bill_id, subject_type, subject_id, units, meter_value, recorded_at, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		consumption.BillID,
		consumption.SubjectType,
		consumption.SubjectID,
		consumption.Units,
		consumption.MeterValue,
		consumption.RecordedAt.UTC().Format(time.RFC3339),
		consumption.Source,
	)
	return err
}

// GetByID retrieves a consumption by ID
func (r *ConsumptionRepository) GetByID(ctx context.Context, id string) (*models.Consumption, error) {
	var row ConsumptionRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM consumptions WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToConsumption(&row), nil
}

// Update updates an existing consumption
func (r *ConsumptionRepository) Update(ctx context.Context, consumption *models.Consumption) error {
	query := `
		UPDATE consumptions SET
			bill_id = ?, subject_type = ?, subject_id = ?, units = ?, meter_value = ?, recorded_at = ?, source = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		consumption.BillID,
		consumption.SubjectType,
		consumption.SubjectID,
		consumption.Units,
		consumption.MeterValue,
		consumption.RecordedAt.UTC().Format(time.RFC3339),
		consumption.Source,
		consumption.ID,
	)
	return err
}

// Delete deletes a consumption
func (r *ConsumptionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM consumptions WHERE id = ?", id)
	return err
}

// List returns all consumptions
func (r *ConsumptionRepository) List(ctx context.Context) ([]models.Consumption, error) {
	var rows []ConsumptionRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM consumptions ORDER BY recorded_at DESC")
	if err != nil {
		return nil, err
	}
	return rowsToConsumptions(rows), nil
}

// ListByBillID returns consumptions for a bill
func (r *ConsumptionRepository) ListByBillID(ctx context.Context, billID string) ([]models.Consumption, error) {
	var rows []ConsumptionRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM consumptions WHERE bill_id = ?", billID)
	if err != nil {
		return nil, err
	}
	return rowsToConsumptions(rows), nil
}

// ListBySubject returns consumptions for a subject
func (r *ConsumptionRepository) ListBySubject(ctx context.Context, subjectType, subjectID string) ([]models.Consumption, error) {
	var rows []ConsumptionRow
	err := r.db.SelectContext(ctx, &rows,
		"SELECT * FROM consumptions WHERE subject_type = ? AND subject_id = ? ORDER BY recorded_at DESC",
		subjectType, subjectID)
	if err != nil {
		return nil, err
	}
	return rowsToConsumptions(rows), nil
}

// ListFiltered returns consumptions with optional filters
func (r *ConsumptionRepository) ListFiltered(ctx context.Context, subjectType, subjectID *string, from, to *time.Time) ([]models.Consumption, error) {
	query := "SELECT * FROM consumptions WHERE 1=1"
	args := []interface{}{}

	if subjectType != nil {
		query += " AND subject_type = ?"
		args = append(args, *subjectType)
	}
	if subjectID != nil {
		query += " AND subject_id = ?"
		args = append(args, *subjectID)
	}
	if from != nil {
		query += " AND recorded_at >= ?"
		args = append(args, from.UTC().Format(time.RFC3339))
	}
	if to != nil {
		query += " AND recorded_at <= ?"
		args = append(args, to.UTC().Format(time.RFC3339))
	}

	query += " ORDER BY recorded_at DESC"

	var rows []ConsumptionRow
	err := r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, err
	}
	return rowsToConsumptions(rows), nil
}

// DeleteByBillID deletes all consumptions for a bill
func (r *ConsumptionRepository) DeleteByBillID(ctx context.Context, billID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM consumptions WHERE bill_id = ?", billID)
	return err
}

func rowToConsumption(row *ConsumptionRow) *models.Consumption {
	consumption := &models.Consumption{
		ID:          row.ID,
		BillID:      row.BillID,
		SubjectType: row.SubjectType,
		SubjectID:   row.SubjectID,
		Units:       row.Units,
		MeterValue:  row.MeterValue,
		Source:      row.Source,
	}

	consumption.RecordedAt, _ = time.Parse(time.RFC3339, row.RecordedAt)

	return consumption
}

func rowsToConsumptions(rows []ConsumptionRow) []models.Consumption {
	consumptions := make([]models.Consumption, len(rows))
	for i, row := range rows {
		consumptions[i] = *rowToConsumption(&row)
	}
	return consumptions
}
