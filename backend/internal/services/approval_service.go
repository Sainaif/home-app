package services

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/sainaif/holy-home/internal/models"
)

type ApprovalService struct {
	db *mongo.Database
}

func NewApprovalService(db *mongo.Database) *ApprovalService {
	return &ApprovalService{db: db}
}

// CreateApprovalRequest creates a new approval request
func (s *ApprovalService) CreateApprovalRequest(
	ctx context.Context,
	userID primitive.ObjectID,
	userEmail, userName string,
	action, resourceType string,
	resourceID *primitive.ObjectID,
	details map[string]interface{},
) (*models.ApprovalRequest, error) {
	request := models.ApprovalRequest{
		ID:           primitive.NewObjectID(),
		UserID:       userID,
		UserEmail:    userEmail,
		UserName:     userName,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}

	_, err := s.db.Collection("approval_requests").InsertOne(ctx, request)
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// GetPendingRequests retrieves all pending approval requests
func (s *ApprovalService) GetPendingRequests(ctx context.Context) ([]models.ApprovalRequest, error) {
	cursor, err := s.db.Collection("approval_requests").Find(
		ctx,
		bson.M{"status": "pending"},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var requests []models.ApprovalRequest
	if err := cursor.All(ctx, &requests); err != nil {
		return nil, err
	}
	return requests, nil
}

// GetAllRequests retrieves all approval requests with pagination
func (s *ApprovalService) GetAllRequests(ctx context.Context, limit, skip int, filter bson.M) ([]models.ApprovalRequest, int64, error) {
	total, err := s.db.Collection("approval_requests").CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(skip)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := s.db.Collection("approval_requests").Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var requests []models.ApprovalRequest
	if err := cursor.All(ctx, &requests); err != nil {
		return nil, 0, err
	}

	return requests, total, nil
}

// ApproveRequest approves an approval request
func (s *ApprovalService) ApproveRequest(ctx context.Context, requestID, reviewerID primitive.ObjectID, notes *string) error {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":       "approved",
			"reviewed_by":  reviewerID,
			"reviewed_at":  now,
			"review_notes": notes,
		},
	}

	result, err := s.db.Collection("approval_requests").UpdateOne(
		ctx,
		bson.M{"_id": requestID, "status": "pending"},
		update,
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("request not found or already processed")
	}
	return nil
}

// RejectRequest rejects an approval request
func (s *ApprovalService) RejectRequest(ctx context.Context, requestID, reviewerID primitive.ObjectID, notes *string) error {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":       "rejected",
			"reviewed_by":  reviewerID,
			"reviewed_at":  now,
			"review_notes": notes,
		},
	}

	result, err := s.db.Collection("approval_requests").UpdateOne(
		ctx,
		bson.M{"_id": requestID, "status": "pending"},
		update,
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("request not found or already processed")
	}
	return nil
}

// GetRequest retrieves a specific approval request
func (s *ApprovalService) GetRequest(ctx context.Context, requestID primitive.ObjectID) (*models.ApprovalRequest, error) {
	var request models.ApprovalRequest
	err := s.db.Collection("approval_requests").FindOne(ctx, bson.M{"_id": requestID}).Decode(&request)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("request not found")
		}
		return nil, err
	}
	return &request, nil
}
