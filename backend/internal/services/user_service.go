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
	"github.com/sainaif/holy-home/internal/utils"
)

type UserService struct {
	db *mongo.Database
}

func NewUserService(db *mongo.Database) *UserService {
	return &UserService{db: db}
}

type CreateUserRequest struct {
	Name         string              `json:"name"`
	Email        string              `json:"email"`
	Role         string              `json:"role"` // ADMIN, RESIDENT
	GroupID      *primitive.ObjectID `json:"groupId,omitempty"`
	TempPassword *string             `json:"tempPassword,omitempty"`
}

type UpdateUserRequest struct {
	Email    *string             `json:"email,omitempty"`
	Name     *string             `json:"name,omitempty"`
	Role     *string             `json:"role,omitempty"`
	GroupID  *primitive.ObjectID `json:"groupId,omitempty"`
	IsActive *bool               `json:"isActive,omitempty"`
}

// CreateUser creates a new user (ADMIN only)
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*models.User, error) {
	// Validate role exists
	roleCount, err := s.db.Collection("roles").CountDocuments(ctx, bson.M{"name": req.Role})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if roleCount == 0 {
		return nil, errors.New("invalid role: role does not exist")
	}

	// Check if email already exists
	count, err := s.db.Collection("users").CountDocuments(ctx, bson.M{"email": req.Email})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if count > 0 {
		return nil, errors.New("user with this email already exists")
	}

	// Generate password hash
	password := "ChangeMe123!" // Default password
	if req.TempPassword != nil {
		password = *req.TempPassword
	}
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Use provided name, or fallback to email if empty
	name := req.Name
	if name == "" {
		name = req.Email
	}

	user := models.User{
		ID:                primitive.NewObjectID(),
		Email:             req.Email,
		Name:              name,
		PasswordHash:      passwordHash,
		Role:              req.Role,
		GroupID:           req.GroupID,
		IsActive:          true,
		MustChangePassword: true, // Force password change on first login
		CreatedAt:         time.Now(),
	}

	_, err = s.db.Collection("users").InsertOne(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// GetUsers retrieves all users (ADMIN only)
func (s *UserService) GetUsers(ctx context.Context) ([]models.User, error) {
	cursor, err := s.db.Collection("users").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	return users, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, userID primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &user, nil
}

// UpdateUser updates a user (ADMIN only)
func (s *UserService) UpdateUser(ctx context.Context, userID primitive.ObjectID, req UpdateUserRequest) error {
	update := bson.M{}
	unset := bson.M{}

	if req.Email != nil {
		// Check if new email already exists
		count, err := s.db.Collection("users").CountDocuments(ctx, bson.M{
			"email": *req.Email,
			"_id":   bson.M{"$ne": userID},
		})
		if err != nil {
			return fmt.Errorf("database error: %w", err)
		}
		if count > 0 {
			return errors.New("email already in use")
		}
		update["email"] = *req.Email
	}

	if req.Name != nil {
		update["name"] = *req.Name
	}

	if req.Role != nil {
		// Validate role exists
		roleCount, err := s.db.Collection("roles").CountDocuments(ctx, bson.M{"name": *req.Role})
		if err != nil {
			return fmt.Errorf("database error: %w", err)
		}
		if roleCount == 0 {
			return errors.New("invalid role: role does not exist")
		}
		update["role"] = *req.Role
	}

	if req.GroupID != nil {
		log.Printf("[DEBUG] GroupID is not nil: %v, IsZero: %v", *req.GroupID, req.GroupID.IsZero())
		if req.GroupID.IsZero() {
			// Zero ObjectID means remove the group
			log.Printf("[DEBUG] Adding group_id to unset")
			unset["group_id"] = ""
		} else {
			log.Printf("[DEBUG] Setting group_id to %v", *req.GroupID)
			update["group_id"] = *req.GroupID
		}
	}

	if req.IsActive != nil {
		update["is_active"] = *req.IsActive
	}

	if len(update) == 0 && len(unset) == 0 {
		return errors.New("no fields to update")
	}

	updateDoc := bson.M{}
	if len(update) > 0 {
		updateDoc["$set"] = update
	}
	if len(unset) > 0 {
		updateDoc["$unset"] = unset
	}

	result, err := s.db.Collection("users").UpdateOne(
		ctx,
		bson.M{"_id": userID},
		updateDoc,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

// ChangePassword allows users to change their own password
func (s *UserService) ChangePassword(ctx context.Context, userID primitive.ObjectID, oldPassword, newPassword string) error {
	var user models.User
	err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return errors.New("user not found")
	}

	// Verify old password
	valid, err := utils.VerifyPassword(oldPassword, user.PasswordHash)
	if err != nil || !valid {
		return errors.New("invalid current password")
	}

	// Hash new password
	newHash, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password and clear must_change_password flag
	_, err = s.db.Collection("users").UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{
			"password_hash": newHash,
			"must_change_password": false,
		}},
	)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// ForcePasswordChange marks a user as needing to change their password
func (s *UserService) ForcePasswordChange(ctx context.Context, userID primitive.ObjectID) error {
	_, err := s.db.Collection("users").UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"must_change_password": true}},
	)
	if err != nil {
		return fmt.Errorf("failed to set password change flag: %w", err)
	}
	return nil
}

// DeleteUser deletes a user from the system
// GetUserIDsByRole returns all user IDs that have a specific role
func (s *UserService) GetUserIDsByRole(ctx context.Context, roleName string) ([]primitive.ObjectID, error) {
	cursor, err := s.db.Collection("users").Find(ctx, bson.M{"role": roleName, "is_active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var userIDs []primitive.ObjectID
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			continue
		}
		userIDs = append(userIDs, user.ID)
	}

	return userIDs, nil
}

func (s *UserService) DeleteUser(ctx context.Context, userID primitive.ObjectID) error {
	// Check if user is active
	var user models.User
	err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	if user.IsActive {
		return fmt.Errorf("cannot delete active user, deactivate first")
	}

	// Delete the user
	result, err := s.db.Collection("users").DeleteOne(ctx, bson.M{"_id": userID})
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
