package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/services"
)

type NotificationPreferenceHandler struct {
	notificationPreferenceService *services.NotificationPreferenceService
}

func NewNotificationPreferenceHandler(notificationPreferenceService *services.NotificationPreferenceService) *NotificationPreferenceHandler {
	return &NotificationPreferenceHandler{notificationPreferenceService: notificationPreferenceService}
}

func (h *NotificationPreferenceHandler) GetPreferences(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	preferences, err := h.notificationPreferenceService.GetPreferences(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get preferences"})
	}

	return c.JSON(preferences)
}

func (h *NotificationPreferenceHandler) UpdatePreferences(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req struct {
		Preferences map[string]bool `json:"preferences"`
		AllEnabled  bool            `json:"allEnabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	preferences, err := h.notificationPreferenceService.UpdatePreferences(c.Context(), userID, req.Preferences, req.AllEnabled)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update preferences"})
	}

	return c.JSON(preferences)
}
