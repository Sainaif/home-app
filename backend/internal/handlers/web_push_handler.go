package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WebPushHandler struct {
	webPushService *services.WebPushService
}

func NewWebPushHandler(webPushService *services.WebPushService) *WebPushHandler {
	return &WebPushHandler{webPushService: webPushService}
}

func (h *WebPushHandler) CreateSubscription(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(primitive.ObjectID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var subscription models.WebPushSubscription
	if err := c.BodyParser(&subscription); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	subscription.UserID = user

	if err := h.webPushService.CreateSubscription(c.Context(), &subscription); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create subscription"})
	}

	return c.SendStatus(fiber.StatusCreated)
}

func (h *WebPushHandler) GetSubscriptions(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(primitive.ObjectID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	subscriptions, err := h.webPushService.GetSubscriptionsByUserID(c.Context(), user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get subscriptions"})
	}

	return c.JSON(subscriptions)
}

func (h *WebPushHandler) DeleteSubscription(c *fiber.Ctx) error {
	_, ok := c.Locals("user").(primitive.ObjectID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	endpoint := c.Query("endpoint")
	if endpoint == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "endpoint is required"})
	}

	if err := h.webPushService.DeleteSubscription(c.Context(), endpoint); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete subscription"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
