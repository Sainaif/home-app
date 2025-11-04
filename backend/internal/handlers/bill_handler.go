package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BillHandler struct {
	billService        *services.BillService
	consumptionService *services.ConsumptionService
	allocationService  *services.AllocationService
	auditService       *services.AuditService
	eventService       *services.EventService
}

func NewBillHandler(billService *services.BillService, consumptionService *services.ConsumptionService, allocationService *services.AllocationService, auditService *services.AuditService, eventService *services.EventService) *BillHandler {
	return &BillHandler{
		billService:        billService,
		consumptionService: consumptionService,
		allocationService:  allocationService,
		auditService:       auditService,
		eventService:       eventService,
	}
}

// CreateBill creates a new bill (ADMIN only)
func (h *BillHandler) CreateBill(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)
	userEmail := c.Locals("userEmail").(string)

	var req services.CreateBillRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	bill, err := h.billService.CreateBill(c.Context(), req)
	if err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_bill", "bill", nil,
			map[string]interface{}{"type": req.Type, "amount": req.TotalAmountPLN},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	auditDetails := map[string]interface{}{"type": bill.Type, "amount": req.TotalAmountPLN, "period_start": req.PeriodStart, "period_end": req.PeriodEnd}
	if bill.CustomType != nil {
		auditDetails["custom_type"] = *bill.CustomType
	}
	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_bill", "bill", &bill.ID,
		auditDetails,
		c.IP(), c.Get("User-Agent"), "success")

	// Broadcast event to all users
	h.eventService.Broadcast(services.EventBillCreated, map[string]interface{}{
		"billId":      bill.ID.Hex(),
		"type":        bill.Type,
		"amount":      req.TotalAmountPLN,
		"createdBy":   userEmail,
		"periodStart": req.PeriodStart,
		"periodEnd":   req.PeriodEnd,
	})

	return c.Status(fiber.StatusCreated).JSON(bill)
}

// GetBills retrieves bills with optional filters
func (h *BillHandler) GetBills(c *fiber.Ctx) error {
	billType := c.Query("type")
	fromStr := c.Query("from")
	toStr := c.Query("to")

	var billTypePtr *string
	if billType != "" {
		billTypePtr = &billType
	}

	var fromPtr, toPtr *time.Time
	if fromStr != "" {
		from, err := time.Parse(time.RFC3339, fromStr)
		if err == nil {
			fromPtr = &from
		}
	}
	if toStr != "" {
		to, err := time.Parse(time.RFC3339, toStr)
		if err == nil {
			toPtr = &to
		}
	}

	bills, err := h.billService.GetBills(c.Context(), billTypePtr, fromPtr, toPtr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(bills)
}

// GetBill retrieves a specific bill
func (h *BillHandler) GetBill(c *fiber.Ctx) error {
	id := c.Params("id")
	billID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid bill ID",
		})
	}

	bill, err := h.billService.GetBill(c.Context(), billID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(bill)
}

// PostBill changes status to posted (ADMIN only)
func (h *BillHandler) PostBill(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)
	userEmail := c.Locals("userEmail").(string)

	id := c.Params("id")
	billID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid bill ID",
		})
	}

	// Get bill info for audit
	bill, _ := h.billService.GetBill(c.Context(), billID)

	if err := h.billService.PostBill(c.Context(), billID); err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "post_bill", "bill", &billID,
			map[string]interface{}{"bill_type": bill.Type, "status": "draft"},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	postAuditDetails := map[string]interface{}{"bill_type": bill.Type, "old_status": "draft", "new_status": "posted"}
	if bill.CustomType != nil {
		postAuditDetails["custom_type"] = *bill.CustomType
	}
	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "post_bill", "bill", &billID,
		postAuditDetails,
		c.IP(), c.Get("User-Agent"), "success")

	// Broadcast bill posted event to all users
	billType := bill.Type
	if bill.CustomType != nil && *bill.CustomType != "" {
		billType = *bill.CustomType
	}

	totalAmount, _ := bill.TotalAmountPLN.MarshalJSON()
	h.eventService.Broadcast(services.EventBillPosted, map[string]interface{}{
		"billId":    bill.ID.Hex(),
		"type":      billType,
		"amount":    string(totalAmount),
		"postedBy":  userEmail,
		"periodEnd": bill.PeriodEnd.Format("2006-01-02"),
	})

	return c.JSON(fiber.Map{
		"message": "Bill posted successfully",
	})
}

// CloseBill changes status to closed (ADMIN only)
func (h *BillHandler) CloseBill(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)
	userEmail := c.Locals("userEmail").(string)

	id := c.Params("id")
	billID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid bill ID",
		})
	}

	// Get bill info for audit
	bill, _ := h.billService.GetBill(c.Context(), billID)

	if err := h.billService.CloseBill(c.Context(), billID); err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "close_bill", "bill", &billID,
			map[string]interface{}{"bill_type": bill.Type, "status": "posted"},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	closeAuditDetails := map[string]interface{}{"bill_type": bill.Type, "old_status": "posted", "new_status": "closed"}
	if bill.CustomType != nil {
		closeAuditDetails["custom_type"] = *bill.CustomType
	}
	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "close_bill", "bill", &billID,
		closeAuditDetails,
		c.IP(), c.Get("User-Agent"), "success")

	return c.JSON(fiber.Map{
		"message": "Bill closed successfully",
	})
}

