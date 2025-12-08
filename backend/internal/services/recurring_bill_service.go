package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/config"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
	"github.com/sainaif/holy-home/internal/utils"
)

type RecurringBillService struct {
	templates           repository.RecurringBillTemplateRepository
	templateAllocations repository.RecurringBillAllocationRepository
	bills               repository.BillRepository
	allocations         repository.AllocationRepository
	payments            repository.PaymentRepository
	users               repository.UserRepository
	cfg                 *config.Config
}

func NewRecurringBillService(
	templates repository.RecurringBillTemplateRepository,
	templateAllocations repository.RecurringBillAllocationRepository,
	bills repository.BillRepository,
	allocations repository.AllocationRepository,
	payments repository.PaymentRepository,
	users repository.UserRepository,
	cfg *config.Config,
) *RecurringBillService {
	return &RecurringBillService{
		templates:           templates,
		templateAllocations: templateAllocations,
		bills:               bills,
		allocations:         allocations,
		payments:            payments,
		users:               users,
		cfg:                 cfg,
	}
}

// CreateTemplate creates a new recurring bill template and generates the first bill immediately
func (s *RecurringBillService) CreateTemplate(ctx context.Context, template *models.RecurringBillTemplate) error {
	// Validate allocations
	if err := s.validateAllocations(template.Allocations); err != nil {
		return err
	}

	// Set timestamps
	now := time.Now()
	template.ID = uuid.New().String()
	template.CreatedAt = now
	template.UpdatedAt = now
	template.IsActive = true

	// Calculate first due date based on start date
	year, month, _ := template.StartDate.Date()
	template.NextDueDate = time.Date(year, month, template.DayOfMonth, 0, 0, 0, 0, time.UTC)

	// If the calculated date is before StartDate (e.g., StartDate is Jan 31 but DayOfMonth is 15),
	// move to next period by adding the frequency interval
	if template.NextDueDate.Before(template.StartDate) {
		switch template.Frequency {
		case "monthly":
			template.NextDueDate = template.NextDueDate.AddDate(0, 1, 0)
		case "quarterly":
			template.NextDueDate = template.NextDueDate.AddDate(0, 3, 0)
		case "yearly":
			template.NextDueDate = template.NextDueDate.AddDate(1, 0, 0)
		default:
			template.NextDueDate = template.NextDueDate.AddDate(0, 1, 0)
		}
	}

	if err := s.templates.Create(ctx, template); err != nil {
		return err
	}

	// Create the first bill immediately
	if err := s.generateBillFromTemplate(ctx, template); err != nil {
		return fmt.Errorf("failed to generate first bill: %w", err)
	}

	return nil
}

// GetTemplate retrieves a recurring bill template by ID
func (s *RecurringBillService) GetTemplate(ctx context.Context, id string) (*models.RecurringBillTemplate, error) {
	template, err := s.templates.GetByID(ctx, id)
	if err != nil {
		return nil, errors.New("template not found")
	}
	return template, nil
}

// ListTemplates retrieves all active recurring bill templates
func (s *RecurringBillService) ListTemplates(ctx context.Context) ([]models.RecurringBillTemplate, error) {
	return s.templates.ListActive(ctx)
}

// UpdateTemplate updates an existing template
func (s *RecurringBillService) UpdateTemplate(ctx context.Context, id string, updates map[string]interface{}) error {
	// Get existing template
	template, err := s.templates.GetByID(ctx, id)
	if err != nil {
		return errors.New("template not found")
	}

	// Validate allocations if being updated
	if allocations, ok := updates["allocations"].([]models.RecurringBillAllocation); ok {
		if err := s.validateAllocations(allocations); err != nil {
			return err
		}
		template.Allocations = allocations
	}

	// Apply updates
	if customType, ok := updates["custom_type"].(string); ok {
		template.CustomType = customType
	}
	if frequency, ok := updates["frequency"].(string); ok {
		template.Frequency = frequency
	}
	if amount, ok := updates["amount"].(string); ok {
		template.Amount = amount
	}
	if dayOfMonth, ok := updates["day_of_month"].(int); ok {
		template.DayOfMonth = dayOfMonth
	}
	if notes, ok := updates["notes"].(*string); ok {
		template.Notes = notes
	}

	template.UpdatedAt = time.Now()

	return s.templates.Update(ctx, template)
}

// DeleteTemplate deletes a template (soft delete by setting IsActive to false)
func (s *RecurringBillService) DeleteTemplate(ctx context.Context, id string) error {
	template, err := s.templates.GetByID(ctx, id)
	if err != nil {
		return errors.New("template not found")
	}

	template.IsActive = false
	template.UpdatedAt = time.Now()

	return s.templates.Update(ctx, template)
}

