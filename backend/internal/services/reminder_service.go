package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
	"github.com/sainaif/holy-home/internal/utils"
)

type ReminderService struct {
	sentReminders       repository.SentReminderRepository
	appSettings         repository.AppSettingsRepository
	users               repository.UserRepository
	loans               repository.LoanRepository
	loanPayments        repository.LoanPaymentRepository
	choreAssignments    repository.ChoreAssignmentRepository
	chores              repository.ChoreRepository
	supplyItems         repository.SupplyItemRepository
	notificationService *NotificationService
}

func NewReminderService(
	sentReminders repository.SentReminderRepository,
	appSettings repository.AppSettingsRepository,
	users repository.UserRepository,
	loans repository.LoanRepository,
	loanPayments repository.LoanPaymentRepository,
	choreAssignments repository.ChoreAssignmentRepository,
	chores repository.ChoreRepository,
	supplyItems repository.SupplyItemRepository,
	notificationService *NotificationService,
) *ReminderService {
	return &ReminderService{
		sentReminders:       sentReminders,
		appSettings:         appSettings,
		users:               users,
		loans:               loans,
		loanPayments:        loanPayments,
		choreAssignments:    choreAssignments,
		chores:              chores,
		supplyItems:         supplyItems,
		notificationService: notificationService,
	}
}

// SendDebtReminder sends a reminder to a user about their debt to the sender
func (s *ReminderService) SendDebtReminder(ctx context.Context, targetUserID, senderUserID string) error {
	// Check rate limit for target user
	if err := s.checkRateLimit(ctx, targetUserID); err != nil {
		return err
	}

	// Get sender info
	sender, err := s.users.GetByID(ctx, senderUserID)
	if err != nil {
		return errors.New("sender not found")
	}

	// Verify target user exists
	_, err = s.users.GetByID(ctx, targetUserID)
	if err != nil {
		return errors.New("target user not found")
	}

	// Calculate total debt from target to sender
	debt, err := s.calculateDebt(ctx, targetUserID, senderUserID)
	if err != nil {
		return fmt.Errorf("failed to calculate debt: %w", err)
	}

	if debt <= 0 {
		return errors.New("user has no debt to you")
	}

	// Check if reminder was already sent
	exists, err := s.sentReminders.Exists(ctx, targetUserID, "debt", senderUserID, "manual")
	if err != nil {
		return fmt.Errorf("failed to check existing reminder: %w", err)
	}
	if exists {
		return errors.New("reminder already sent for this debt")
	}

	// Create notification
	if s.notificationService != nil {
		_ = s.notificationService.CreateNotification(ctx, &models.Notification{
			UserID:     &targetUserID,
			TemplateID: "debt_reminder",
			Title:      "Przypomnienie o zadłużeniu",
			Body:       fmt.Sprintf("%s przypomina o spłacie %.2f zł", sender.Name, debt),
		})
	}

	// Record sent reminder
	reminder := &models.SentReminder{
		UserID:       targetUserID,
		ResourceType: "debt",
		ResourceID:   senderUserID,
		ReminderType: "manual",
	}
	if err := s.sentReminders.Create(ctx, reminder); err != nil {
		return fmt.Errorf("failed to record reminder: %w", err)
	}

	return nil
}

// SendChoreReminder sends a reminder about a specific chore assignment
func (s *ReminderService) SendChoreReminder(ctx context.Context, assignmentID, senderUserID string) error {
	// Get assignment
	assignment, err := s.choreAssignments.GetByID(ctx, assignmentID)
	if err != nil {
		return errors.New("assignment not found")
	}

	if assignment.Status != "pending" {
		return errors.New("chore is not pending")
	}

	// Check rate limit for target user
	if err := s.checkRateLimit(ctx, assignment.AssigneeUserID); err != nil {
		return err
	}

	// Get sender info
	sender, err := s.users.GetByID(ctx, senderUserID)
	if err != nil {
		return errors.New("sender not found")
	}

	// Get chore info
	chore, err := s.chores.GetByID(ctx, assignment.ChoreID)
	if err != nil {
		return errors.New("chore not found")
	}

	// Check if reminder was already sent
	exists, err := s.sentReminders.Exists(ctx, assignment.AssigneeUserID, "chore_assignment", assignmentID, "manual")
	if err != nil {
		return fmt.Errorf("failed to check existing reminder: %w", err)
	}
	if exists {
		return errors.New("reminder already sent for this chore")
	}

	// Create notification
	if s.notificationService != nil {
		_ = s.notificationService.CreateNotification(ctx, &models.Notification{
			UserID:     &assignment.AssigneeUserID,
			TemplateID: "chore_reminder",
			Title:      "Przypomnienie o obowiązku",
			Body:       fmt.Sprintf("%s przypomina o wykonaniu: %s", sender.Name, chore.Name),
		})
	}

	// Record sent reminder
	reminder := &models.SentReminder{
		UserID:       assignment.AssigneeUserID,
		ResourceType: "chore_assignment",
		ResourceID:   assignmentID,
		ReminderType: "manual",
	}
	if err := s.sentReminders.Create(ctx, reminder); err != nil {
		return fmt.Errorf("failed to record reminder: %w", err)
	}

	return nil
}

