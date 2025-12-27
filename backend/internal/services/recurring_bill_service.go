package services

import (
	"context"
	"errors"
	"fmt"
	"log"
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

	// Calculate first due date based on start date, handling end-of-month edge cases
	year, month, _ := template.StartDate.Date()

	// Get the last day of the start month
	lastDayOfMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()

	// If the requested day exceeds the last day of the month, use the last day
	actualDay := template.DayOfMonth
	if template.DayOfMonth > lastDayOfMonth {
		actualDay = lastDayOfMonth
	}

	template.NextDueDate = time.Date(year, month, actualDay, 0, 0, 0, 0, time.UTC)

	// If the calculated date is before StartDate (e.g., StartDate is Jan 31 but DayOfMonth is 15),
	// move to next period by adding the frequency interval
	if template.NextDueDate.Before(template.StartDate) {
		template.NextDueDate = s.calculateNextDueDate(template.NextDueDate, template.DayOfMonth, template.Frequency)
	}

	if err := s.templates.Create(ctx, template); err != nil {
		return err
	}

	// Save allocations for this template
	for _, alloc := range template.Allocations {
		if err := s.templateAllocations.Create(ctx, template.ID, &alloc); err != nil {
			return fmt.Errorf("failed to create template allocation: %w", err)
		}
	}

	// Create the first bill immediately
	if err := s.generateBillFromTemplate(ctx, template); err != nil {
		return fmt.Errorf("failed to generate first bill: %w", err)
	}

	log.Printf("[RECURRING BILL] Template created: %q (ID: %s, frequency: %s, amount: %s PLN, next due: %s)",
		template.CustomType, template.ID, template.Frequency, template.Amount, template.NextDueDate.Format("2006-01-02"))

	return nil
}

// GetTemplate retrieves a recurring bill template by ID
func (s *RecurringBillService) GetTemplate(ctx context.Context, id string) (*models.RecurringBillTemplate, error) {
	template, err := s.templates.GetByID(ctx, id)
	if err != nil {
		return nil, errors.New("template not found")
	}

	// Load allocations
	allocations, err := s.templateAllocations.GetByTemplateID(ctx, id)
	if err != nil {
		return nil, err
	}
	template.Allocations = allocations

	return template, nil
}

// ListTemplates retrieves all active recurring bill templates
func (s *RecurringBillService) ListTemplates(ctx context.Context) ([]models.RecurringBillTemplate, error) {
	templates, err := s.templates.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	// Load allocations for each template
	for i := range templates {
		allocations, err := s.templateAllocations.GetByTemplateID(ctx, templates[i].ID)
		if err != nil {
			return nil, err
		}
		templates[i].Allocations = allocations
	}

	return templates, nil
}

// UpdateTemplate updates an existing template
func (s *RecurringBillService) UpdateTemplate(ctx context.Context, id string, updates map[string]interface{}) error {
	// Get existing template
	template, err := s.templates.GetByID(ctx, id)
	if err != nil {
		return errors.New("template not found")
	}

	// Handle allocations if present in updates
	// JSON unmarshaling into map[string]interface{} results in []interface{}, not []models.RecurringBillAllocation
	if rawAllocations, ok := updates["allocations"]; ok {
		allocations, err := s.parseAllocations(rawAllocations)
		if err != nil {
			return fmt.Errorf("invalid allocations: %w", err)
		}
		if err := s.validateAllocations(allocations); err != nil {
			return err
		}
		template.Allocations = allocations
	}

	// Apply updates
	if customType, ok := updates["customType"].(string); ok {
		template.CustomType = customType
	}
	if frequency, ok := updates["frequency"].(string); ok {
		template.Frequency = frequency
	}
	if amount, ok := updates["amount"].(string); ok {
		template.Amount = amount
	}
	if dayOfMonth, ok := updates["dayOfMonth"]; ok {
		switch v := dayOfMonth.(type) {
		case float64:
			template.DayOfMonth = int(v)
		case int:
			template.DayOfMonth = v
		}
	}
	if notes, ok := updates["notes"]; ok {
		if notes == nil {
			template.Notes = nil
		} else if notesStr, ok := notes.(string); ok {
			template.Notes = &notesStr
		}
	}

	template.UpdatedAt = time.Now()

	// Update template in database
	if err := s.templates.Update(ctx, template); err != nil {
		return err
	}

	// If allocations were provided, replace them in the database
	if _, ok := updates["allocations"]; ok && len(template.Allocations) > 0 {
		if err := s.templateAllocations.ReplaceForTemplate(ctx, id, template.Allocations); err != nil {
			return fmt.Errorf("failed to update allocations: %w", err)
		}
	}

	return nil
}

