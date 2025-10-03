package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserHandler struct {
	userService  *services.UserService
	auditService *services.AuditService
}

func NewUserHandler(userService *services.UserService, auditService *services.AuditService) *UserHandler {
	return &UserHandler{
		userService:  userService,
		auditService: auditService,
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
	id := c.Params("id")
	userID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
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
	id := c.Params("id")
	userID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
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

	// Debug logging
	log.Printf("[DEBUG] UpdateUser request for user %s: Email=%v, Name=%v, Role=%v, GroupID=%v, IsActive=%v",
		userID.Hex(),
		req.Email,
		req.Name,
		req.Role,
		req.GroupID,
		req.IsActive)

	// Get target user before update for audit
	targetUser, _ := h.userService.GetUser(c.Context(), userID)

	if err := h.userService.UpdateUser(c.Context(), userID, req); err != nil {
		currentEmail := c.Locals(middleware.UserEmail).(string)
		h.auditService.LogAction(c.Context(), currentUserID, currentEmail, "", "user.update", "user", &userID, map[string]interface{}{"error": err.Error()}, c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Log successful update
	currentEmail := c.Locals(middleware.UserEmail).(string)
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
		log.Printf("[ChangePassword] Failed to parse request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	log.Printf("[ChangePassword] Request from user %s: oldPassword length=%d, newPassword length=%d",
		userID.Hex(), len(req.OldPassword), len(req.NewPassword))

	if err := h.userService.ChangePassword(c.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		log.Printf("[ChangePassword] Service error for user %s: %v", userID.Hex(), err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

// GetMe retrieves the current user's profile
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

	return c.JSON(user)
}

// ForcePasswordChange forces a user to change their password on next login (ADMIN only)
func (h *UserHandler) ForcePasswordChange(c *fiber.Ctx) error {
	id := c.Params("id")
	userID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
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

// DeleteUser deletes a user (ADMIN only)
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	userID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
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