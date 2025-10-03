package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/services"
	"go.mongodb.org/mongo-driver/bson"
)

type AuditHandler struct {
	auditService *services.AuditService
}

func NewAuditHandler(auditService *services.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// GetLogs retrieves audit logs (ADMIN only)
func (h *AuditHandler) GetLogs(c *fiber.Ctx) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	skip := (page - 1) * limit

	// Parse filters
	filter := bson.M{}
	if userID := c.Query("userId"); userID != "" {
		filter["user_id"] = userID
	}
	if userEmail := c.Query("userEmail"); userEmail != "" {
		filter["user_email"] = userEmail
	}
	if action := c.Query("action"); action != "" {
		filter["action"] = action
	}
	if resourceType := c.Query("resourceType"); resourceType != "" {
		filter["resource_type"] = resourceType
	}
	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}

	logs, total, err := h.auditService.GetLogs(c.Context(), limit, skip, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve audit logs",
		})
	}

	return c.JSON(fiber.Map{
		"logs":       logs,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": (total + int64(limit) - 1) / int64(limit),
	})
}
