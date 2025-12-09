package handlers

import (
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/services"
)

type PaymentHandler struct {
	paymentService *services.PaymentService
	auditService   *services.AuditService
}

func NewPaymentHandler(paymentService *services.PaymentService, auditService *services.AuditService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		auditService:   auditService,
	}
}

type RecordPaymentRequest struct {
	BillID string  `json:"billId"`
	Amount string  `json:"amount"`
	Method *string `json:"method,omitempty"`
}

// RecordPayment records a payment made by the current user for a bill
func (h *PaymentHandler) RecordPayment(c *fiber.Ctx) error {
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

	var req RecordPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate bill ID
	if req.BillID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid bill ID",
		})
	}

	// Parse amount
	amountFloat, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid amount format",
		})
	}

	// Record the payment
	payment, err := h.paymentService.RecordPayment(c.Context(), services.RecordPaymentRequest{
		BillID: req.BillID,
		Amount: amountFloat,
		Method: req.Method,
	}, userID)

	if err != nil {
		log.Printf("Payment error for bill %s, user %s: %v", req.BillID, userID, err)
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "record_payment", "payment", nil,
			map[string]interface{}{"bill_id": req.BillID, "amount": amountFloat},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "record_payment", "payment", &payment.ID,
		map[string]interface{}{"bill_id": req.BillID, "amount": amountFloat},
		c.IP(), c.Get("User-Agent"), "success")

	return c.Status(fiber.StatusCreated).JSON(payment)
}

// GetBillPayments returns all payments for a specific bill
func (h *PaymentHandler) GetBillPayments(c *fiber.Ctx) error {
	billID := c.Params("billId")
	if billID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid bill ID",
		})
	}

	payments, err := h.paymentService.GetBillPayments(c.Context(), billID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch payments",
		})
	}

	if payments == nil {
		payments = []models.Payment{}
	}

	return c.JSON(payments)
}

// GetUserPayments returns all payments made by the current user
func (h *PaymentHandler) GetUserPayments(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	payments, err := h.paymentService.GetUserPayments(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch payments",
		})
	}

	if payments == nil {
		payments = []models.Payment{}
	}

	return c.JSON(payments)
}