// SendLowSuppliesReminder broadcasts a reminder about low supplies to all users
func (s *ReminderService) SendLowSuppliesReminder(ctx context.Context, senderUserID string) (int, error) {
	// Get sender info
	sender, err := s.users.GetByID(ctx, senderUserID)
	if err != nil {
		return 0, errors.New("sender not found")
	}

	// Get low supply items
	items, err := s.supplyItems.List(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get supply items: %w", err)
	}

	var lowItems []models.SupplyItem
	for _, item := range items {
		if item.CurrentQuantity < item.MinQuantity {
			lowItems = append(lowItems, item)
		}
	}

	if len(lowItems) == 0 {
		return 0, errors.New("no low supply items found")
	}

	// Get all users
	users, err := s.users.List(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get users: %w", err)
	}

	// Build message
	itemNames := ""
	for i, item := range lowItems {
		if i > 0 {
			itemNames += ", "
		}
		itemNames += item.Name
		if i >= 4 {
			itemNames += fmt.Sprintf(" i jeszcze %d...", len(lowItems)-5)
			break
		}
	}

	// Send notifications to all active users
	notifiedCount := 0
	for _, user := range users {
		if !user.IsActive || user.ID == senderUserID {
			continue
		}

		// Check rate limit
		if err := s.checkRateLimit(ctx, user.ID); err != nil {
			continue // Skip users who exceeded rate limit
		}

		if s.notificationService != nil {
			_ = s.notificationService.CreateNotification(ctx, &models.Notification{
				UserID:     &user.ID,
				TemplateID: "low_supplies_reminder",
				Title:      "Przypomnienie o zakupach",
				Body:       fmt.Sprintf("%s przypomina o uzupełnieniu: %s", sender.Name, itemNames),
			})
			notifiedCount++
		}

		// Record sent reminder (use sender ID as resource to allow re-sending later)
		reminder := &models.SentReminder{
			UserID:       user.ID,
			ResourceType: "supplies",
			ResourceID:   "broadcast_" + time.Now().Format("2006-01-02"),
			ReminderType: "manual",
		}
		_ = s.sentReminders.Create(ctx, reminder)
	}

	return notifiedCount, nil
}

// calculateDebt calculates how much borrowerID owes to lenderID
func (s *ReminderService) calculateDebt(ctx context.Context, borrowerID, lenderID string) (float64, error) {
	loans, err := s.loans.ListByBorrowerID(ctx, borrowerID)
	if err != nil {
		return 0, err
	}

	totalDebt := 0.0
	for _, loan := range loans {
		if loan.LenderID != lenderID {
			continue
		}
		if loan.Status == "settled" {
			continue
		}

		loanAmount := utils.DecimalStringToFloat(loan.AmountPLN)
		sumStr, err := s.loanPayments.SumByLoanID(ctx, loan.ID)
		if err != nil {
			continue
		}
		totalPaid := utils.DecimalStringToFloat(sumStr)
		remaining := loanAmount - totalPaid
		if remaining > 0 {
			totalDebt += remaining
		}
	}

	return totalDebt, nil
}

// checkRateLimit checks if a user has exceeded the reminder rate limit
func (s *ReminderService) checkRateLimit(ctx context.Context, userID string) error {
	settings, err := s.appSettings.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get app settings: %w", err)
	}

	// If no settings or rate limit is 0 (unlimited), skip check
	if settings == nil || settings.ReminderRateLimitPerHour == 0 {
		return nil
	}

	// Count reminders sent in the last hour
	since := time.Now().Add(-1 * time.Hour)
	count, err := s.sentReminders.CountRecentByUserID(ctx, userID, since)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}

	if count >= settings.ReminderRateLimitPerHour {
		return fmt.Errorf("rate limit exceeded: user already received %d reminder(s) in the last hour", count)
	}

	return nil
}

// CleanupOldReminders removes reminders older than 30 days
func (s *ReminderService) CleanupOldReminders(ctx context.Context) error {
	before := time.Now().AddDate(0, 0, -30)
	return s.sentReminders.DeleteOlderThan(ctx, before)
}
