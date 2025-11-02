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

	"github.com/sainaif/holy-home/internal/config"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/utils"
)

type UserService struct {
	db     *mongo.Database
	config *config.Config
}

func NewUserService(db *mongo.Database, cfg *config.Config) *UserService {
	return &UserService{
		db:     db,
		config: cfg,
	}
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

// GeneratePasswordResetToken generates a password reset token for a user
// Returns the full reset URL that should be shared with the user
func (s *UserService) GeneratePasswordResetToken(ctx context.Context, userID primitive.ObjectID, adminID primitive.ObjectID, expirationMinutes int, baseURL string) (string, error) {
	// Verify user exists
	var user models.User
	err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", errors.New("user not found")
		}
		return "", fmt.Errorf("database error: %w", err)
	}

	// Generate secure token
	token, err := utils.GenerateSecureToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Hash the token for storage
	tokenHash := utils.HashToken(token)

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(expirationMinutes) * time.Minute)

	// Create reset token record
	resetToken := models.PasswordResetToken{
		ID:               primitive.NewObjectID(),
		UserID:           userID,
		TokenHash:        tokenHash,
		ExpiresAt:        expiresAt,
		Used:             false,
		CreatedAt:        time.Now(),
		CreatedByAdminID: adminID,
	}

	// Store in database
	_, err = s.db.Collection("password_reset_tokens").InsertOne(ctx, resetToken)
	if err != nil {
		return "", fmt.Errorf("failed to store reset token: %w", err)
	}

	// Construct the full reset URL
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", baseURL, token)

	return resetURL, nil
}

// ValidateResetToken validates a password reset token
// Returns the token record if valid, or an error if invalid/expired/used
func (s *UserService) ValidateResetToken(ctx context.Context, token string) (*models.PasswordResetToken, error) {
	// Validate token format
	if err := utils.ValidateTokenFormat(token); err != nil {
		return nil, fmt.Errorf("invalid token format: %w", err)
	}

	// Hash the token to find it in database
	tokenHash := utils.HashToken(token)

	// Find the token in database
	var resetToken models.PasswordResetToken
	err := s.db.Collection("password_reset_tokens").FindOne(ctx, bson.M{
		"token_hash": tokenHash,
	}).Decode(&resetToken)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("invalid or expired reset token")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Check if token is expired
	if time.Now().After(resetToken.ExpiresAt) {
		return nil, errors.New("reset token has expired")
	}

	// Check if token has already been used
	if resetToken.Used {
		return nil, errors.New("reset token has already been used")
	}

	return &resetToken, nil
}

// ResetPasswordWithToken resets a user's password using a valid reset token
// Returns JWT tokens for automatic login after password reset
func (s *UserService) ResetPasswordWithToken(ctx context.Context, token string, newPassword string) (map[string]string, error) {
	// Validate the token
	resetToken, err := s.ValidateResetToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Hash the new password
	newHash, err := utils.HashPassword(newPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update user password and clear must_change_password flag
	now := time.Now()
	_, err = s.db.Collection("users").UpdateOne(
		ctx,
		bson.M{"_id": resetToken.UserID},
		bson.M{"$set": bson.M{
			"password_hash":        newHash,
			"must_change_password": false,
		}},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	// Mark token as used
	tokenHash := utils.HashToken(token)
	_, err = s.db.Collection("password_reset_tokens").UpdateOne(
		ctx,
		bson.M{"token_hash": tokenHash},
		bson.M{"$set": bson.M{
			"used":    true,
			"used_at": &now,
		}},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mark token as used: %w", err)
	}

	// Get user details for JWT generation
	var user models.User
	err = s.db.Collection("users").FindOne(ctx, bson.M{"_id": resetToken.UserID}).Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Generate JWT tokens for automatic login
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email, user.Role, s.config.JWT.Secret, s.config.JWT.AccessTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, s.config.JWT.RefreshSecret, s.config.JWT.RefreshTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return map[string]string{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	}, nil
}

// CleanupExpiredResetTokens removes expired and used tokens from the database
// This should be called periodically (e.g., via a cron job)
func (s *UserService) CleanupExpiredResetTokens(ctx context.Context) error {
	now := time.Now()

	// Delete tokens that are either expired or used
	filter := bson.M{
		"$or": []bson.M{
			{"expires_at": bson.M{"$lt": now}},
			{"used": true},
		},
	}

	result, err := s.db.Collection("password_reset_tokens").DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	if result.DeletedCount > 0 {
		log.Printf("Cleaned up %d expired/used password reset tokens", result.DeletedCount)
	}

	return nil
}
