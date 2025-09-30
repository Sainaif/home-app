package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/sainaif/holy-home/internal/services"
)

type ExportHandler struct {
	exportService *services.ExportService
}

func NewExportHandler(exportService *services.ExportService) *ExportHandler {
	return &ExportHandler{exportService: exportService}
}

// ExportBills exports bills to CSV
func (h *ExportHandler) ExportBills(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse query parameters
	var billType *string
	if t := c.Query("type"); t != "" {
		billType = &t
	}

	var from, to *time.Time
	if f := c.Query("from"); f != "" {
		if parsed, err := time.Parse("2006-01-02", f); err == nil {
			from = &parsed
		}
	}
	if t := c.Query("to"); t != "" {
		if parsed, err := time.Parse("2006-01-02", t); err == nil {
			to = &parsed
		}
	}

	// Export CSV
	csv, err := h.exportService.ExportBillsCSV(ctx, billType, from, to)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=bills.csv")
	return c.Send(csv)
}

// ExportBalances exports loan balances to CSV
func (h *ExportHandler) ExportBalances(c *fiber.Ctx) error {
	ctx := c.Context()

	// Export CSV
	csv, err := h.exportService.ExportBalancesCSV(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=balances.csv")
	return c.Send(csv)
}

// ExportChores exports chore assignments to CSV
func (h *ExportHandler) ExportChores(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse query parameters
	var userID *primitive.ObjectID
	if u := c.Query("user_id"); u != "" {
		if parsed, err := primitive.ObjectIDFromHex(u); err == nil {
			userID = &parsed
		}
	}

	var status *string
	if s := c.Query("status"); s != "" {
		status = &s
	}

	// Export CSV
	csv, err := h.exportService.ExportChoresCSV(ctx, userID, status)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=chores.csv")
	return c.Send(csv)
}

// ExportConsumptions exports consumption history to CSV
func (h *ExportHandler) ExportConsumptions(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse query parameters
	var userID *primitive.ObjectID
	if u := c.Query("user_id"); u != "" {
		if parsed, err := primitive.ObjectIDFromHex(u); err == nil {
			userID = &parsed
		}
	}

	var from, to *time.Time
	if f := c.Query("from"); f != "" {
		if parsed, err := time.Parse("2006-01-02", f); err == nil {
			from = &parsed
		}
	}
	if t := c.Query("to"); t != "" {
		if parsed, err := time.Parse("2006-01-02", t); err == nil {
			to = &parsed
		}
	}

	// Export CSV
	csv, err := h.exportService.ExportConsumptionsCSV(ctx, userID, from, to)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=consumptions.csv")
	return c.Send(csv)
}