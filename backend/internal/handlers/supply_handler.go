package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
)

type SupplyHandler struct {
	supplyService *services.SupplyService
	auditService  *services.AuditService
	eventService  *services.EventService
}

func NewSupplyHandler(supplyService *services.SupplyService, auditService *services.AuditService, eventService *services.EventService) *SupplyHandler {
	return &SupplyHandler{
		supplyService: supplyService,
		auditService:  auditService,
		eventService:  eventService,
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

// GetItems retrieves supply items with optional filters and sorting
func (h *SupplyHandler) GetItems(c *fiber.Ctx) error {
	filterParam := c.Query("filter")
	var filter *string
	if filterParam != "" {
		filter = &filterParam
	}

	sortParam := c.Query("sort")
	var sort *string
	if sortParam != "" {
		sort = &sortParam
	}

	items, err := h.supplyService.GetItems(c.Context(), filter, sort)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(items)
}

// CreateItem adds a new supply item with initial inventory
func (h *SupplyHandler) CreateItem(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}
	userEmail, err := middleware.GetUserEmail(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		Name            string  `json:"name"`
		Category        string  `json:"category"`
		CurrentQuantity int     `json:"currentQuantity"`
		MinQuantity     int     `json:"minQuantity"`
		Unit            string  `json:"unit"`
		Priority        int     `json:"priority"`
		Notes           *string `json:"notes"`
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

	// Default unit to pcs if not specified
	if req.Unit == "" {
		req.Unit = "pcs"
	}

	item, err := h.supplyService.CreateItem(c.Context(), userID, req.Name, req.Category, req.CurrentQuantity, req.MinQuantity, req.Unit, req.Priority, req.Notes)
	if err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_supply_item", "supply", nil,
			map[string]interface{}{"name": req.Name, "error": err.Error()},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_supply_item", "supply", &item.ID,
		map[string]interface{}{"name": req.Name, "category": req.Category, "quantity": req.CurrentQuantity},
		c.IP(), c.Get("User-Agent"), "success")

	// Broadcast event to all users
	h.eventService.Broadcast(services.EventSupplyItemAdded, map[string]interface{}{
		"itemId":   item.ID,
		"name":     req.Name,
		"category": req.Category,
		"addedBy":  userEmail,
	})

	return c.Status(fiber.StatusCreated).JSON(item)
}

// UpdateItem updates item details
func (h *SupplyHandler) UpdateItem(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}
	userEmail, err := middleware.GetUserEmail(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	itemID := c.Params("id")
	if itemID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid item ID",
		})
	}

	var req struct {
		Name        *string `json:"name"`
		Category    *string `json:"category"`
		MinQuantity *int    `json:"minQuantity"`
		Unit        *string `json:"unit"`
		Priority    *int    `json:"priority"`
		Notes       *string `json:"notes"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.supplyService.UpdateItem(c.Context(), itemID, req.Name, req.Category, req.MinQuantity, req.Unit, req.Priority, req.Notes); err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "update_supply_item", "supply", &itemID,
			map[string]interface{}{"error": err.Error()},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "update_supply_item", "supply", &itemID,
		map[string]interface{}{"changes": req},
		c.IP(), c.Get("User-Agent"), "success")

	return c.JSON(fiber.Map{
		"message": "Item updated successfully",
	})
}

// RestockItem increases item quantity
func (h *SupplyHandler) RestockItem(c *fiber.Ctx) error {
	itemID := c.Params("id")
	if itemID == "" {
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
	userEmail, err := middleware.GetUserEmail(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		QuantityToAdd int      `json:"quantityToAdd"`
		AmountPLN     *float64 `json:"amountPLN"`
		NeedsRefund   bool     `json:"needsRefund"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.supplyService.RestockItem(c.Context(), itemID, userID, req.QuantityToAdd, req.AmountPLN, req.NeedsRefund); err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "restock_supply_item", "supply", &itemID,
			map[string]interface{}{"quantity": req.QuantityToAdd, "error": err.Error()},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "restock_supply_item", "supply", &itemID,
		map[string]interface{}{"quantity": req.QuantityToAdd, "amount": req.AmountPLN, "needs_refund": req.NeedsRefund},
		c.IP(), c.Get("User-Agent"), "success")

	return c.JSON(fiber.Map{
		"message": "Item restocked successfully",
	})
}

// ConsumeItem reduces item quantity
func (h *SupplyHandler) ConsumeItem(c *fiber.Ctx) error {
	itemID := c.Params("id")
	if itemID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid item ID",
		})
	}

	var req struct {
		QuantityToSubtract int `json:"quantityToSubtract"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.supplyService.ConsumeItem(c.Context(), itemID, req.QuantityToSubtract); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Item consumed successfully",
	})
}

// SetQuantity directly sets item quantity
func (h *SupplyHandler) SetQuantity(c *fiber.Ctx) error {
	itemID := c.Params("id")
	if itemID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid item ID",
		})
	}

	var req struct {
		NewQuantity int `json:"newQuantity"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.supplyService.SetQuantity(c.Context(), itemID, req.NewQuantity); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Quantity updated successfully",
	})
}

// MarkAsRefunded marks an item's restock as refunded
func (h *SupplyHandler) MarkAsRefunded(c *fiber.Ctx) error {
	itemID := c.Params("id")
	if itemID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid item ID",
		})
	}

	if err := h.supplyService.MarkAsRefunded(c.Context(), itemID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Marked as refunded successfully",
	})
}

// DeleteItem deletes an item
func (h *SupplyHandler) DeleteItem(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}
	userEmail, err := middleware.GetUserEmail(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	itemID := c.Params("id")
	if itemID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid item ID",
		})
	}

	if err := h.supplyService.DeleteItem(c.Context(), itemID); err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "delete_supply_item", "supply", &itemID,
			map[string]interface{}{"error": err.Error()},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "delete_supply_item", "supply", &itemID,
		map[string]interface{}{},
		c.IP(), c.Get("User-Agent"), "success")

	return c.JSON(fiber.Map{
		"message": "Item deleted successfully",
	})
}

// ========== Contribution Handlers ==========

// GetContributions retrieves contributions
func (h *SupplyHandler) GetContributions(c *fiber.Ctx) error {
	var userID *string
	userIDParam := c.Query("userId")
	if userIDParam != "" {
		userID = &userIDParam
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
