package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/config"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
)

type UserHandler struct {
	userService  *services.UserService
	auditService *services.AuditService
	roleService  *services.RoleService
	config       *config.Config
}

func NewUserHandler(userService *services.UserService, auditService *services.AuditService, roleService *services.RoleService, cfg *config.Config) *UserHandler {
	return &UserHandler{
		userService:  userService,
		auditService: auditService,
		roleService:  roleService,
		config:       cfg,
	}
}

// CreateUser creates a new user (ADMIN only)
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	var req services.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := h.userService.CreateUser(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

// GetUsers retrieves all users (ADMIN only)
func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
	users, err := h.userService.GetUsers(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(users)
}

// GetUser retrieves a specific user
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	user, err := h.userService.GetUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(user)
}

// UpdateUser updates a user
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Get current user
	currentUserID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	currentUserRole, err := middleware.GetUserRole(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Allow users to update themselves or admins to update anyone
	if currentUserID != userID && currentUserRole != "ADMIN" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You can only update your own profile",
		})
	}

	var req services.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get target user before update for audit
	targetUser, _ := h.userService.GetUser(c.Context(), userID)

	// Get current user email for audit logging
	currentEmail, err := middleware.GetUserEmail(c)
	if err != nil {
		currentEmail = "unknown"
	}

	if err := h.userService.UpdateUser(c.Context(), userID, req); err != nil {
		h.auditService.LogAction(c.Context(), currentUserID, currentEmail, "", "user.update", "user", &userID, map[string]interface{}{"error": err.Error()}, c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Log successful update
	details := map[string]interface{}{"targetUser": targetUser.Email}
	if req.Role != nil {
		details["newRole"] = *req.Role
	}
	h.auditService.LogAction(c.Context(), currentUserID, currentEmail, "", "user.update", "user", &userID, details, c.IP(), c.Get("User-Agent"), "success")

	return c.JSON(fiber.Map{
		"message": "User updated successfully",
	})
}

// ChangePassword allows users to change their own password
func (h *UserHandler) ChangePassword(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.userService.ChangePassword(c.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

// GetMe retrieves the current user's profile with permissions
func (h *UserHandler) GetMe(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	user, err := h.userService.GetUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Fetch role permissions
	permissions, err := h.roleService.GetRolePermissions(c.Context(), user.Role)
	if err != nil {
		// If role not found, return user without permissions
		permissions = []string{}
	}

	// Return user with permissions
	return c.JSON(fiber.Map{
		"id":                 user.ID,
		"email":              user.Email,
		"name":               user.Name,
		"role":               user.Role,
		"groupId":            user.GroupID,
		"isActive":           user.IsActive,
		"mustChangePassword": user.MustChangePassword,
		"createdAt":          user.CreatedAt,
		"permissions":        permissions,
	})
}

// ForcePasswordChange forces a user to change their password on next login (ADMIN only)
func (h *UserHandler) ForcePasswordChange(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	if err := h.userService.ForcePasswordChange(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User will be required to change password on next login",
	})
}

// GeneratePasswordResetLink generates a password reset link for a user (ADMIN only)
func (h *UserHandler) GeneratePasswordResetLink(c *fiber.Ctx) error {
	// Get target user ID from params
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Get admin user ID from context
	adminID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Parse request body
	var req struct {
		ExpirationMinutes int `json:"expirationMinutes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate expiration time (1 minute to 24 hours)
	if req.ExpirationMinutes < 1 || req.ExpirationMinutes > 1440 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Expiration minutes must be between 1 and 1440 (24 hours)",
		})
	}

	// Generate reset link
	resetURL, err := h.userService.GeneratePasswordResetToken(
		c.Context(),
		userID,
		adminID,
		req.ExpirationMinutes,
		h.config.App.BaseURL,
	)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Log the action
	adminEmail, err := middleware.GetUserEmail(c)
	if err != nil {
		adminEmail = "unknown"
	}
	h.auditService.LogAction(
		c.Context(),
		adminID,
		adminEmail,
		"",
		"password_reset_link.generate",
		"user",
		&userID,
		map[string]interface{}{
			"expirationMinutes": req.ExpirationMinutes,
		},
		c.IP(),
		c.Get("User-Agent"),
		"success",
	)

	return c.JSON(fiber.Map{
		"resetURL":         resetURL,
		"expiresInMinutes": req.ExpirationMinutes,
		"message":          "Password reset link generated successfully",
	})
}

// DeleteUser deletes a user (ADMIN only)
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Get current user to prevent self-deletion
	currentUserID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if currentUserID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot delete your own account",
		})
	}

	if err := h.userService.DeleteUser(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}
