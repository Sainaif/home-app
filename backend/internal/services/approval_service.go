package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
)

type ApprovalService struct {
	approvalRequests repository.ApprovalRequestRepository
}

func NewApprovalService(approvalRequests repository.ApprovalRequestRepository) *ApprovalService {
	return &ApprovalService{approvalRequests: approvalRequests}
}

// CreateApprovalRequest creates a new approval request
func (s *ApprovalService) CreateApprovalRequest(
	ctx context.Context,
	userID string,
	userEmail, userName string,
	action, resourceType string,
	resourceID *string,
	details map[string]interface{},
) (*models.ApprovalRequest, error) {
	// Convert details map to JSON string
	var detailsJSON string
	if details != nil {
		detailsBytes, _ := json.Marshal(details)
		detailsJSON = string(detailsBytes)
	}

	request := &models.ApprovalRequest{
		ID:           uuid.New().String(),
		UserID:       userID,
		UserEmail:    userEmail,
		UserName:     userName,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		DetailsJSON:  detailsJSON,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}

	if err := s.approvalRequests.Create(ctx, request); err != nil {
		return nil, err
	}
	return request, nil
}

// GetPendingRequests retrieves all pending approval requests
func (s *ApprovalService) GetPendingRequests(ctx context.Context) ([]models.ApprovalRequest, error) {
	return s.approvalRequests.ListPending(ctx)
}

// GetAllRequests retrieves all approval requests
func (s *ApprovalService) GetAllRequests(ctx context.Context) ([]models.ApprovalRequest, error) {
	return s.approvalRequests.List(ctx)
}

// ApproveRequest approves an approval request
func (s *ApprovalService) ApproveRequest(ctx context.Context, requestID, reviewerID string, notes *string) error {
	request, err := s.approvalRequests.GetByID(ctx, requestID)
	if err != nil {
		return errors.New("request not found")
	}

	if request.Status != "pending" {
		return errors.New("request not found or already processed")
	}

	now := time.Now()
	request.Status = "approved"
	request.ReviewedBy = &reviewerID
	request.ReviewedAt = &now
	request.ReviewNotes = notes

	return s.approvalRequests.Update(ctx, request)
}

// RejectRequest rejects an approval request
func (s *ApprovalService) RejectRequest(ctx context.Context, requestID, reviewerID string, notes *string) error {
	request, err := s.approvalRequests.GetByID(ctx, requestID)
	if err != nil {
		return errors.New("request not found")
	}

	if request.Status != "pending" {
		return errors.New("request not found or already processed")
	}

	now := time.Now()
	request.Status = "rejected"
	request.ReviewedBy = &reviewerID
	request.ReviewedAt = &now
	request.ReviewNotes = notes

	return s.approvalRequests.Update(ctx, request)
}

// GetRequest retrieves a specific approval request
func (s *ApprovalService) GetRequest(ctx context.Context, requestID string) (*models.ApprovalRequest, error) {
	request, err := s.approvalRequests.GetByID(ctx, requestID)
	if err != nil {
		return nil, errors.New("request not found")
	}
	return request, nil
}
