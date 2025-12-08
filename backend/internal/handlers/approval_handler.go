package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
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

// GetAllRequests retrieves all approval requests (ADMIN only)
func (h *ApprovalHandler) GetAllRequests(c *fiber.Ctx) error {
	requests, err := h.approvalService.GetAllRequests(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve requests",
		})
	}

	return c.JSON(fiber.Map{
		"requests": requests,
		"total":    len(requests),
	})
}

// ApproveRequest approves an approval request (ADMIN only)
func (h *ApprovalHandler) ApproveRequest(c *fiber.Ctx) error {
	requestID := c.Params("id")
	if requestID == "" {
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
	requestID := c.Params("id")
	if requestID == "" {
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
