package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/config"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
	"github.com/sainaif/holy-home/internal/utils"
)

type UserService struct {
	users               repository.UserRepository
	groups              repository.GroupRepository
	roles               repository.RoleRepository
	passwordResetTokens repository.PasswordResetTokenRepository
	config              *config.Config
}

func NewUserService(
	users repository.UserRepository,
	groups repository.GroupRepository,
	roles repository.RoleRepository,
	passwordResetTokens repository.PasswordResetTokenRepository,
	cfg *config.Config,
) *UserService {
	return &UserService{
		users:               users,
		groups:              groups,
		roles:               roles,
		passwordResetTokens: passwordResetTokens,
		config:              cfg,
	}
}

type CreateUserRequest struct {
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Username     string  `json:"username,omitempty"` // Optional username for login
	Role         string  `json:"role"`               // ADMIN, RESIDENT
	GroupID      *string `json:"groupId,omitempty"`
	TempPassword *string `json:"tempPassword,omitempty"`
}

type UpdateUserRequest struct {
	Email    *string `json:"email,omitempty"`
	Username *string `json:"username,omitempty"`
	Name     *string `json:"name,omitempty"`
	Role     *string `json:"role,omitempty"`
	GroupID  *string `json:"groupId,omitempty"`
	IsActive *bool   `json:"isActive,omitempty"`
}

