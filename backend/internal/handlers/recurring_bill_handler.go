package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/services"
	"github.com/sainaif/holy-home/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RecurringBillTemplateRequest struct {
	CustomType  string                           `json:"customType"`
	Frequency   string                           `json:"frequency"`
	Amount      string                           `json:"amount"` // Comes as string from JSON
	DayOfMonth  int                              `json:"dayOfMonth"`
	StartDate   time.Time                        `json:"startDate"` // Required
	Allocations []models.RecurringBillAllocation `json:"allocations"`
	Notes       *string                          `json:"notes,omitempty"`
}

type RecurringBillHandler struct {
	recurringBillService *services.RecurringBillService
	auditService         *services.AuditService
}

func NewRecurringBillHandler(recurringBillService *services.RecurringBillService, auditService *services.AuditService) *RecurringBillHandler {
	return &RecurringBillHandler{
		recurringBillService: recurringBillService,
		auditService:         auditService,
	}
}

// CreateRecurringBillTemplate creates a new recurring bill template (ADMIN only)
func (h *RecurringBillHandler) CreateRecurringBillTemplate(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)
	userEmail := c.Locals("userEmail").(string)

	var req RecurringBillTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid request body: %v", err),
		})
	}

	// Convert amount string to Decimal128
	amountFloat, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid amount: %v", err),
		})
	}
	amountDecimal, err := utils.DecimalFromFloat(amountFloat)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to convert amount: %v", err),
		})
	}

	// Build template model
	template := &models.RecurringBillTemplate{
		CustomType:  req.CustomType,
		Frequency:   req.Frequency,
		Amount:      amountDecimal,
		DayOfMonth:  req.DayOfMonth,
		StartDate:   req.StartDate,
		Allocations: req.Allocations,
		Notes:       req.Notes,
	}

	if err := h.recurringBillService.CreateTemplate(c.Context(), template); err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_recurring_bill_template", "recurring_bill_template", nil,
			map[string]interface{}{"custom_type": template.CustomType, "frequency": template.Frequency},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_recurring_bill_template", "recurring_bill_template", &template.ID,
		map[string]interface{}{"custom_type": template.CustomType, "frequency": template.Frequency, "amount": template.Amount},
		c.IP(), c.Get("User-Agent"), "success")

	return c.Status(fiber.StatusCreated).JSON(template)
}

// GetRecurringBillTemplates retrieves all recurring bill templates
func (h *RecurringBillHandler) GetRecurringBillTemplates(c *fiber.Ctx) error {
	templates, err := h.recurringBillService.ListTemplates(c.Context())
	if err != nil {
		fmt.Printf("Error listing recurring bill templates: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if templates == nil {
		templates = []models.RecurringBillTemplate{}
	}

	return c.JSON(templates)
}

// GetRecurringBillTemplate retrieves a specific recurring bill template
func (h *RecurringBillHandler) GetRecurringBillTemplate(c *fiber.Ctx) error {
	id := c.Params("id")
	templateID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template ID",
		})
	}

	template, err := h.recurringBillService.GetTemplate(c.Context(), templateID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(template)
}

// UpdateRecurringBillTemplate updates an existing template (ADMIN only)
func (h *RecurringBillHandler) UpdateRecurringBillTemplate(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)
	userEmail := c.Locals("userEmail").(string)

	id := c.Params("id")
	templateID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template ID",
		})
	}

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Convert amount to Decimal128 if present
	if amountStr, ok := updates["amount"].(string); ok {
		amountFloat, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid amount: %v", err),
			})
		}
		amountDecimal, err := utils.DecimalFromFloat(amountFloat)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to convert amount: %v", err),
			})
		}
		updates["amount"] = amountDecimal
	}

	if err := h.recurringBillService.UpdateTemplate(c.Context(), templateID, updates); err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "update_recurring_bill_template", "recurring_bill_template", &templateID,
			map[string]interface{}{"updates": updates},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "update_recurring_bill_template", "recurring_bill_template", &templateID,
		map[string]interface{}{"updates": updates},
		c.IP(), c.Get("User-Agent"), "success")

	return c.JSON(fiber.Map{
		"message": "Template updated successfully",
	})
}

// DeleteRecurringBillTemplate deletes a template (ADMIN only)
func (h *RecurringBillHandler) DeleteRecurringBillTemplate(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)
	userEmail := c.Locals("userEmail").(string)

	id := c.Params("id")
	templateID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid template ID",
		})
	}

	if err := h.recurringBillService.DeleteTemplate(c.Context(), templateID); err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "delete_recurring_bill_template", "recurring_bill_template", &templateID,
			nil, c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "delete_recurring_bill_template", "recurring_bill_template", &templateID,
		nil, c.IP(), c.Get("User-Agent"), "success")

	return c.JSON(fiber.Map{
		"message": "Template deleted successfully",
	})
}

// GenerateRecurringBills manually triggers generation of bills from templates (ADMIN only)
func (h *RecurringBillHandler) GenerateRecurringBills(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)
	userEmail := c.Locals("userEmail").(string)

	if err := h.recurringBillService.GenerateBillsFromTemplates(c.Context()); err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "generate_recurring_bills", "recurring_bill_template", nil,
			nil, c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "generate_recurring_bills", "recurring_bill_template", nil,
		nil, c.IP(), c.Get("User-Agent"), "success")

	return c.JSON(fiber.Map{
		"message": "Recurring bills generated successfully",
	})
}
