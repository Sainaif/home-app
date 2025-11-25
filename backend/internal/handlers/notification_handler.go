
package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationHandler struct {
	notificationService *services.NotificationService
}

func NewNotificationHandler(notificationService *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

func (h *NotificationHandler) GetNotifications(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(primitive.ObjectID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	notifications, err := h.notificationService.GetNotificationsForUser(c.Context(), user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get notifications"})
	}

	return c.JSON(notifications)
}

func (h *NotificationHandler) MarkNotificationAsRead(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(primitive.ObjectID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	notificationID, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid notification id"})
	}

	if err := h.notificationService.MarkNotificationAsRead(c.Context(), notificationID, user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to mark notification as read"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *NotificationHandler) MarkAllNotificationsAsRead(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(primitive.ObjectID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if err := h.notificationService.MarkAllNotificationsAsRead(c.Context(), user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to mark all notifications as read"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
