package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/services"
	"github.com/sainaif/holy-home/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PaymentHandler struct {
	db                   *mongo.Database
	auditService         *services.AuditService
	recurringBillService *services.RecurringBillService
}

func NewPaymentHandler(db *mongo.Database, auditService *services.AuditService, recurringBillService *services.RecurringBillService) *PaymentHandler {
	return &PaymentHandler{
		db:                   db,
		auditService:         auditService,
		recurringBillService: recurringBillService,
	}
}

type RecordPaymentRequest struct {
	BillID string  `json:"billId"`
	Amount string  `json:"amount"`
	Method *string `json:"method,omitempty"`
}

// RecordPayment records a payment made by the current user for a bill
func (h *PaymentHandler) RecordPayment(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)
	userEmail := c.Locals("userEmail").(string)

	var req RecordPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate bill ID
	billID, err := primitive.ObjectIDFromHex(req.BillID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid bill ID",
		})
	}

	// Check if bill exists and is posted
	var bill models.Bill
	err = h.db.Collection("bills").FindOne(c.Context(), bson.M{"_id": billID}).Decode(&bill)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Bill not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch bill",
		})
	}

	if bill.Status != "posted" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Can only record payments for posted bills",
		})
	}

	// Convert amount to Decimal128
	amountFloat, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid amount format",
		})
	}

	amountDecimal, err := utils.DecimalFromFloat(amountFloat)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to convert amount",
		})
	}

	// Create payment record
	payment := models.Payment{
		BillID:      billID,
		PayerUserID: userID,
		AmountPLN:   amountDecimal,
		PaidAt:      time.Now(),
		Method:      req.Method,
	}

	result, err := h.db.Collection("payments").InsertOne(c.Context(), payment)
	if err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "record_payment", "payment", nil,
			map[string]interface{}{"bill_id": billID, "amount": amountFloat},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to record payment",
		})
	}

	payment.ID = result.InsertedID.(primitive.ObjectID)

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "record_payment", "payment", &payment.ID,
		map[string]interface{}{"bill_id": billID, "amount": amountFloat},
		c.IP(), c.Get("User-Agent"), "success")

	// Check if this payment completes a recurring bill and generate next bill if needed
	if h.recurringBillService != nil {
		if err := h.recurringBillService.CheckAndGenerateNextBill(c.Context(), billID); err != nil {
			// Log the error but don't fail the payment
			// The next bill can be generated manually if needed
			// TODO: consider adding this to a job queue instead
		}
	}

	return c.Status(fiber.StatusCreated).JSON(payment)
}

// GetBillPayments returns all payments for a specific bill
func (h *PaymentHandler) GetBillPayments(c *fiber.Ctx) error {
	billIDStr := c.Params("billId")
	billID, err := primitive.ObjectIDFromHex(billIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid bill ID",
		})
	}

	cursor, err := h.db.Collection("payments").Find(c.Context(), bson.M{"bill_id": billID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch payments",
		})
	}
	defer cursor.Close(c.Context())

	var payments []models.Payment
	if err := cursor.All(c.Context(), &payments); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode payments",
		})
	}

	if payments == nil {
		payments = []models.Payment{}
	}

	return c.JSON(payments)
}

// GetUserPayments returns all payments made by the current user
func (h *PaymentHandler) GetUserPayments(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)

	cursor, err := h.db.Collection("payments").Find(c.Context(), bson.M{"payer_user_id": userID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch payments",
		})
	}
	defer cursor.Close(c.Context())

	var payments []models.Payment
	if err := cursor.All(c.Context(), &payments); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode payments",
		})
	}

	if payments == nil {
		payments = []models.Payment{}
	}

	return c.JSON(payments)
}