// parseAllocations converts []interface{} (from JSON) to []models.RecurringBillAllocation
func (s *RecurringBillService) parseAllocations(raw interface{}) ([]models.RecurringBillAllocation, error) {
	rawSlice, ok := raw.([]interface{})
	if !ok {
		return nil, errors.New("allocations must be an array")
	}

	allocations := make([]models.RecurringBillAllocation, 0, len(rawSlice))
	for i, item := range rawSlice {
		allocMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("allocation %d is not an object", i+1)
		}

		alloc := models.RecurringBillAllocation{}

		if subjectType, ok := allocMap["subjectType"].(string); ok {
			alloc.SubjectType = subjectType
		}
		if subjectId, ok := allocMap["subjectId"].(string); ok {
			alloc.SubjectID = subjectId
		}
		if allocationType, ok := allocMap["allocationType"].(string); ok {
			alloc.AllocationType = allocationType
		}
		if percentage, ok := allocMap["percentage"].(float64); ok {
			alloc.Percentage = &percentage
		}
		if fractionNum, ok := allocMap["fractionNumerator"].(float64); ok {
			num := int(fractionNum)
			alloc.FractionNum = &num
		}
		if fractionDenom, ok := allocMap["fractionDenominator"].(float64); ok {
			denom := int(fractionDenom)
			alloc.FractionDenom = &denom
		}
		if fixedAmount, ok := allocMap["fixedAmount"].(string); ok {
			alloc.FixedAmount = &fixedAmount
		}

		allocations = append(allocations, alloc)
	}

	return allocations, nil
}

// DeleteTemplate deletes a template (soft delete by setting IsActive to false)
func (s *RecurringBillService) DeleteTemplate(ctx context.Context, id string) error {
	template, err := s.templates.GetByID(ctx, id)
	if err != nil {
		return errors.New("template not found")
	}

	template.IsActive = false
	template.UpdatedAt = time.Now()

	if err := s.templates.Update(ctx, template); err != nil {
		return err
	}

	log.Printf("[RECURRING BILL] Template deleted (soft): %q (ID: %s)", template.CustomType, id)
	return nil
}