// CreateUser creates a new user (ADMIN only)
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*models.User, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		return nil, errors.New("email is required")
	}
	req.Email = email

	// Handle username
	username := strings.TrimSpace(req.Username)
	if s.config.Auth.RequireUsername && username == "" {
		return nil, errors.New("username is required")
	}

	// Validate role exists
	_, err := s.roles.GetByName(ctx, req.Role)
	if err != nil {
		return nil, errors.New("invalid role: role does not exist")
	}

	// Check if email already exists (case-insensitive)
	existingUser, _ := s.users.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Check if username already exists (case-insensitive) if provided
	if username != "" {
		existingUser, _ := s.users.GetByUsername(ctx, username)
		if existingUser != nil {
			return nil, errors.New("user with this username already exists")
		}
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
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = req.Email
	}

	user := models.User{
		ID:                 uuid.New().String(),
		Email:              req.Email,
		Username:           username,
		Name:               name,
		PasswordHash:       passwordHash,
		Role:               req.Role,
		GroupID:            req.GroupID,
		IsActive:           true,
		MustChangePassword: true, // Force password change on first login
		CreatedAt:          time.Now(),
	}

	if err := s.users.Create(ctx, &user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// UserWithGroup extends User with group name for API responses
type UserWithGroup struct {
	models.User
	GroupName string `json:"groupName,omitempty"`
}

// GetUsers retrieves all users with their group names (ADMIN only)
func (s *UserService) GetUsers(ctx context.Context) ([]UserWithGroup, error) {
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Fetch all groups for lookup
	groups, err := s.groups.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch groups: %w", err)
	}

	// Create group lookup map
	groupMap := make(map[string]string)
	for _, g := range groups {
		groupMap[g.ID] = g.Name
	}

	// Map users to UserWithGroup
	result := make([]UserWithGroup, len(users))
	for i, u := range users {
		uwg := UserWithGroup{User: u}
		if u.GroupID != nil {
			uwg.GroupName = groupMap[*u.GroupID]
		}
		result[i] = uwg
	}

	return result, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// UpdateUser updates a user (ADMIN only)
func (s *UserService) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) error {
	// Get existing user
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}

	if req.Email != nil {
		normalizedEmail := strings.ToLower(strings.TrimSpace(*req.Email))
		if normalizedEmail == "" {
			return errors.New("email cannot be empty")
		}
		if !strings.Contains(normalizedEmail, "@") {
			return errors.New("invalid email format")
		}

		// Check if email is already in use by another user
		existingUser, _ := s.users.GetByEmail(ctx, normalizedEmail)
		if existingUser != nil && existingUser.ID != userID {
			return errors.New("email already in use")
		}
		user.Email = normalizedEmail
	}

	if req.Name != nil {
		user.Name = *req.Name
	}

	if req.Role != nil {
		// Validate role exists
		_, err := s.roles.GetByName(ctx, *req.Role)
		if err != nil {
			return errors.New("invalid role: role does not exist")
		}
		user.Role = *req.Role
	}

	if req.GroupID != nil {
		if *req.GroupID == "" {
			// Empty string means remove the group
			user.GroupID = nil
		} else {
			// Verify group exists before assigning
			_, err := s.groups.GetByID(ctx, *req.GroupID)
			if err != nil {
				return errors.New("invalid group: group does not exist")
			}
			user.GroupID = req.GroupID
		}
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := s.users.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// ChangePassword allows users to change their own password
func (s *UserService) ChangePassword(ctx context.Context, userID string, oldPassword, newPassword string) error {
	user, err := s.users.GetByID(ctx, userID)
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
	if err := s.users.UpdatePassword(ctx, userID, newHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if err := s.users.SetMustChangePassword(ctx, userID, false); err != nil {
		return fmt.Errorf("failed to clear must_change_password: %w", err)
	}

	return nil
}

// ForcePasswordChange marks a user as needing to change their password
func (s *UserService) ForcePasswordChange(ctx context.Context, userID string) error {
	if err := s.users.SetMustChangePassword(ctx, userID, true); err != nil {
		return fmt.Errorf("failed to set password change flag: %w", err)
	}
	return nil
}

// GetUserIDsByRole returns all user IDs that have a specific role
func (s *UserService) GetUserIDsByRole(ctx context.Context, roleName string) ([]string, error) {
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, err
	}

	var userIDs []string
	for _, user := range users {
		if user.Role == roleName && user.IsActive {
			userIDs = append(userIDs, user.ID)
		}
	}

	return userIDs, nil
}

// DeleteUser deletes a user from the system
func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
	// Check if user is active
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if user.IsActive {
		return fmt.Errorf("cannot delete active user, deactivate first")
	}

	// Delete the user
	if err := s.users.Delete(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// GeneratePasswordResetToken generates a password reset token for a user
// Returns the full reset URL that should be shared with the user
func (s *UserService) GeneratePasswordResetToken(ctx context.Context, userID string, adminID string, expirationMinutes int, baseURL string) (string, error) {
	// Verify user exists
	_, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return "", errors.New("user not found")
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
		ID:               uuid.New().String(),
		UserID:           userID,
		TokenHash:        tokenHash,
		ExpiresAt:        expiresAt,
		Used:             false,
		CreatedAt:        time.Now(),
		CreatedByAdminID: adminID,
	}

	// Store in database
	if err := s.passwordResetTokens.Create(ctx, &resetToken); err != nil {
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
	resetToken, err := s.passwordResetTokens.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, errors.New("invalid or expired reset token")
	}

	// Check if token is expired
	if time.Now().After(resetToken.ExpiresAt) {
		return nil, errors.New("reset token has expired")
	}

	// Check if token has already been used
	if resetToken.Used {
		return nil, errors.New("reset token has already been used")
	}

	return resetToken, nil
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
	if err := s.users.UpdatePassword(ctx, resetToken.UserID, newHash); err != nil {
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	if err := s.users.SetMustChangePassword(ctx, resetToken.UserID, false); err != nil {
		return nil, fmt.Errorf("failed to clear must_change_password: %w", err)
	}

	// Mark token as used
	if err := s.passwordResetTokens.MarkUsed(ctx, resetToken.ID); err != nil {
		return nil, fmt.Errorf("failed to mark token as used: %w", err)
	}

	// Get user details for JWT generation
	user, err := s.users.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Double-check that the stored password matches what we just set
	valid, err := utils.VerifyPassword(newPassword, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to verify updated password: %w", err)
	}
	if !valid {
		return nil, errors.New("password update verification failed")
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
	if err := s.passwordResetTokens.DeleteExpired(ctx); err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	log.Printf("Cleaned up expired/used password reset tokens")
	return nil
}
