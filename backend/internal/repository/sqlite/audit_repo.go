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

// AuditLogRow represents an audit log row in SQLite
type AuditLogRow struct {
	ID           string  `db:"id"`
	UserID       string  `db:"user_id"`
	UserEmail    string  `db:"user_email"`
	UserName     string  `db:"user_name"`
	Action       string  `db:"action"`
	ResourceType string  `db:"resource_type"`
	ResourceID   *string `db:"resource_id"`
	Details      *string `db:"details"`
	IPAddress    string  `db:"ip_address"`
	UserAgent    string  `db:"user_agent"`
	Status       string  `db:"status"`
	CreatedAt    string  `db:"created_at"`
}

// AuditLogRepository implements repository.AuditLogRepository for SQLite
type AuditLogRepository struct {
	db *sqlx.DB
}

// NewAuditLogRepository creates a new SQLite audit log repository
func NewAuditLogRepository(db *sqlx.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create creates a new audit log entry
func (r *AuditLogRepository) Create(ctx context.Context, log *models.AuditLog) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	var details *string
	if log.Details != nil {
		detailsJSON, _ := json.Marshal(log.Details)
		d := string(detailsJSON)
		details = &d
	}

	query := `
		INSERT INTO audit_logs (id, user_id, user_email, user_name, action, resource_type, resource_id, details, ip_address, user_agent, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		log.UserID,
		log.UserEmail,
		log.UserName,
		log.Action,
		log.ResourceType,
		log.ResourceID,
		details,
		log.IPAddress,
		log.UserAgent,
		log.Status,
		now,
	)
	return err
}

// List returns audit logs with pagination
func (r *AuditLogRepository) List(ctx context.Context, limit, offset int) ([]models.AuditLog, error) {
	var rows []AuditLogRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}
	return rowsToAuditLogs(rows), nil
}

// ListByUserID returns audit logs by user
func (r *AuditLogRepository) ListByUserID(ctx context.Context, userID string, limit int) ([]models.AuditLog, error) {
	var rows []AuditLogRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM audit_logs WHERE user_id = ? ORDER BY created_at DESC LIMIT ?", userID, limit)
	if err != nil {
		return nil, err
	}
	return rowsToAuditLogs(rows), nil
}

// ListByAction returns audit logs by action
func (r *AuditLogRepository) ListByAction(ctx context.Context, action string, limit int) ([]models.AuditLog, error) {
	var rows []AuditLogRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM audit_logs WHERE action = ? ORDER BY created_at DESC LIMIT ?", action, limit)
	if err != nil {
		return nil, err
	}
	return rowsToAuditLogs(rows), nil
}

// ListByResourceType returns audit logs by resource type
func (r *AuditLogRepository) ListByResourceType(ctx context.Context, resourceType string, limit int) ([]models.AuditLog, error) {
	var rows []AuditLogRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM audit_logs WHERE resource_type = ? ORDER BY created_at DESC LIMIT ?", resourceType, limit)
	if err != nil {
		return nil, err
	}
	return rowsToAuditLogs(rows), nil
}

func rowToAuditLog(row *AuditLogRow) *models.AuditLog {
	log := &models.AuditLog{
		ID:           row.ID,
		UserID:       row.UserID,
		UserEmail:    row.UserEmail,
		UserName:     row.UserName,
		Action:       row.Action,
		ResourceType: row.ResourceType,
		ResourceID:   row.ResourceID,
		IPAddress:    row.IPAddress,
		UserAgent:    row.UserAgent,
		Status:       row.Status,
	}
	log.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)

	if row.Details != nil {
		json.Unmarshal([]byte(*row.Details), &log.Details)
	}

	return log
}

func rowsToAuditLogs(rows []AuditLogRow) []models.AuditLog {
	logs := make([]models.AuditLog, len(rows))
	for i, row := range rows {
		logs[i] = *rowToAuditLog(&row)
	}
	return logs
}

// ApprovalRequestRow represents an approval request row in SQLite
type ApprovalRequestRow struct {
	ID           string  `db:"id"`
	UserID       string  `db:"user_id"`
	UserEmail    string  `db:"user_email"`
	UserName     string  `db:"user_name"`
	Action       string  `db:"action"`
	ResourceType string  `db:"resource_type"`
	ResourceID   *string `db:"resource_id"`
	Details      *string `db:"details"`
	Status       string  `db:"status"`
	ReviewedBy   *string `db:"reviewed_by"`
	ReviewedAt   *string `db:"reviewed_at"`
	ReviewNotes  *string `db:"review_notes"`
	CreatedAt    string  `db:"created_at"`
}

// ApprovalRequestRepository implements repository.ApprovalRequestRepository for SQLite
type ApprovalRequestRepository struct {
	db *sqlx.DB
}

// NewApprovalRequestRepository creates a new SQLite approval request repository
func NewApprovalRequestRepository(db *sqlx.DB) *ApprovalRequestRepository {
	return &ApprovalRequestRepository{db: db}
}

// Create creates a new approval request
func (r *ApprovalRequestRepository) Create(ctx context.Context, request *models.ApprovalRequest) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	var details *string
	if request.Details != nil {
		detailsJSON, _ := json.Marshal(request.Details)
		d := string(detailsJSON)
		details = &d
	}

	query := `
		INSERT INTO approval_requests (id, user_id, user_email, user_name, action, resource_type, resource_id, details, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		request.UserID,
		request.UserEmail,
		request.UserName,
		request.Action,
		request.ResourceType,
		request.ResourceID,
		details,
		request.Status,
		now,
	)
	return err
}