// GenerateBillsFromTemplates generates bills from all active templates that are due
func (s *RecurringBillService) GenerateBillsFromTemplates(ctx context.Context) error {
	now := time.Now()

	// Find all active templates where next_due_date <= now
	templates, err := s.templates.ListDueBefore(ctx, now)
	if err != nil {
		return err
	}

	// Generate bills for each template
	for _, template := range templates {
		if err := s.generateBillFromTemplate(ctx, &template); err != nil {
			fmt.Printf("Error generating bill from template %s: %v\n", template.ID, err)
			continue
		}
	}

	return nil
}

// generateBillFromTemplate generates a single bill from a template
func (s *RecurringBillService) generateBillFromTemplate(ctx context.Context, template *models.RecurringBillTemplate) error {
	now := time.Now()

	// Calculate period based on frequency
	periodStart, periodEnd := s.calculatePeriod(template.NextDueDate, template.Frequency)

	// Create the bill
	allocationType := "simple"
	billID := uuid.New().String()
	bill := &models.Bill{
		ID:                  billID,
		Type:                "inne",
		CustomType:          &template.CustomType,
		AllocationType:      &allocationType,
		PeriodStart:         periodStart,
		PeriodEnd:           periodEnd,
		PaymentDeadline:     &template.NextDueDate,
		TotalAmountPLN:      template.Amount,
		Notes:               template.Notes,
		Status:              "draft", // Start as draft so it's modifiable
		RecurringTemplateID: &template.ID,
		CreatedAt:           now,
	}

	// Insert the bill
	if err := s.bills.Create(ctx, bill); err != nil {
		return err
	}

	// Create allocations based on template
	for _, allocTemplate := range template.Allocations {
		var allocatedAmount string

		// Debug logging to track allocation creation
		fmt.Printf("[RecurringBill] Creating allocation - Type: %s, SubjectType: %s\n", allocTemplate.AllocationType, allocTemplate.SubjectType)

		switch allocTemplate.AllocationType {
		case "fixed":
			allocatedAmount = *allocTemplate.FixedAmount
			amountFloat := utils.DecimalStringToFloat(allocatedAmount)
			fmt.Printf("[RecurringBill] Fixed allocation: %.2f PLN\n", amountFloat)
		case "percentage":
			amountFloat := utils.DecimalStringToFloat(template.Amount)
			percentage := *allocTemplate.Percentage / 100.0
			allocatedFloat := amountFloat * percentage
			roundedAmount := utils.RoundPLN(allocatedFloat)
			allocatedAmount = utils.FloatToDecimalString(roundedAmount)
			fmt.Printf("[RecurringBill] Percentage allocation: %.2f%% of %.2f = %.2f PLN\n", *allocTemplate.Percentage, amountFloat, roundedAmount)
		case "fraction":
			amountFloat := utils.DecimalStringToFloat(template.Amount)
			fraction := float64(*allocTemplate.FractionNum) / float64(*allocTemplate.FractionDenom)
			allocatedFloat := amountFloat * fraction
			roundedAmount := utils.RoundPLN(allocatedFloat)
			allocatedAmount = utils.FloatToDecimalString(roundedAmount)
			fmt.Printf("[RecurringBill] Fraction allocation: %d/%d of %.2f = %.2f PLN\n", *allocTemplate.FractionNum, *allocTemplate.FractionDenom, amountFloat, roundedAmount)
		}

		if err := s.allocations.Create(ctx, billID, allocTemplate.SubjectType, allocTemplate.SubjectID, allocatedAmount); err != nil {
			return fmt.Errorf("failed to create allocation: %w", err)
		}
	}

	// Update template's next due date, current bill ID, and last generated timestamp
	nextDueDate := s.calculateNextDueDate(template.NextDueDate, template.DayOfMonth, template.Frequency)
	template.CurrentBillID = &billID
	template.NextDueDate = nextDueDate
	template.LastGeneratedAt = &now
	template.UpdatedAt = now

	return s.templates.Update(ctx, template)
}

// calculateNextDueDate calculates the next due date based on frequency
func (s *RecurringBillService) calculateNextDueDate(from time.Time, dayOfMonth int, frequency string) time.Time {
	var next time.Time

	switch frequency {
	case "monthly":
		next = from.AddDate(0, 1, 0)
	case "quarterly":
		next = from.AddDate(0, 3, 0)
	case "yearly":
		next = from.AddDate(1, 0, 0)
	default:
		next = from.AddDate(0, 1, 0)
	}

	// Adjust to the specified day of month
	year, month, _ := next.Date()
	next = time.Date(year, month, dayOfMonth, 0, 0, 0, 0, next.Location())

	return next
}

