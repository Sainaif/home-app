package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/services"
)

type ReminderHandler struct {
	reminderService *services.ReminderService
}

func NewReminderHandler(reminderService *services.ReminderService) *ReminderHandler {
	return &ReminderHandler{reminderService: reminderService}
}

// SendDebtReminder sends a reminder to a user about their debt to the sender
// POST /api/reminders/debt/:userId
func (h *ReminderHandler) SendDebtReminder(c *fiber.Ctx) error {
	targetUserID := c.Params("userId")
	if targetUserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	senderUserID := c.Locals("userID").(string)

	if err := h.reminderService.SendDebtReminder(c.Context(), targetUserID, senderUserID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Reminder sent successfully",
	})
}

// SendChoreReminder sends a reminder about a specific chore assignment
// POST /api/reminders/chore/:assignmentId
func (h *ReminderHandler) SendChoreReminder(c *fiber.Ctx) error {
	assignmentID := c.Params("assignmentId")
	if assignmentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Assignment ID is required",
		})
	}

	senderUserID := c.Locals("userID").(string)

	if err := h.reminderService.SendChoreReminder(c.Context(), assignmentID, senderUserID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Reminder sent successfully",
	})
}

// SendLowSuppliesReminder broadcasts a reminder about low supplies to all users
// POST /api/reminders/supplies
func (h *ReminderHandler) SendLowSuppliesReminder(c *fiber.Ctx) error {
	senderUserID := c.Locals("userID").(string)

	notifiedCount, err := h.reminderService.SendLowSuppliesReminder(c.Context(), senderUserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":       "Reminder sent successfully",
		"notifiedCount": notifiedCount,
	})
}
