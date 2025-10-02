package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SupplyHandler struct {
	supplyService *services.SupplyService
}

func NewSupplyHandler(supplyService *services.SupplyService) *SupplyHandler {
	return &SupplyHandler{
		supplyService: supplyService,
	}
}

// ========== Settings Handlers ==========

// GetSettings retrieves supply settings
func (h *SupplyHandler) GetSettings(c *fiber.Ctx) error {
	settings, err := h.supplyService.GetSettings(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(settings)
}

// UpdateSettings updates supply settings (ADMIN only)
func (h *SupplyHandler) UpdateSettings(c *fiber.Ctx) error {
	var req struct {
		WeeklyContributionPLN float64 `json:"weeklyContributionPLN"`
		ContributionDay       string  `json:"contributionDay"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.supplyService.UpdateSettings(c.Context(), req.WeeklyContributionPLN, req.ContributionDay); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Settings updated successfully",
	})
}

// AdjustBudget manually adjusts budget (ADMIN only)
func (h *SupplyHandler) AdjustBudget(c *fiber.Ctx) error {
	var req struct {
		Adjustment float64 `json:"adjustment"`
		Notes      string  `json:"notes"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.supplyService.AdjustBudget(c.Context(), req.Adjustment, req.Notes); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Budget adjusted successfully",
	})
}

// ========== Item Handlers ==========

// GetItems retrieves supply items with optional status filter
func (h *SupplyHandler) GetItems(c *fiber.Ctx) error {
	statusParam := c.Query("status")
	var status *string
	if statusParam != "" {
		status = &statusParam
	}

	items, err := h.supplyService.GetItems(c.Context(), status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(items)
}

// CreateItem adds a new item to shopping list
func (h *SupplyHandler) CreateItem(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		Name     string  `json:"name"`
		Category string  `json:"category"`
		Quantity *string `json:"quantity"`
		Priority int     `json:"priority"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Default priority to 3 if not specified
	if req.Priority == 0 {
		req.Priority = 3
	}

	item, err := h.supplyService.CreateItem(c.Context(), userID, req.Name, req.Category, req.Quantity, req.Priority)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(item)
}

// UpdateItem updates item details
func (h *SupplyHandler) UpdateItem(c *fiber.Ctx) error {
	id := c.Params("id")
	itemID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid item ID",
		})
	}

	var req struct {
		Name     *string `json:"name"`
		Category *string `json:"category"`
		Quantity *string `json:"quantity"`
		Priority *int    `json:"priority"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.supplyService.UpdateItem(c.Context(), itemID, req.Name, req.Category, req.Quantity, req.Priority); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Item updated successfully",
	})
}

// MarkAsBought marks item as bought
func (h *SupplyHandler) MarkAsBought(c *fiber.Ctx) error {
	id := c.Params("id")
	itemID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid item ID",
		})
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		AmountPLN float64 `json:"amountPLN"`
		Notes     *string `json:"notes"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.supplyService.MarkAsBought(c.Context(), itemID, userID, req.AmountPLN, req.Notes); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Item marked as bought successfully",
	})
}

// DeleteItem deletes an item
func (h *SupplyHandler) DeleteItem(c *fiber.Ctx) error {
	id := c.Params("id")
	itemID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid item ID",
		})
	}

	if err := h.supplyService.DeleteItem(c.Context(), itemID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Item deleted successfully",
	})
}

// ========== Contribution Handlers ==========

// GetContributions retrieves contributions
func (h *SupplyHandler) GetContributions(c *fiber.Ctx) error {
	var userID *primitive.ObjectID
	userIDParam := c.Query("userId")
	if userIDParam != "" {
		id, err := primitive.ObjectIDFromHex(userIDParam)
		if err == nil {
			userID = &id
		}
	}

	var fromDate *time.Time
	fromParam := c.Query("from")
	if fromParam != "" {
		from, err := time.Parse(time.RFC3339, fromParam)
		if err == nil {
			fromDate = &from
		}
	}

	contributions, err := h.supplyService.GetContributions(c.Context(), userID, fromDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(contributions)
}

// CreateContribution creates a manual contribution
func (h *SupplyHandler) CreateContribution(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		AmountPLN float64 `json:"amountPLN"`
		Notes     *string `json:"notes"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.supplyService.CreateManualContribution(c.Context(), userID, req.AmountPLN, req.Notes); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Contribution added successfully",
	})
}

// ========== Stats Handler ==========

// GetStats retrieves spending statistics
func (h *SupplyHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.supplyService.GetStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}
