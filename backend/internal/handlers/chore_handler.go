package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChoreHandler struct {
	choreService *services.ChoreService
}

func NewChoreHandler(choreService *services.ChoreService) *ChoreHandler {
	return &ChoreHandler{choreService: choreService}
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

	return c.Status(fiber.StatusCreated).JSON(assignment)
}

// GetChoreAssignments retrieves chore assignments with optional filters
func (h *ChoreHandler) GetChoreAssignments(c *fiber.Ctx) error {
	userIDStr := c.Query("userId")
	status := c.Query("status")

	var userIDPtr *primitive.ObjectID
	if userIDStr != "" {
		userID, err := primitive.ObjectIDFromHex(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid user ID",
			})
		}
		userIDPtr = &userID
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
	id := c.Params("id")
	assignmentID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
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

	assignment1ID, err := primitive.ObjectIDFromHex(req.Assignment1ID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid assignment1Id",
		})
	}

	assignment2ID, err := primitive.ObjectIDFromHex(req.Assignment2ID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid assignment2Id",
		})
	}

	if err := h.choreService.SwapChoreAssignment(c.Context(), assignment1ID, assignment2ID); err != nil {
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
	id := c.Params("id")
	choreID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
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
	id := c.Params("id")
	choreID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
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
