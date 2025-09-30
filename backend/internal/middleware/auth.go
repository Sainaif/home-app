package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/config"
	"github.com/sainaif/holy-home/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ContextKey string

const (
	UserIDKey  ContextKey = "userId"
	UserEmail  ContextKey = "userEmail"
	UserRole   ContextKey = "userRole"
)

// AuthMiddleware validates JWT tokens
// Supports both Authorization header and query param (for SSE)
func AuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var token string

		// Try Authorization header first
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		// Fall back to query param (for EventSource/SSE)
		if token == "" {
			token = c.Query("token")
		}

		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization token",
			})
		}

		claims, err := utils.ValidateAccessToken(token, cfg.JWT.Secret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Store user info in context
		c.Locals("userId", claims.UserID)
		c.Locals("userEmail", claims.Email)
		c.Locals("userRole", claims.Role)

		return c.Next()
	}
}

// RequireRole creates a middleware that checks for specific roles
func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("userRole").(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access forbidden: role not found",
			})
		}

		for _, role := range roles {
			if userRole == role {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access forbidden: insufficient permissions",
		})
	}
}

// GetUserID extracts the user ID from the request context
func GetUserID(c *fiber.Ctx) (primitive.ObjectID, error) {
	userID, ok := c.Locals("userId").(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, fiber.ErrUnauthorized
	}
	return userID, nil
}

// GetUserRole extracts the user role from the request context
func GetUserRole(c *fiber.Ctx) (string, error) {
	role, ok := c.Locals("userRole").(string)
	if !ok {
		return "", fiber.ErrUnauthorized
	}
	return role, nil
}

// RequirePermission creates a middleware that checks for specific permissions
// This requires the RoleService to check if the user's role has the permission
func RequirePermission(permission string, roleServiceGetter func() interface{}) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("userRole").(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access forbidden: role not found",
				"debug": "userRole not found in context",
			})
		}

		// Get the RoleService from the getter function
		roleService := roleServiceGetter()
		if roleService == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Role service not available",
				"debug": "roleServiceGetter returned nil",
			})
		}

		// Type assert to RoleService interface with HasPermission method
		type PermissionChecker interface {
			HasPermission(ctx context.Context, roleName, permission string) (bool, error)
		}

		checker, ok := roleService.(PermissionChecker)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":       "Invalid role service",
				"debug":       "type assertion failed",
				"serviceType": fmt.Sprintf("%T", roleService),
			})
		}

		hasPermission, err := checker.HasPermission(c.Context(), userRole, permission)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to check permissions",
				"debug": err.Error(),
				"role":  userRole,
				"perm":  permission,
			})
		}

		if !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access forbidden: insufficient permissions",
				"debug": fmt.Sprintf("Role '%s' does not have permission '%s'", userRole, permission),
			})
		}

		return c.Next()
	}
}