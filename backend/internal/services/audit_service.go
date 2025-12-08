package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
)

type AuditService struct {
	auditLogs repository.AuditLogRepository
}

func NewAuditService(auditLogs repository.AuditLogRepository) *AuditService {
	return &AuditService{auditLogs: auditLogs}
}

// LogAction creates an audit log entry
func (s *AuditService) LogAction(
	ctx context.Context,
	userID string,
	userEmail, userName string,
	action, resourceType string,
	resourceID *string,
	details map[string]interface{},
	ipAddress, userAgent string,
	status string,
) error {
	// Convert details map to JSON string
	var detailsJSON string
	if details != nil {
		detailsBytes, _ := json.Marshal(details)
		detailsJSON = string(detailsBytes)
	}

	log := &models.AuditLog{
		ID:           uuid.New().String(),
		UserID:       userID,
		UserEmail:    userEmail,
		UserName:     userName,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		DetailsJSON:  detailsJSON,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       status,
		CreatedAt:    time.Now(),
	}

	return s.auditLogs.Create(ctx, log)
}

// GetLogs retrieves audit logs with pagination
func (s *AuditService) GetLogs(ctx context.Context, limit, offset int) ([]models.AuditLog, error) {
	return s.auditLogs.List(ctx, limit, offset)
}

// GetLogsByUserID retrieves audit logs for a specific user
func (s *AuditService) GetLogsByUserID(ctx context.Context, userID string, limit int) ([]models.AuditLog, error) {
	return s.auditLogs.ListByUserID(ctx, userID, limit)
}

// GetLogsByAction retrieves audit logs for a specific action
func (s *AuditService) GetLogsByAction(ctx context.Context, action string, limit int) ([]models.AuditLog, error) {
	return s.auditLogs.ListByAction(ctx, action, limit)
}

// GetLogsByResourceType retrieves audit logs for a specific resource type
func (s *AuditService) GetLogsByResourceType(ctx context.Context, resourceType string, limit int) ([]models.AuditLog, error) {
	return s.auditLogs.ListByResourceType(ctx, resourceType, limit)
}
