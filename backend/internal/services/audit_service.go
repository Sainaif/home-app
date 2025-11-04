package services

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/sainaif/holy-home/internal/models"
)

type AuditService struct {
	db *mongo.Database
}

func NewAuditService(db *mongo.Database) *AuditService {
	return &AuditService{db: db}
}

// LogAction creates an audit log entry
func (s *AuditService) LogAction(
	ctx context.Context,
	userID primitive.ObjectID,
	userEmail, userName string,
	action, resourceType string,
	resourceID *primitive.ObjectID,
	details map[string]interface{},
	ipAddress, userAgent string,
	status string,
) error {
	log := models.AuditLog{
		ID:           primitive.NewObjectID(),
		UserID:       userID,
		UserEmail:    userEmail,
		UserName:     userName,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       status,
		CreatedAt:    time.Now(),
	}

	_, err := s.db.Collection("audit_logs").InsertOne(ctx, log)
	return err
}

// GetLogs retrieves audit logs with pagination and filtering
func (s *AuditService) GetLogs(ctx context.Context, limit, skip int, filter bson.M) ([]models.AuditLog, int64, error) {
	// Get total count
	total, err := s.db.Collection("audit_logs").CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get logs with pagination
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(skip)).
		SetSort(bson.D{{Key: "created_at", Value: -1}}) // Most recent first

	cursor, err := s.db.Collection("audit_logs").Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var logs []models.AuditLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
