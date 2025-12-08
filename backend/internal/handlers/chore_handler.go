package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
)

type ChoreHandler struct {
	choreService    *services.ChoreService
	approvalService *services.ApprovalService
	roleService     *services.RoleService
	auditService    *services.AuditService
	eventService    *services.EventService
}

func NewChoreHandler(choreService *services.ChoreService, approvalService *services.ApprovalService, roleService *services.RoleService, auditService *services.AuditService, eventService *services.EventService) *ChoreHandler {
	return &ChoreHandler{
		choreService:    choreService,
		approvalService: approvalService,
		roleService:     roleService,
		auditService:    auditService,
		eventService:    eventService,
	}
}

// CreateChore creates a new chore (ADMIN only)
func (h *ChoreHandler) CreateChore(c *fiber.Ctx) error {
	var req services.CreateChoreRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	chore, err := h.choreService.CreateChore(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Broadcast event to all users
	h.eventService.Broadcast(services.EventChoreUpdated, map[string]interface{}{
		"choreId":     chore.ID,
		"name":        chore.Name,
		"description": chore.Description,
		"action":      "created",
	})

	return c.Status(fiber.StatusCreated).JSON(chore)
}

// GetChores retrieves all chores
func (h *ChoreHandler) GetChores(c *fiber.Ctx) error {
	chores, err := h.choreService.GetChores(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(chores)
}

// GetChoresWithAssignments retrieves chores with current assignments
func (h *ChoreHandler) GetChoresWithAssignments(c *fiber.Ctx) error {
	chores, err := h.choreService.GetChoresWithAssignments(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(chores)
}

// AssignChore assigns a chore to a user (ADMIN only)
func (h *ChoreHandler) AssignChore(c *fiber.Ctx) error {
	var req services.AssignChoreRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	assignment, err := h.choreService.AssignChore(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Get chore and user details for notification
	chore, _ := h.choreService.GetChore(c.Context(), req.ChoreID)
	user, _ := h.choreService.GetUserByID(c.Context(), req.AssigneeUserID)

	// Broadcast chore assignment event to the assigned user
	if chore != nil && user != nil {
		h.eventService.BroadcastToUser(req.AssigneeUserID, services.EventChoreAssigned, map[string]interface{}{
			"choreId":      chore.ID,
			"choreName":    chore.Name,
			"assigneeId":   user.ID,
			"assigneeName": user.Name,
			"dueDate":      req.DueDate.Format(time.RFC3339),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(assignment)
}

// GetChoreAssignments retrieves chore assignments with optional filters
func (h *ChoreHandler) GetChoreAssignments(c *fiber.Ctx) error {
	userIDStr := c.Query("userId")
	status := c.Query("status")

	var userIDPtr *string
	if userIDStr != "" {
		userIDPtr = &userIDStr
	}

	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	assignments, err := h.choreService.GetChoreAssignments(c.Context(), userIDPtr, statusPtr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(assignments)
}

// GetMyChoreAssignments retrieves the current user's chore assignments
func (h *ChoreHandler) GetMyChoreAssignments(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	status := c.Query("status")
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	assignments, err := h.choreService.GetChoreAssignments(c.Context(), &userID, statusPtr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(assignments)
}

// UpdateChoreAssignment updates a chore assignment status
func (h *ChoreHandler) UpdateChoreAssignment(c *fiber.Ctx) error {
	assignmentID := c.Params("id")
	if assignmentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid assignment ID",
		})
	}

	var req services.UpdateChoreAssignmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.choreService.UpdateChoreAssignment(c.Context(), assignmentID, req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Chore assignment updated successfully",
	})
}

// SwapChoreAssignment swaps two chore assignments (ADMIN only)
func (h *ChoreHandler) SwapChoreAssignment(c *fiber.Ctx) error {
	var req struct {
		Assignment1ID string `json:"assignment1Id"`
		Assignment2ID string `json:"assignment2Id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Assignment1ID == "" || req.Assignment2ID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid assignment IDs",
		})
	}

	if err := h.choreService.SwapChoreAssignment(c.Context(), req.Assignment1ID, req.Assignment2ID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Chore assignments swapped successfully",
	})
}

// RotateChore creates a new assignment based on rotation (ADMIN only)
func (h *ChoreHandler) RotateChore(c *fiber.Ctx) error {
	choreID := c.Params("id")
	if choreID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid chore ID",
		})
	}

	var req struct {
		DueDate time.Time `json:"dueDate"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	assignment, err := h.choreService.RotateChore(c.Context(), choreID, req.DueDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(assignment)
}

// AutoAssignChore automatically assigns a chore to user with least workload (ADMIN only)
func (h *ChoreHandler) AutoAssignChore(c *fiber.Ctx) error {
	choreID := c.Params("id")
	if choreID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid chore ID",
		})
	}

	var req struct {
		DueDate time.Time `json:"dueDate"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	assignment, err := h.choreService.AutoAssignChore(c.Context(), choreID, req.DueDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(assignment)
}

// GetUserLeaderboard retrieves user leaderboard based on points
func (h *ChoreHandler) GetUserLeaderboard(c *fiber.Ctx) error {
	leaderboard, err := h.choreService.GetUserLeaderboard(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(leaderboard)
}

// DeleteChore deletes a chore (requires approval for non-admins)
func (h *ChoreHandler) DeleteChore(c *fiber.Ctx) error {
	choreID := c.Params("id")
	if choreID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid chore ID",
		})
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	userRole, err := middleware.GetUserRole(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Check if user has permission
	hasPermission, err := h.roleService.HasPermission(c.Context(), userRole, "chores.delete")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permissions",
		})
	}

	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have permission to delete chores",
		})
	}

	// For ADMIN, delete directly. For others, create approval request
	if userRole == "ADMIN" {
		if err := h.choreService.DeleteChore(c.Context(), choreID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Log action
		h.auditService.LogAction(c.Context(), userID, "", "", "chore.delete", "chore", &choreID, nil, c.IP(), c.Get("User-Agent"), "success")

		return c.JSON(fiber.Map{"success": true})
	}

	// Create approval request for non-admin
	_, err = h.approvalService.CreateApprovalRequest(
		c.Context(),
		userID,
		"", // Will be filled from user object
		"", // Will be filled from user object
		"chore.delete",
		"chore",
		&choreID,
		map[string]interface{}{
			"choreId": choreID,
		},
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create approval request",
		})
	}

	return c.JSON(fiber.Map{
		"success":          true,
		"requiresApproval": true,
		"message":          "Delete request submitted for admin approval",
	})
}
