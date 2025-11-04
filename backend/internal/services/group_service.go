package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/sainaif/holy-home/internal/models"
)

type GroupService struct {
	db *mongo.Database
}

func NewGroupService(db *mongo.Database) *GroupService {
	return &GroupService{db: db}
}

type CreateGroupRequest struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

type UpdateGroupRequest struct {
	Name   *string  `json:"name,omitempty"`
	Weight *float64 `json:"weight,omitempty"`
}

// CreateGroup creates a new household group (ADMIN only)
func (s *GroupService) CreateGroup(ctx context.Context, req CreateGroupRequest) (*models.Group, error) {
	if req.Name == "" {
		return nil, errors.New("group name is required")
	}

	weight := req.Weight
	if weight == 0 {
		weight = 1.0 // Default weight
	}

	group := models.Group{
		ID:        primitive.NewObjectID(),
		Name:      req.Name,
		Weight:    weight,
		CreatedAt: time.Now(),
	}

	_, err := s.db.Collection("groups").InsertOne(ctx, group)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	return &group, nil
}

// GetGroups retrieves all groups
func (s *GroupService) GetGroups(ctx context.Context) ([]models.Group, error) {
	cursor, err := s.db.Collection("groups").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var groups []models.Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, fmt.Errorf("failed to decode groups: %w", err)
	}

	return groups, nil
}

// GetGroup retrieves a group by ID
func (s *GroupService) GetGroup(ctx context.Context, groupID primitive.ObjectID) (*models.Group, error) {
	var group models.Group
	err := s.db.Collection("groups").FindOne(ctx, bson.M{"_id": groupID}).Decode(&group)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("group not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &group, nil
}

// UpdateGroup updates a group (ADMIN only)
func (s *GroupService) UpdateGroup(ctx context.Context, groupID primitive.ObjectID, req UpdateGroupRequest) error {
	update := bson.M{}

	if req.Name != nil {
		if *req.Name == "" {
			return errors.New("group name cannot be empty")
		}
		update["name"] = *req.Name
	}

	if req.Weight != nil {
		if *req.Weight <= 0 {
			return errors.New("weight must be positive")
		}
		update["weight"] = *req.Weight
	}

	if len(update) == 0 {
		return errors.New("no fields to update")
	}

	result, err := s.db.Collection("groups").UpdateOne(
		ctx,
		bson.M{"_id": groupID},
		bson.M{"$set": update},
	)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("group not found")
	}

	return nil
}

// DeleteGroup deletes a group (ADMIN only)
// Note: Should check if any users are still assigned to this group
func (s *GroupService) DeleteGroup(ctx context.Context, groupID primitive.ObjectID) error {
	// Check if any users are in this group
	count, err := s.db.Collection("users").CountDocuments(ctx, bson.M{"group_id": groupID})
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if count > 0 {
		return errors.New("cannot delete group: users are still assigned to it")
	}

	// Delete all allocations for this group
	_, err = s.db.Collection("allocations").DeleteMany(ctx, bson.M{
		"subject_type": "group",
		"subject_id":   groupID,
	})
	if err != nil {
		log.Printf("[WARN] Failed to delete allocations for group %s: %v", groupID.Hex(), err)
	}

	result, err := s.db.Collection("groups").DeleteOne(ctx, bson.M{"_id": groupID})
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	if result.DeletedCount == 0 {
		return errors.New("group not found")
	}

	log.Printf("[INFO] Deleted group %s and its allocations", groupID.Hex())
	return nil
}
