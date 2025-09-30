package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/services"
)

type BackupHandler struct {
	backupService *services.BackupService
}

func NewBackupHandler(backupService *services.BackupService) *BackupHandler {
	return &BackupHandler{
		backupService: backupService,
	}
}

// ExportBackup exports all data as JSON (ADMIN only)
func (h *BackupHandler) ExportBackup(c *fiber.Ctx) error {
	jsonData, err := h.backupService.ExportJSON(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Set headers for file download
	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition", "attachment; filename=holy-home-backup.json")

	return c.Send(jsonData)
}

// ImportBackup imports all data from JSON (ADMIN only, DANGEROUS)
func (h *BackupHandler) ImportBackup(c *fiber.Ctx) error {
	// Read the uploaded JSON file
	jsonData := c.Body()

	if len(jsonData) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Empty backup file",
		})
	}

	// Import the backup
	if err := h.backupService.ImportJSON(c.Context(), jsonData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Backup imported successfully",
	})
}