// calculatePeriod calculates the billing period for a given due date
func (s *RecurringBillService) calculatePeriod(dueDate time.Time, frequency string) (time.Time, time.Time) {
	var periodStart time.Time

	switch frequency {
	case "monthly":
		periodStart = dueDate.AddDate(0, -1, 0)
	case "quarterly":
		periodStart = dueDate.AddDate(0, -3, 0)
	case "yearly":
		periodStart = dueDate.AddDate(-1, 0, 0)
	default:
		periodStart = dueDate.AddDate(0, -1, 0)
	}

	periodEnd := dueDate

	return periodStart, periodEnd
}

// CheckAndGenerateNextBill checks if a bill is from a recurring template and all payments are made,
// then generates the next bill if ready
func (s *RecurringBillService) CheckAndGenerateNextBill(ctx context.Context, billID string) error {
	// Find the recurring template that has this bill as its current bill
	bill, err := s.bills.GetByRecurringTemplateID(ctx, billID)
	if err != nil || bill == nil {
		// Not a recurring bill or already processed, that's okay
		return nil
	}

	if bill.RecurringTemplateID == nil {
		return nil
	}

	template, err := s.templates.GetByID(ctx, *bill.RecurringTemplateID)
	if err != nil || !template.IsActive {
		return nil
	}

	// Only generate next bill if the current bill is posted (not draft)
	if bill.Status != "posted" {
		return nil
	}

	// Get all allocations for this bill
	storedAllocations, err := s.allocations.GetByBillID(ctx, billID)
	if err != nil {
		return err
	}

	// Get all payments for this bill
	payments, err := s.payments.ListByBillID(ctx, billID)
	if err != nil {
		return err
	}

	// Build a map of users who have paid
	paidUsers := make(map[string]bool)
	for _, payment := range payments {
		paidUsers[payment.PayerUserID] = true
	}

	// Check if all users with allocations have paid
	allPaid := true
	for _, alloc := range storedAllocations {
		if alloc.SubjectType == "user" {
			if !paidUsers[alloc.SubjectID] {
				allPaid = false
				break
			}
		} else if alloc.SubjectType == "group" {
			// Get all users in this group
			groupUsers, err := s.users.ListByGroupID(ctx, alloc.SubjectID)
			if err != nil {
				return err
			}

			// Check if any group member hasn't paid
			for _, user := range groupUsers {
				if !paidUsers[user.ID] {
					allPaid = false
					break
				}
			}
			if !allPaid {
				break
			}
		}
	}

	// If all users have paid, generate the next bill
	if allPaid {
		return s.generateBillFromTemplate(ctx, template)
	}

	return nil
}

// validateAllocations validates that allocations are properly configured
func (s *RecurringBillService) validateAllocations(allocations []models.RecurringBillAllocation) error {
	if len(allocations) == 0 {
		return errors.New("at least one allocation is required")
	}

	for i, alloc := range allocations {
		switch alloc.AllocationType {
		case "percentage":
			if alloc.Percentage == nil {
				return fmt.Errorf("allocation %d: percentage is required for percentage type", i+1)
			}
			if *alloc.Percentage <= 0 || *alloc.Percentage > 100 {
				return fmt.Errorf("allocation %d: percentage must be between 0 and 100", i+1)
			}
		case "fraction":
			if alloc.FractionNum == nil || alloc.FractionDenom == nil {
				return fmt.Errorf("allocation %d: fraction numerator and denominator are required for fraction type", i+1)
			}
			if *alloc.FractionNum <= 0 || *alloc.FractionDenom <= 0 {
				return fmt.Errorf("allocation %d: fraction values must be positive", i+1)
			}
			if *alloc.FractionNum > *alloc.FractionDenom {
				return fmt.Errorf("allocation %d: fraction numerator cannot be greater than denominator", i+1)
			}
		case "fixed":
			if alloc.FixedAmount == nil {
				return fmt.Errorf("allocation %d: fixed amount is required for fixed type", i+1)
			}
		default:
			return fmt.Errorf("allocation %d: invalid allocation type '%s'", i+1, alloc.AllocationType)
		}
	}

	// Validate that allocations sum to 100% (for percentage/fraction types)
	totalFraction := 0.0
	hasNonFixed := false

	for _, alloc := range allocations {
		switch alloc.AllocationType {
		case "percentage":
			totalFraction += *alloc.Percentage / 100.0
			hasNonFixed = true
		case "fraction":
			totalFraction += float64(*alloc.FractionNum) / float64(*alloc.FractionDenom)
			hasNonFixed = true
		}
	}

	if hasNonFixed && (totalFraction < 0.999 || totalFraction > 1.001) {
		return fmt.Errorf("allocations must sum to 100%% (currently %.2f%%)", totalFraction*100)
	}

	return nil
}