// GenerateBillsFromTemplates generates bills from all active templates that are due
func (s *RecurringBillService) GenerateBillsFromTemplates(ctx context.Context) error {
	now := time.Now()

	// Find all active templates where next_due_date <= now
	templates, err := s.templates.ListDueBefore(ctx, now)
	if err != nil {
		return err
	}

	if len(templates) > 0 {
		log.Printf("[RECURRING BILL] Found %d templates due for bill generation", len(templates))
	}

	// Generate bills for each template
	for i := range templates {
		// Load allocations for this template
		allocations, err := s.templateAllocations.GetByTemplateID(ctx, templates[i].ID)
		if err != nil {
			log.Printf("[RECURRING BILL] Error loading allocations for template %s: %v", templates[i].ID, err)
			continue
		}
		templates[i].Allocations = allocations

		if err := s.generateBillFromTemplate(ctx, &templates[i]); err != nil {
			log.Printf("[RECURRING BILL] Error generating bill from template %s: %v", templates[i].ID, err)
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
		log.Printf("[RECURRING BILL] Creating allocation - Type: %s, SubjectType: %s", allocTemplate.AllocationType, allocTemplate.SubjectType)

		switch allocTemplate.AllocationType {
		case "fixed":
			allocatedAmount = *allocTemplate.FixedAmount
			amountFloat := utils.DecimalStringToFloat(allocatedAmount)
			log.Printf("[RECURRING BILL] Fixed allocation: %.2f PLN", amountFloat)
		case "percentage":
			amountFloat := utils.DecimalStringToFloat(template.Amount)
			percentage := *allocTemplate.Percentage / 100.0
			allocatedFloat := amountFloat * percentage
			roundedAmount := utils.RoundPLN(allocatedFloat)
			allocatedAmount = utils.FloatToDecimalString(roundedAmount)
			log.Printf("[RECURRING BILL] Percentage allocation: %.2f%% of %.2f = %.2f PLN", *allocTemplate.Percentage, amountFloat, roundedAmount)
		case "fraction":
			amountFloat := utils.DecimalStringToFloat(template.Amount)
			fraction := float64(*allocTemplate.FractionNum) / float64(*allocTemplate.FractionDenom)
			allocatedFloat := amountFloat * fraction
			roundedAmount := utils.RoundPLN(allocatedFloat)
			allocatedAmount = utils.FloatToDecimalString(roundedAmount)
			log.Printf("[RECURRING BILL] Fraction allocation: %d/%d of %.2f = %.2f PLN", *allocTemplate.FractionNum, *allocTemplate.FractionDenom, amountFloat, roundedAmount)
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

	if err := s.templates.Update(ctx, template); err != nil {
		return err
	}

	log.Printf("[RECURRING BILL] Bill generated from template %q (bill ID: %s, amount: %s PLN, period: %s to %s, next due: %s)",
		template.CustomType, billID, template.Amount, periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"), nextDueDate.Format("2006-01-02"))

	return nil
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

	// Adjust to the specified day of month, handling end-of-month edge cases
	year, month, _ := next.Date()

	// Get the last day of the target month
	lastDayOfMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, next.Location()).Day()

	// If the requested day exceeds the last day of the month, use the last day
	actualDay := dayOfMonth
	if dayOfMonth > lastDayOfMonth {
		actualDay = lastDayOfMonth
	}

	next = time.Date(year, month, actualDay, 0, 0, 0, 0, next.Location())

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
	// Get the bill by its ID
	bill, err := s.bills.GetByID(ctx, billID)
	if err != nil || bill == nil {
		// Bill not found, that's okay
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

	// Build a map of total amount paid by each user
	paymentMap := make(map[string]float64)
	for _, payment := range payments {
		paidFloat := utils.DecimalStringToFloat(payment.AmountPLN)
		paymentMap[payment.PayerUserID] += paidFloat
	}

	// Check if all users with allocations have paid their full amount
	allPaid := true
	for _, alloc := range storedAllocations {
		allocFloat := utils.DecimalStringToFloat(alloc.AllocatedPLN)

		if alloc.SubjectType == "user" {
			paidFloat := paymentMap[alloc.SubjectID]
			if paidFloat < allocFloat-0.01 { // Allow 1 cent tolerance for rounding
				allPaid = false
				break
			}
		} else if alloc.SubjectType == "group" {
			// Get all users in this group
			groupUsers, err := s.users.ListByGroupID(ctx, alloc.SubjectID)
			if err != nil {
				return err
			}

			// Calculate total paid by all group members
			totalPaidFloat := 0.0
			for _, user := range groupUsers {
				totalPaidFloat += paymentMap[user.ID]
			}

			if totalPaidFloat < allocFloat-0.01 { // Allow 1 cent tolerance for rounding
				allPaid = false
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
		// Validate subject type and ID
		if alloc.SubjectType != "user" && alloc.SubjectType != "group" {
			return fmt.Errorf("allocation %d: subject type must be 'user' or 'group'", i+1)
		}
		if alloc.SubjectID == "" {
			return fmt.Errorf("allocation %d: subject ID is required", i+1)
		}

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