// ReopenBill reopens a bill to a previous status (ADMIN only)
func (h *BillHandler) ReopenBill(c *fiber.Ctx) error {
	id := c.Params("id")
	billID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid bill ID",
		})
	}

	// Get user ID from context
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		TargetStatus string `json:"targetStatus"`
		Reason       string `json:"reason"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.billService.ReopenBill(c.Context(), billID, userID, req.TargetStatus, req.Reason); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Bill reopened successfully",
	})
}

// CreateConsumption records a consumption reading
func (h *BillHandler) CreateConsumption(c *fiber.Ctx) error {
	// Get user ID from context
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	userEmail := c.Locals("userEmail").(string)

	var req services.CreateConsumptionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Set user ID from authenticated user
	req.UserID = userID

	// Determine source based on user role
	role, _ := middleware.GetUserRole(c)
	source := "user"
	if role == "ADMIN" {
		source = "admin"
	}

	// Get bill info for audit
	bill, _ := h.billService.GetBill(c.Context(), req.BillID)

	consumption, err := h.consumptionService.CreateConsumption(c.Context(), req, source)
	if err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_reading", "consumption", nil,
			map[string]interface{}{"bill_id": req.BillID.Hex(), "bill_type": bill.Type, "meter_value": req.MeterValue},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_reading", "consumption", &consumption.ID,
		map[string]interface{}{"bill_id": req.BillID.Hex(), "bill_type": bill.Type, "meter_value": req.MeterValue, "source": source},
		c.IP(), c.Get("User-Agent"), "success")

	// Broadcast event to all users
	h.eventService.Broadcast(services.EventConsumptionCreated, map[string]interface{}{
		"consumptionId": consumption.ID.Hex(),
		"billId":        req.BillID.Hex(),
		"billType":      bill.Type,
		"meterValue":    req.MeterValue,
		"createdBy":     userEmail,
	})

	return c.Status(fiber.StatusCreated).JSON(consumption)
}

// GetConsumptions retrieves consumptions for a bill (or all consumptions if no billId)
func (h *BillHandler) GetConsumptions(c *fiber.Ctx) error {
	billIDStr := c.Query("billId")

	var billID *primitive.ObjectID
	if billIDStr != "" {
		id, err := primitive.ObjectIDFromHex(billIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid bill ID",
			})
		}
		billID = &id
	}

	consumptions, err := h.consumptionService.GetConsumptions(c.Context(), billID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(consumptions)
}

// DeleteBill deletes a bill (ADMIN only)
func (h *BillHandler) DeleteBill(c *fiber.Ctx) error {
	userID := c.Locals("userId").(primitive.ObjectID)
	userEmail := c.Locals("userEmail").(string)

	id := c.Params("id")
	billID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid bill ID",
		})
	}

	// Get bill details before deletion for audit trail
	bill, err := h.billService.GetBill(c.Context(), billID)
	if err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "bill.delete", "bill", &billID,
			map[string]interface{}{"error": "Bill not found"}, c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Bill not found",
		})
	}

	if err := h.billService.DeleteBill(c.Context(), billID); err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "bill.delete", "bill", &billID,
			map[string]interface{}{
				"billType": bill.Type,
				"error":    err.Error(),
			}, c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Log successful deletion
	billType := bill.Type
	if bill.CustomType != nil && *bill.CustomType != "" {
		billType = *bill.CustomType
	}
	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "bill.delete", "bill", &billID,
		map[string]interface{}{
			"billType":    billType,
			"periodStart": bill.PeriodStart.Format("2006-01-02"),
			"periodEnd":   bill.PeriodEnd.Format("2006-01-02"),
			"status":      bill.Status,
		}, c.IP(), c.Get("User-Agent"), "success")

	return c.JSON(fiber.Map{"message": "Bill deleted successfully"})
}

// DeleteConsumption deletes a consumption/reading (ADMIN only)
func (h *BillHandler) DeleteConsumption(c *fiber.Ctx) error {
	id := c.Params("id")
	consumptionID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid consumption ID",
		})
	}

	if err := h.consumptionService.DeleteConsumption(c.Context(), consumptionID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"message": "Consumption deleted successfully"})
}

// MarkConsumptionInvalid marks a consumption as invalid (user can mark their own)
func (h *BillHandler) MarkConsumptionInvalid(c *fiber.Ctx) error {
	id := c.Params("id")
	consumptionID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid consumption ID",
		})
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := h.consumptionService.MarkConsumptionInvalid(c.Context(), consumptionID, userID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"message": "Consumption marked as invalid"})
}

// GetBillAllocation returns allocation breakdown for a bill
func (h *BillHandler) GetBillAllocation(c *fiber.Ctx) error {
	id := c.Params("id")
	billID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid bill ID",
		})
	}

	breakdown, err := h.allocationService.GetAllocationBreakdown(c.Context(), billID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(breakdown)
}

// GetBillPaymentStatus returns payment status showing who paid and who hasn't
func (h *BillHandler) GetBillPaymentStatus(c *fiber.Ctx) error {
	billID, err := primitive.ObjectIDFromHex(c.Params("billId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid bill ID",
		})
	}

	status, err := h.billService.GetBillPaymentStatus(c.Context(), billID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(status)
}
