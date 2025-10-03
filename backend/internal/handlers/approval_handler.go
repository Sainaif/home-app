package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ApprovalHandler struct {
	approvalService *services.ApprovalService
}

func NewApprovalHandler(approvalService *services.ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{approvalService: approvalService}
}

// GetPendingRequests retrieves all pending approval requests (ADMIN only)
func (h *ApprovalHandler) GetPendingRequests(c *fiber.Ctx) error {
	requests, err := h.approvalService.GetPendingRequests(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve pending requests",
		})
	}
	return c.JSON(requests)
}

// GetAllRequests retrieves all approval requests with pagination (ADMIN only)
func (h *ApprovalHandler) GetAllRequests(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	skip := (page - 1) * limit

	filter := bson.M{}
	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}
	if userID := c.Query("userId"); userID != "" {
		filter["user_id"] = userID
	}

	requests, total, err := h.approvalService.GetAllRequests(c.Context(), limit, skip, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve requests",
		})
	}

	return c.JSON(fiber.Map{
		"requests":   requests,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": (total + int64(limit) - 1) / int64(limit),
	})
}

// ApproveRequest approves an approval request (ADMIN only)
func (h *ApprovalHandler) ApproveRequest(c *fiber.Ctx) error {
	id := c.Params("id")
	requestID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request ID",
		})
	}

	reviewerID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		Notes *string `json:"notes"`
	}
	c.BodyParser(&req)

	err = h.approvalService.ApproveRequest(c.Context(), requestID, reviewerID, req.Notes)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

// RejectRequest rejects an approval request (ADMIN only)
func (h *ApprovalHandler) RejectRequest(c *fiber.Ctx) error {
	id := c.Params("id")
	requestID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request ID",
		})
	}

	reviewerID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		Notes *string `json:"notes"`
	}
	c.BodyParser(&req)

	err = h.approvalService.RejectRequest(c.Context(), requestID, reviewerID, req.Notes)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"success": true})
}
