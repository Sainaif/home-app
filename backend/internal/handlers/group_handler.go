package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GroupHandler struct {
	groupService *services.GroupService
	auditService *services.AuditService
}

func NewGroupHandler(groupService *services.GroupService, auditService *services.AuditService) *GroupHandler {
	return &GroupHandler{
		groupService: groupService,
		auditService: auditService,
	}
}

// CreateGroup creates a new group (ADMIN only)
func (h *GroupHandler) CreateGroup(c *fiber.Ctx) error {
	userID, _ := middleware.GetUserID(c)
	userEmail := c.Locals("userEmail").(string)

	var req services.CreateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	group, err := h.groupService.CreateGroup(c.Context(), req)
	if err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_group", "group", nil,
			map[string]interface{}{"name": req.Name, "error": err.Error()},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_group", "group", &group.ID,
		map[string]interface{}{"name": req.Name, "weight": req.Weight},
		c.IP(), c.Get("User-Agent"), "success")

	return c.Status(fiber.StatusCreated).JSON(group)
}

// GetGroups retrieves all groups
func (h *GroupHandler) GetGroups(c *fiber.Ctx) error {
	groups, err := h.groupService.GetGroups(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(groups)
}

// GetGroup retrieves a specific group
func (h *GroupHandler) GetGroup(c *fiber.Ctx) error {
	id := c.Params("id")
	groupID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	group, err := h.groupService.GetGroup(c.Context(), groupID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(group)
}

// UpdateGroup updates a group (ADMIN only)
func (h *GroupHandler) UpdateGroup(c *fiber.Ctx) error {
	userID, _ := middleware.GetUserID(c)
	userEmail := c.Locals("userEmail").(string)

	id := c.Params("id")
	groupID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	var req services.UpdateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.groupService.UpdateGroup(c.Context(), groupID, req); err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "update_group", "group", &groupID,
			map[string]interface{}{"error": err.Error()},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "update_group", "group", &groupID,
		map[string]interface{}{"changes": req},
		c.IP(), c.Get("User-Agent"), "success")

	return c.JSON(fiber.Map{
		"message": "Group updated successfully",
	})
}

// DeleteGroup deletes a group (ADMIN only)
func (h *GroupHandler) DeleteGroup(c *fiber.Ctx) error {
	userID, _ := middleware.GetUserID(c)
	userEmail := c.Locals("userEmail").(string)

	id := c.Params("id")
	groupID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid group ID",
		})
	}

	if err := h.groupService.DeleteGroup(c.Context(), groupID); err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "delete_group", "group", &groupID,
			map[string]interface{}{"error": err.Error()},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "delete_group", "group", &groupID,
		map[string]interface{}{},
		c.IP(), c.Get("User-Agent"), "success")

	return c.JSON(fiber.Map{
		"message": "Group deleted successfully",
	})
}