// GetByID retrieves an approval request by ID
func (r *ApprovalRequestRepository) GetByID(ctx context.Context, id string) (*models.ApprovalRequest, error) {
	var row ApprovalRequestRow
	err := r.db.GetContext(ctx, &row, "SELECT * FROM approval_requests WHERE id = ?", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToApprovalRequest(&row), nil
}

// Update updates an existing approval request
func (r *ApprovalRequestRepository) Update(ctx context.Context, request *models.ApprovalRequest) error {
	var reviewedAt *string
	if request.ReviewedAt != nil {
		ra := request.ReviewedAt.UTC().Format(time.RFC3339)
		reviewedAt = &ra
	}

	query := `UPDATE approval_requests SET status = ?, reviewed_by = ?, reviewed_at = ?, review_notes = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, request.Status, request.ReviewedBy, reviewedAt, request.ReviewNotes, request.ID)
	return err
}

// List returns all approval requests
func (r *ApprovalRequestRepository) List(ctx context.Context) ([]models.ApprovalRequest, error) {
	var rows []ApprovalRequestRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM approval_requests ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	return rowsToApprovalRequests(rows), nil
}

// ListPending returns pending approval requests
func (r *ApprovalRequestRepository) ListPending(ctx context.Context) ([]models.ApprovalRequest, error) {
	var rows []ApprovalRequestRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM approval_requests WHERE status = 'pending' ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	return rowsToApprovalRequests(rows), nil
}

// ListByUserID returns approval requests by user
func (r *ApprovalRequestRepository) ListByUserID(ctx context.Context, userID string) ([]models.ApprovalRequest, error) {
	var rows []ApprovalRequestRow
	err := r.db.SelectContext(ctx, &rows, "SELECT * FROM approval_requests WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, err
	}
	return rowsToApprovalRequests(rows), nil
}

func rowToApprovalRequest(row *ApprovalRequestRow) *models.ApprovalRequest {
	request := &models.ApprovalRequest{
		ID:           row.ID,
		UserID:       row.UserID,
		UserEmail:    row.UserEmail,
		UserName:     row.UserName,
		Action:       row.Action,
		ResourceType: row.ResourceType,
		ResourceID:   row.ResourceID,
		Status:       row.Status,
		ReviewedBy:   row.ReviewedBy,
		ReviewNotes:  row.ReviewNotes,
	}
	request.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)

	if row.Details != nil {
		json.Unmarshal([]byte(*row.Details), &request.Details)
	}
	if row.ReviewedAt != nil {
		t, _ := time.Parse(time.RFC3339, *row.ReviewedAt)
		request.ReviewedAt = &t
	}

	return request
}

func rowsToApprovalRequests(rows []ApprovalRequestRow) []models.ApprovalRequest {
	requests := make([]models.ApprovalRequest, len(rows))
	for i, row := range rows {
		requests[i] = *rowToApprovalRequest(&row)
	}
	return requests
}
