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

// GetLogs retrieves audit logs with comprehensive filtering (ADMIN only)
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

	// Build comprehensive filters
	filter := bson.M{}

	// User filters
	if userID := c.Query("userId"); userID != "" {
		filter["user_id"] = userID
	}
	if userEmail := c.Query("userEmail"); userEmail != "" {
		// Support partial email search
		filter["user_email"] = bson.M{"$regex": userEmail, "$options": "i"}
	}
	if userName := c.Query("userName"); userName != "" {
		// Support partial name search
		filter["user_name"] = bson.M{"$regex": userName, "$options": "i"}
	}

	// Action filters
	if action := c.Query("action"); action != "" {
		filter["action"] = action
	}
	if actions := c.Query("actions"); actions != "" {
		// Support multiple actions comma-separated
		filter["action"] = bson.M{"$in": parseCommaSeparated(actions)}
	}

	// Resource filters
	if resourceType := c.Query("resourceType"); resourceType != "" {
		filter["resource_type"] = resourceType
	}
	if resourceTypes := c.Query("resourceTypes"); resourceTypes != "" {
		// Support multiple resource types
		filter["resource_type"] = bson.M{"$in": parseCommaSeparated(resourceTypes)}
	}
	if resourceID := c.Query("resourceId"); resourceID != "" {
		filter["resource_id"] = resourceID
	}

	// Status filter
	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}

	// IP Address filter
	if ipAddress := c.Query("ipAddress"); ipAddress != "" {
		filter["ip_address"] = ipAddress
	}

	// Date range filters
	if dateFrom := c.Query("dateFrom"); dateFrom != "" {
		if filter["created_at"] == nil {
			filter["created_at"] = bson.M{}
		}
		filter["created_at"].(bson.M)["$gte"] = dateFrom
	}
	if dateTo := c.Query("dateTo"); dateTo != "" {
		if filter["created_at"] == nil {
			filter["created_at"] = bson.M{}
		}
		filter["created_at"].(bson.M)["$lte"] = dateTo
	}

	// Search across multiple fields
	if search := c.Query("search"); search != "" {
		filter["$or"] = []bson.M{
			{"user_email": bson.M{"$regex": search, "$options": "i"}},
			{"user_name": bson.M{"$regex": search, "$options": "i"}},
			{"action": bson.M{"$regex": search, "$options": "i"}},
			{"resource_type": bson.M{"$regex": search, "$options": "i"}},
			{"ip_address": bson.M{"$regex": search, "$options": "i"}},
		}
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
		"filters":    filter, // Return active filters for debugging
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
