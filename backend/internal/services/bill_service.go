package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
	"github.com/sainaif/holy-home/internal/utils"
)

type BillService struct {
	bills               repository.BillRepository
	consumptions        repository.ConsumptionRepository
	allocations         repository.AllocationRepository
	payments            repository.PaymentRepository
	users               repository.UserRepository
	groups              repository.GroupRepository
	notificationService *NotificationService
}

func NewBillService(
	bills repository.BillRepository,
	consumptions repository.ConsumptionRepository,
	allocations repository.AllocationRepository,
	payments repository.PaymentRepository,
	users repository.UserRepository,
	groups repository.GroupRepository,
	notificationService *NotificationService,
) *BillService {
	return &BillService{
		bills:               bills,
		consumptions:        consumptions,
		allocations:         allocations,
		payments:            payments,
		users:               users,
		groups:              groups,
		notificationService: notificationService,
	}
}

type CreateBillRequest struct {
	Type            string     `json:"type"`                     // electricity, gas, internet, inne
	CustomType      *string    `json:"customType,omitempty"`     // required when type is "inne"
	AllocationType  *string    `json:"allocationType,omitempty"` // "simple" or "metered", required when type is "inne"
	PeriodStart     time.Time  `json:"periodStart"`
	PeriodEnd       time.Time  `json:"periodEnd"`
	PaymentDeadline *time.Time `json:"paymentDeadline,omitempty"` // optional payment deadline
	TotalAmountPLN  float64    `json:"totalAmountPLN"`
	TotalUnits      *float64   `json:"totalUnits,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
}

// CreateBill creates a new bill in the database
func (s *BillService) CreateBill(ctx context.Context, req CreateBillRequest, creatorID string) (*models.Bill, error) {
	// Validate bill type
	validTypes := map[string]bool{"electricity": true, "gas": true, "internet": true, "inne": true}
	if !validTypes[req.Type] {
		return nil, errors.New("invalid bill type")
	}

	// Validate customType is provided when type is "inne"
	if req.Type == "inne" && (req.CustomType == nil || *req.CustomType == "") {
		return nil, errors.New("customType is required when type is 'inne'")
	}

	// Validate customType is not provided for other types
	if req.Type != "inne" && req.CustomType != nil {
		return nil, errors.New("customType should only be provided when type is 'inne'")
	}

	// Validate allocationType for "inne" type
	if req.Type == "inne" && (req.AllocationType == nil || (*req.AllocationType != "simple" && *req.AllocationType != "metered")) {
		return nil, errors.New("allocationType must be 'simple' or 'metered' when type is 'inne'")
	}

	// Set default allocation types for standard bill types
	allocationType := req.AllocationType
	if req.Type == "gas" || req.Type == "internet" {
		simpleType := "simple"
		allocationType = &simpleType
	} else if req.Type == "electricity" {
		meteredType := "metered"
		allocationType = &meteredType
	}

	if req.PeriodEnd.Before(req.PeriodStart) {
		return nil, errors.New("period end must be after period start")
	}

	amountStr := utils.FloatToDecimalString(req.TotalAmountPLN)

	bill := models.Bill{
		ID:              uuid.New().String(),
		Type:            req.Type,
		CustomType:      req.CustomType,
		AllocationType:  allocationType,
		PeriodStart:     req.PeriodStart,
		PeriodEnd:       req.PeriodEnd,
		PaymentDeadline: req.PaymentDeadline,
		TotalAmountPLN:  amountStr,
		Notes:           req.Notes,
		Status:          "draft",
		CreatedAt:       time.Now(),
	}

	if req.TotalUnits != nil {
		unitsStr := utils.FloatToDecimalString(*req.TotalUnits)
		bill.TotalUnits = unitsStr
	}

	if err := s.bills.Create(ctx, &bill); err != nil {
		return nil, fmt.Errorf("failed to create bill: %w", err)
	}

	// Create a notification for all users except the creator
	users, err := s.users.ListActive(ctx)
	if err != nil {
		log.Printf("failed to get all active users: %v", err)
	} else {
		for _, user := range users {
			if user.ID != creatorID {
				now := time.Now()
				notification := &models.Notification{
					UserID:       &user.ID,
					Channel:      "app",
					TemplateID:   "bill",
					ScheduledFor: now,
					SentAt:       &now,
					Status:       "sent",
					Title:        "Nowy rachunek",
					Body:         fmt.Sprintf("Dodano nowy rachunek: %s", bill.Type),
				}
				s.notificationService.CreateNotification(ctx, notification)
			}
		}
	}

	return &bill, nil
}

// GetBills gets all bills, can filter by type and dates
func (s *BillService) GetBills(ctx context.Context, billType *string, from *time.Time, to *time.Time) ([]models.Bill, error) {
	var bills []models.Bill
	var err error

	if billType != nil && from != nil && to != nil {
		// Filter by type and period
		allBills, err := s.bills.ListByType(ctx, *billType)
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		// Further filter by date
		for _, b := range allBills {
			if (b.PeriodStart.Equal(*from) || b.PeriodStart.After(*from)) &&
				(b.PeriodStart.Equal(*to) || b.PeriodStart.Before(*to)) {
				bills = append(bills, b)
			}
		}
	} else if billType != nil {
		bills, err = s.bills.ListByType(ctx, *billType)
	} else if from != nil && to != nil {
		bills, err = s.bills.ListByPeriod(ctx, *from, *to)
	} else {
		bills, err = s.bills.List(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return bills, nil
}

// GetBill gets a single bill by ID
func (s *BillService) GetBill(ctx context.Context, billID string) (*models.Bill, error) {
	bill, err := s.bills.GetByID(ctx, billID)
	if err != nil {
		return nil, errors.New("bill not found")
	}
	return bill, nil
}

// PostBill marks bill as posted (freezes allocations)
func (s *BillService) PostBill(ctx context.Context, billID string) error {
	return s.updateBillStatus(ctx, billID, "draft", "posted")
}

// CloseBill marks bill as closed (no more changes)
func (s *BillService) CloseBill(ctx context.Context, billID string) error {
	return s.updateBillStatus(ctx, billID, "posted", "closed")
}

// ReopenBill reverts a bill back to draft or posted status
func (s *BillService) ReopenBill(ctx context.Context, billID string, userID string, targetStatus, reason string) error {
	// Validate target status
	if targetStatus != "draft" && targetStatus != "posted" {
		return errors.New("target status must be 'draft' or 'posted'")
	}

	if reason == "" {
		return errors.New("reopen reason is required")
	}

	// Get current bill
	bill, err := s.GetBill(ctx, billID)
	if err != nil {
		return err
	}

	// Validate state transition
	if bill.Status == "draft" {
		return errors.New("bill is already in draft status")
	}

	if bill.Status == "posted" && targetStatus == "posted" {
		return errors.New("bill is already in posted status")
	}

	now := time.Now()

	// Update bill status and reopen metadata
	bill.Status = targetStatus
	bill.ReopenedAt = &now
	bill.ReopenReason = &reason
	bill.ReopenedBy = &userID

	if err := s.bills.Update(ctx, bill); err != nil {
		return fmt.Errorf("failed to reopen bill: %w", err)
	}

	return nil
}

func (s *BillService) updateBillStatus(ctx context.Context, billID string, fromStatus, toStatus string) error {
	bill, err := s.bills.GetByID(ctx, billID)
	if err != nil {
		return fmt.Errorf("bill not found or not in %s status", fromStatus)
	}

	if bill.Status != fromStatus {
		return fmt.Errorf("bill not found or not in %s status", fromStatus)
	}

	bill.Status = toStatus
	if err := s.bills.Update(ctx, bill); err != nil {
		return fmt.Errorf("failed to update bill status: %w", err)
	}

	return nil
}

// DeleteBill deletes a bill and all associated data
func (s *BillService) DeleteBill(ctx context.Context, billID string) error {
	// Check bill exists
	_, err := s.bills.GetByID(ctx, billID)
	if err != nil {
		return errors.New("bill not found")
	}

	// Delete all consumptions
	if err := s.consumptions.DeleteByBillID(ctx, billID); err != nil {
		return fmt.Errorf("failed to delete consumptions: %w", err)
	}

	// Delete all allocations
	if err := s.allocations.DeleteByBillID(ctx, billID); err != nil {
		return fmt.Errorf("failed to delete allocations: %w", err)
	}

	// Note: payments are not deleted as they represent actual money transactions
	// They could be kept for audit purposes or handled separately

	// Delete the bill
	if err := s.bills.Delete(ctx, billID); err != nil {
		return fmt.Errorf("failed to delete bill: %w", err)
	}

	return nil
}

// PaymentStatusEntry represents payment status for a user/group
type PaymentStatusEntry struct {
	SubjectID    string `json:"subjectId"`
	SubjectType  string `json:"subjectType"` // "user" or "group"
	SubjectName  string `json:"subjectName"`
	AllocatedPLN string `json:"allocatedPLN"`
	PaidPLN      string `json:"paidPLN"`
	RemainingPLN string `json:"remainingPLN"`
	IsPaid       bool   `json:"isPaid"`
}

// GetBillPaymentStatus returns detailed payment status showing who paid and who hasn't
func (s *BillService) GetBillPaymentStatus(ctx context.Context, billID string) ([]PaymentStatusEntry, error) {
	// Get all allocations for this bill
	allocations, err := s.allocations.GetByBillID(ctx, billID)
	if err != nil {
		return nil, err
	}

	// Get all payments for this bill
	payments, err := s.payments.ListByBillID(ctx, billID)
	if err != nil {
		return nil, err
	}

	// Build payment map by payer
	paymentMap := make(map[string]float64)
	for _, payment := range payments {
		paidFloat := utils.DecimalStringToFloat(payment.AmountPLN)
		paymentMap[payment.PayerUserID] += paidFloat
	}

	// Build status entries
	var statusEntries []PaymentStatusEntry
	for _, alloc := range allocations {
		var subjectName string

		// Get subject name
		if alloc.SubjectType == "user" {
			user, err := s.users.GetByID(ctx, alloc.SubjectID)
			if err == nil {
				subjectName = user.Name
			} else {
				subjectName = "Unknown User"
			}

			// Get paid amount for this user
			paidFloat := paymentMap[alloc.SubjectID]
			allocFloat := utils.DecimalStringToFloat(alloc.AllocatedPLN)
			remainingFloat := allocFloat - paidFloat

			statusEntries = append(statusEntries, PaymentStatusEntry{
				SubjectID:    alloc.SubjectID,
				SubjectType:  alloc.SubjectType,
				SubjectName:  subjectName,
				AllocatedPLN: alloc.AllocatedPLN,
				PaidPLN:      utils.FloatToDecimalString(paidFloat),
				RemainingPLN: utils.FloatToDecimalString(remainingFloat),
				IsPaid:       paidFloat >= allocFloat,
			})
		} else if alloc.SubjectType == "group" {
			group, err := s.groups.GetByID(ctx, alloc.SubjectID)
			if err == nil {
				subjectName = group.Name
			} else {
				subjectName = "Unknown Group"
			}

			// Get all users in this group
			groupUsers, err := s.users.ListByGroupID(ctx, alloc.SubjectID)
			if err != nil {
				continue
			}

			// Calculate total paid by all group members
			totalPaidFloat := 0.0
			for _, user := range groupUsers {
				totalPaidFloat += paymentMap[user.ID]
			}

			allocFloat := utils.DecimalStringToFloat(alloc.AllocatedPLN)
			remainingFloat := allocFloat - totalPaidFloat

			statusEntries = append(statusEntries, PaymentStatusEntry{
				SubjectID:    alloc.SubjectID,
				SubjectType:  alloc.SubjectType,
				SubjectName:  subjectName,
				AllocatedPLN: alloc.AllocatedPLN,
				PaidPLN:      utils.FloatToDecimalString(totalPaidFloat),
				RemainingPLN: utils.FloatToDecimalString(remainingFloat),
				IsPaid:       totalPaidFloat >= allocFloat,
			})
		}
	}

	return statusEntries, nil
}
