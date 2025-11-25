package handlers

import (
	"context"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/services"
)

type OCRHandler struct {
	ocrService *services.OCRService
}

func NewOCRHandler(ocr *services.OCRService) *OCRHandler {
	return &OCRHandler{ocrService: ocr}
}

func (h *OCRHandler) HandleOCR(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file missing"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "cannot open file"})
	}
	defer file.Close()

	imgBytes, _ := io.ReadAll(file)

	result, err := h.ocrService.ParseInvoice(context.Background(), imgBytes)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}
