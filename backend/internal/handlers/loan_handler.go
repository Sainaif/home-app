package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
)

type LoanHandler struct {
	loanService  *services.LoanService
	eventService *services.EventService
	auditService *services.AuditService
}

func NewLoanHandler(loanService *services.LoanService, eventService *services.EventService, auditService *services.AuditService) *LoanHandler {
	return &LoanHandler{
		loanService:  loanService,
		eventService: eventService,
		auditService: auditService,
	}
}

// CreateLoan creates a new loan
func (h *LoanHandler) CreateLoan(c *fiber.Ctx) error {
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

	var req services.CreateLoanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	loan, err := h.loanService.CreateLoan(c.Context(), req)
	if err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_loan", "loan", nil,
			map[string]interface{}{"error": err.Error()},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_loan", "loan", &loan.ID,
		map[string]interface{}{"lender_id": req.LenderID, "borrower_id": req.BorrowerID, "amount": req.AmountPLN},
		c.IP(), c.Get("User-Agent"), "success")

	// Broadcast loan created event
	h.eventService.Broadcast(services.EventLoanCreated, map[string]interface{}{
		"loan_id": loan.ID,
	})

	// Broadcast balance updated event
	h.eventService.Broadcast(services.EventBalanceUpdated, map[string]interface{}{
		"timestamp": loan.CreatedAt,
	})

	return c.Status(fiber.StatusCreated).JSON(loan)
}

// CreateLoanPayment records a loan repayment
func (h *LoanHandler) CreateLoanPayment(c *fiber.Ctx) error {
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

	var req services.CreateLoanPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	payment, err := h.loanService.CreateLoanPayment(c.Context(), req)
	if err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_loan_payment", "loan", nil,
			map[string]interface{}{"error": err.Error()},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "create_loan_payment", "loan", &payment.ID,
		map[string]interface{}{"loan_id": req.LoanID, "amount": req.AmountPLN},
		c.IP(), c.Get("User-Agent"), "success")

	// Broadcast loan payment created event
	h.eventService.Broadcast(services.EventLoanPaymentCreated, map[string]interface{}{
		"payment_id": payment.ID,
		"loan_id":    payment.LoanID,
	})

	// Broadcast balance updated event
	h.eventService.Broadcast(services.EventBalanceUpdated, map[string]interface{}{
		"timestamp": payment.PaidAt,
	})

	return c.Status(fiber.StatusCreated).JSON(payment)
}

// GetLoans retrieves all loans with optional sorting and pagination
func (h *LoanHandler) GetLoans(c *fiber.Ctx) error {
	// Parse query parameters
	sortBy := c.Query("sort", "createdAt")
	order := c.Query("order", "desc")
	limit := c.QueryInt("limit", 0)
	offset := c.QueryInt("offset", 0)

	opts := services.GetLoansOptions{
		SortBy: sortBy,
		Order:  order,
		Limit:  limit,
		Offset: offset,
	}

	loans, err := h.loanService.GetLoansWithOptions(c.Context(), opts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(loans)
}

// GetBalances retrieves pairwise balances
func (h *LoanHandler) GetBalances(c *fiber.Ctx) error {
	balances, err := h.loanService.GetBalances(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(balances)
}

// GetMyBalance retrieves the current user's balance
func (h *LoanHandler) GetMyBalance(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	balance, err := h.loanService.GetUserBalance(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(balance)
}

// GetUserBalance retrieves a specific user's balance (ADMIN)
func (h *LoanHandler) GetUserBalance(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	balance, err := h.loanService.GetUserBalance(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(balance)
}

// GetLoanPayments retrieves all payments for a specific loan
func (h *LoanHandler) GetLoanPayments(c *fiber.Ctx) error {
	loanID := c.Params("id")
	if loanID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid loan ID",
		})
	}

	payments, err := h.loanService.GetLoanPayments(c.Context(), loanID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(payments)
}

// DeleteLoan deletes a loan (ADMIN only)
func (h *LoanHandler) DeleteLoan(c *fiber.Ctx) error {
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

	loanID := c.Params("id")
	if loanID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid loan ID",
		})
	}

	err = h.loanService.DeleteLoan(c.Context(), loanID)
	if err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "delete_loan", "loan", &loanID,
			map[string]interface{}{"error": err.Error()},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "delete_loan", "loan", &loanID,
		map[string]interface{}{},
		c.IP(), c.Get("User-Agent"), "success")

	// Broadcast loan deleted event
	h.eventService.Broadcast(services.EventLoanDeleted, map[string]interface{}{
		"loan_id": loanID,
	})

	// Broadcast balance updated event
	h.eventService.Broadcast(services.EventBalanceUpdated, map[string]interface{}{
		"timestamp": time.Now(),
	})

	return c.JSON(fiber.Map{
		"message": "Loan deleted successfully",
	})
}

// CompensateLoan manually triggers group debt compensation
func (h *LoanHandler) CompensateLoan(c *fiber.Ctx) error {
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

	result, err := h.loanService.PerformGroupCompensation(c.Context())
	if err != nil {
		h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "compensate_loans", "loan", nil,
			map[string]interface{}{"error": err.Error()},
			c.IP(), c.Get("User-Agent"), "failure")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	h.auditService.LogAction(c.Context(), userID, userEmail, userEmail, "compensate_loans", "loan", nil,
		map[string]interface{}{
			"compensations_performed": result.CompensationsPerformed,
			"total_amount":            result.TotalAmountCompensated,
		},
		c.IP(), c.Get("User-Agent"), "success")

	// Broadcast balance updated event if compensations were performed
	if result.CompensationsPerformed > 0 {
		h.eventService.Broadcast(services.EventBalanceUpdated, map[string]interface{}{
			"timestamp": time.Now(),
		})
	}

	return c.JSON(result)
}
