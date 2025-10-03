package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RoleHandler struct {
	roleService       *services.RoleService
	permissionService *services.PermissionService
	auditService      *services.AuditService
	eventService      *services.EventService
	userService       *services.UserService
}

func NewRoleHandler(roleService *services.RoleService, permissionService *services.PermissionService, auditService *services.AuditService, eventService *services.EventService, userService *services.UserService) *RoleHandler {
	return &RoleHandler{
		roleService:       roleService,
		permissionService: permissionService,
		auditService:      auditService,
		eventService:      eventService,
		userService:       userService,
	}
}

// GetAllRoles retrieves all roles (ADMIN only)
func (h *RoleHandler) GetAllRoles(c *fiber.Ctx) error {
	roles, err := h.roleService.GetAllRoles(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve roles",
		})
	}
	return c.JSON(roles)
}

// GetAllPermissions retrieves all permissions (ADMIN only)
func (h *RoleHandler) GetAllPermissions(c *fiber.Ctx) error {
	permissions, err := h.permissionService.GetPermissionsByCategory(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve permissions",
		})
	}
	return c.JSON(permissions)
}

// CreateRole creates a new custom role (ADMIN only)
func (h *RoleHandler) CreateRole(c *fiber.Ctx) error {
	userID := c.Locals(middleware.UserIDKey).(primitive.ObjectID)
	userEmail := c.Locals(middleware.UserEmail).(string)

	var req struct {
		Name        string   `json:"name"`
		DisplayName string   `json:"displayName"`
		Permissions []string `json:"permissions"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" || req.DisplayName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name and display name are required",
		})
	}

	role, err := h.roleService.CreateRole(c.Context(), req.Name, req.DisplayName, req.Permissions)
	if err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, "", "role.create", "role", nil, map[string]interface{}{"name": req.Name, "error": err.Error()}, c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, "", "role.create", "role", &role.ID, map[string]interface{}{"name": req.Name, "displayName": req.DisplayName}, c.IP(), c.Get("User-Agent"), "success")
	return c.Status(fiber.StatusCreated).JSON(role)
}

// UpdateRole updates a role's permissions (ADMIN only)
func (h *RoleHandler) UpdateRole(c *fiber.Ctx) error {
	userID := c.Locals(middleware.UserIDKey).(primitive.ObjectID)
	userEmail := c.Locals(middleware.UserEmail).(string)

	id := c.Params("id")
	roleID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role ID",
		})
	}

	var req struct {
		DisplayName string   `json:"displayName"`
		Permissions []string `json:"permissions"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get role before update for audit trail
	oldRole, err := h.roleService.GetRoleByID(c.Context(), roleID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Role not found",
		})
	}

	err = h.roleService.UpdateRole(c.Context(), roleID, req.DisplayName, req.Permissions)
	if err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, "", "role.update", "role", &roleID, map[string]interface{}{
			"roleName": oldRole.Name,
			"error":    err.Error(),
		}, c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, "", "role.update", "role", &roleID, map[string]interface{}{
		"roleName":       oldRole.Name,
		"oldDisplayName": oldRole.DisplayName,
		"newDisplayName": req.DisplayName,
		"oldPermissions": oldRole.Permissions,
		"newPermissions": req.Permissions,
	}, c.IP(), c.Get("User-Agent"), "success")

	// Notify all users with this role that their permissions have changed
	userIDs, err := h.userService.GetUserIDsByRole(c.Context(), oldRole.Name)
	if err == nil && len(userIDs) > 0 {
		h.eventService.BroadcastToUserIDs(userIDs, services.EventPermissionsUpdated, map[string]interface{}{
			"message": "Twoje uprawnienia zostały zaktualizowane. Odśwież stronę, aby zastosować zmiany.",
		})
	}

	return c.JSON(fiber.Map{"success": true})
}

// DeleteRole deletes a custom role (ADMIN only)
func (h *RoleHandler) DeleteRole(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)
	userEmail := c.Locals("userEmail").(string)

	id := c.Params("id")
	roleID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role ID",
		})
	}

	// Get role before deletion for audit trail
	role, err := h.roleService.GetRoleByID(c.Context(), roleID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Role not found",
		})
	}

	err = h.roleService.DeleteRole(c.Context(), roleID)
	if err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, "", "role.delete", "role", &roleID, map[string]interface{}{
			"roleName": role.Name,
			"error":    err.Error(),
		}, c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, "", "role.delete", "role", &roleID, map[string]interface{}{
		"roleName":    role.Name,
		"displayName": role.DisplayName,
	}, c.IP(), c.Get("User-Agent"), "success")
	return c.JSON(fiber.Map{"success": true})
}
