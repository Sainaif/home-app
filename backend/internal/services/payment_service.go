package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
	"github.com/sainaif/holy-home/internal/utils"
)

type PaymentService struct {
	payments             repository.PaymentRepository
	bills                repository.BillRepository
	recurringBillService *RecurringBillService
}

func NewPaymentService(
	payments repository.PaymentRepository,
	bills repository.BillRepository,
	recurringBillService *RecurringBillService,
) *PaymentService {
	return &PaymentService{
		payments:             payments,
		bills:                bills,
		recurringBillService: recurringBillService,
	}
}

type RecordPaymentRequest struct {
	BillID string  `json:"billId"`
	Amount float64 `json:"amount"`
	Method *string `json:"method,omitempty"`
}

// RecordPayment records a payment made by a user for a bill
func (s *PaymentService) RecordPayment(ctx context.Context, req RecordPaymentRequest, userID string) (*models.Payment, error) {
	// Validate amount is positive
	if req.Amount <= 0 {
		return nil, fmt.Errorf("payment amount must be positive")
	}

	// Verify bill exists and is posted
	bill, err := s.bills.GetByID(ctx, req.BillID)
	if err != nil {
		return nil, fmt.Errorf("bill not found: %w", err)
	}
	if bill == nil {
		return nil, fmt.Errorf("bill %s does not exist", req.BillID)
	}

	if bill.Status != "posted" && bill.Status != "closed" {
		return nil, fmt.Errorf("can only record payments for posted or closed bills (current status: %s)", bill.Status)
	}

	// Create payment record
	payment := &models.Payment{
		ID:          uuid.New().String(),
		BillID:      req.BillID,
		PayerUserID: userID,
		AmountPLN:   utils.FloatToDecimalString(req.Amount),
		PaidAt:      time.Now(),
		Method:      req.Method,
	}

	if err := s.payments.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to record payment: %w", err)
	}

	log.Printf("[PAYMENT] Recorded: %.2f PLN for bill %s by user %s (payment ID: %s)", req.Amount, req.BillID, userID, payment.ID)

	// Check if this payment completes a recurring bill and generate next bill if needed
	if s.recurringBillService != nil {
		// Log error but don't fail the payment - the next bill can be generated manually if needed
		_ = s.recurringBillService.CheckAndGenerateNextBill(ctx, req.BillID)
	}

	return payment, nil
}

// GetBillPayments returns all payments for a specific bill
func (s *PaymentService) GetBillPayments(ctx context.Context, billID string) ([]models.Payment, error) {
	return s.payments.ListByBillID(ctx, billID)
}

// GetUserPayments returns all payments made by a user
func (s *PaymentService) GetUserPayments(ctx context.Context, userID string) ([]models.Payment, error) {
	return s.payments.ListByPayerID(ctx, userID)
}
