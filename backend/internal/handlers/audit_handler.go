package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/services"
)

type AuditHandler struct {
	auditService *services.AuditService
}

func NewAuditHandler(auditService *services.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// GetLogs retrieves audit logs with pagination (ADMIN only)
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
	offset := (page - 1) * limit

	logs, err := h.auditService.GetLogs(c.Context(), limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve audit logs",
		})
	}

	return c.JSON(fiber.Map{
		"logs":  logs,
		"page":  page,
		"limit": limit,
	})
}

// Helper function to parse comma-separated values
func parseCommaSeparated(input string) []string {
	if input == "" {
		return []string{}
	}

	parts := []string{}
	current := ""
	for _, c := range input {
		if c == ',' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
