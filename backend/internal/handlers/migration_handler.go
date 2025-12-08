package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/services"
)

// MigrationHandler handles MongoDB to SQLite migration endpoints
type MigrationHandler struct {
	migrationService *services.MigrationService
}

// NewMigrationHandler creates a new migration handler
func NewMigrationHandler(migrationService *services.MigrationService) *MigrationHandler {
	return &MigrationHandler{
		migrationService: migrationService,
	}
}

// GetMigrationStatus returns the current migration status
// GET /migrate/status - Public endpoint (used to check if migration mode is enabled)
func (h *MigrationHandler) GetMigrationStatus(c *fiber.Ctx) error {
	status, err := h.migrationService.GetStatus(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(status)
}

// ImportFromMongoDB imports data from a MongoDB backup JSON
// POST /migrate/import - Admin only
func (h *MigrationHandler) ImportFromMongoDB(c *fiber.Ctx) error {
	// Check if there's existing data (optional safety check)
	status, err := h.migrationService.GetStatus(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check migration status: " + err.Error(),
		})
	}

	// Warn if there's existing data but allow override with query param
	if status.HasExistingData {
		allowOverwrite := c.Query("overwrite", "false")
		if allowOverwrite != "true" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error":           "Database already contains data. Use ?overwrite=true to proceed.",
				"hasExistingData": true,
			})
		}
	}

	// Read the uploaded JSON file
	jsonData := c.Body()
	if len(jsonData) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Empty backup file",
		})
	}

	// Perform the migration
	result, err := h.migrationService.ImportFromJSON(c.Context(), jsonData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  err.Error(),
			"result": result,
		})
	}

	if !result.Success {
		return c.Status(fiber.StatusPartialContent).JSON(fiber.Map{
			"message": "Migration completed with errors",
			"result":  result,
		})
	}

	return c.JSON(fiber.Map{
		"message": "Migration completed successfully",
		"result":  result,
	})
}
