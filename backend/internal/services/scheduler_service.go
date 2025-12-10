package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
	"github.com/sainaif/holy-home/internal/utils"
)

type SchedulerService struct {
	sentReminders       repository.SentReminderRepository
	users               repository.UserRepository
	bills               repository.BillRepository
	loans               repository.LoanRepository
	loanPayments        repository.LoanPaymentRepository
	choreAssignments    repository.ChoreAssignmentRepository
	chores              repository.ChoreRepository
	supplyItems         repository.SupplyItemRepository
	notificationService *NotificationService
}

func NewSchedulerService(
	sentReminders repository.SentReminderRepository,
	users repository.UserRepository,
	bills repository.BillRepository,
	loans repository.LoanRepository,
	loanPayments repository.LoanPaymentRepository,
	choreAssignments repository.ChoreAssignmentRepository,
	chores repository.ChoreRepository,
	supplyItems repository.SupplyItemRepository,
	notificationService *NotificationService,
) *SchedulerService {
	return &SchedulerService{
		sentReminders:       sentReminders,
		users:               users,
		bills:               bills,
		loans:               loans,
		loanPayments:        loanPayments,
		choreAssignments:    choreAssignments,
		chores:              chores,
		supplyItems:         supplyItems,
		notificationService: notificationService,
	}
}

// RunAllChecks runs all scheduled reminder checks
func (s *SchedulerService) RunAllChecks(ctx context.Context) {
	log.Println("Running scheduled reminder checks...")

	if err := s.CheckChoreReminders(ctx); err != nil {
		log.Printf("Error checking chore reminders: %v", err)
	}

	if err := s.CheckBillReminders(ctx); err != nil {
		log.Printf("Error checking bill reminders: %v", err)
	}

	if err := s.CheckLoanReminders(ctx); err != nil {
		log.Printf("Error checking loan reminders: %v", err)
	}

	if err := s.CheckLowSupplyReminders(ctx); err != nil {
		log.Printf("Error checking low supply reminders: %v", err)
	}

	log.Println("Scheduled reminder checks completed")
}

// CheckChoreReminders sends reminders for chores due soon
func (s *SchedulerService) CheckChoreReminders(ctx context.Context) error {
	// Get all pending chore assignments
	assignments, err := s.choreAssignments.ListByStatus(ctx, "pending")
	if err != nil {
		return fmt.Errorf("failed to list pending assignments: %w", err)
	}

	now := time.Now()
	remindersCreated := 0

	for _, assignment := range assignments {
		// Get the chore to check reminder settings
		chore, err := s.chores.GetByID(ctx, assignment.ChoreID)
		if err != nil {
			continue
		}

		// Skip if notifications disabled or no reminder hours set
		if !chore.NotificationsEnabled || chore.ReminderHours == nil {
			continue
		}

		// Calculate when reminder should be sent
		reminderTime := assignment.DueDate.Add(-time.Duration(*chore.ReminderHours) * time.Hour)

		// Check if it's time to send reminder (within the current hour)
		if now.Before(reminderTime) || now.After(assignment.DueDate) {
			continue
		}

		// Check if reminder was already sent
		exists, err := s.sentReminders.Exists(ctx, assignment.AssigneeUserID, "chore_assignment", assignment.ID, "auto_scheduled")
		if err != nil {
			continue
		}
		if exists {
			continue
		}

		// Create notification
		if s.notificationService != nil {
			timeLeft := time.Until(assignment.DueDate)
			hoursLeft := int(timeLeft.Hours())

			body := fmt.Sprintf("Obowiązek '%s' - termin za %d godz.", chore.Name, hoursLeft)
			if hoursLeft <= 0 {
				body = fmt.Sprintf("Obowiązek '%s' - termin upłynął!", chore.Name)
			}

			_ = s.notificationService.CreateNotification(ctx, &models.Notification{
				UserID:     &assignment.AssigneeUserID,
				TemplateID: "chore_due_reminder",
				Title:      "Przypomnienie o obowiązku",
				Body:       body,
			})
		}

		// Record sent reminder
		reminder := &models.SentReminder{
			UserID:       assignment.AssigneeUserID,
			ResourceType: "chore_assignment",
			ResourceID:   assignment.ID,
			ReminderType: "auto_scheduled",
		}
		if err := s.sentReminders.Create(ctx, reminder); err != nil {
			log.Printf("Failed to record chore reminder: %v", err)
		}
		remindersCreated++
	}

	if remindersCreated > 0 {
		log.Printf("Created %d chore reminders", remindersCreated)
	}
	return nil
}

// CheckBillReminders sends reminders for bills with upcoming payment deadlines
func (s *SchedulerService) CheckBillReminders(ctx context.Context) error {
	// Get all posted bills
	bills, err := s.bills.ListByStatus(ctx, "posted")
	if err != nil {
		return fmt.Errorf("failed to list posted bills: %w", err)
	}

	// Get all users to notify about bills
	users, err := s.users.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	now := time.Now()
	remindersCreated := 0

	for _, bill := range bills {
		// Skip bills without payment deadline
		if bill.PaymentDeadline == nil {
			continue
		}

		// Remind 3 days before deadline
		reminderTime := bill.PaymentDeadline.AddDate(0, 0, -3)

		// Check if it's time to send reminder
		if now.Before(reminderTime) || now.After(*bill.PaymentDeadline) {
			continue
		}

		// Get bill type name
		billTypeName := getBillTypeName(bill.Type, bill.CustomType)

		for _, user := range users {
			if !user.IsActive {
				continue
			}

			// Check if reminder was already sent
			exists, err := s.sentReminders.Exists(ctx, user.ID, "bill", bill.ID, "auto_scheduled")
			if err != nil {
				continue
			}
			if exists {
				continue
			}

			// Create notification
			if s.notificationService != nil {
				daysLeft := int(time.Until(*bill.PaymentDeadline).Hours() / 24)
				body := fmt.Sprintf("Rachunek '%s' - termin płatności za %d dni", billTypeName, daysLeft)
				if daysLeft <= 0 {
					body = fmt.Sprintf("Rachunek '%s' - termin płatności minął!", billTypeName)
				}

				_ = s.notificationService.CreateNotification(ctx, &models.Notification{
					UserID:     &user.ID,
					TemplateID: "bill_deadline_reminder",
					Title:      "Przypomnienie o płatności",
					Body:       body,
				})
			}

			// Record sent reminder
			reminder := &models.SentReminder{
				UserID:       user.ID,
				ResourceType: "bill",
				ResourceID:   bill.ID,
				ReminderType: "auto_scheduled",
			}
			if err := s.sentReminders.Create(ctx, reminder); err != nil {
				log.Printf("Failed to record bill reminder: %v", err)
			}
			remindersCreated++
		}
	}

	if remindersCreated > 0 {
		log.Printf("Created %d bill reminders", remindersCreated)
	}
	return nil
}

// CheckLoanReminders sends reminders for loans with upcoming due dates
func (s *SchedulerService) CheckLoanReminders(ctx context.Context) error {
	// Get all open/partial loans
	openLoans, err := s.loans.ListByStatus(ctx, "open")
	if err != nil {
		return fmt.Errorf("failed to list open loans: %w", err)
	}

	partialLoans, err := s.loans.ListByStatus(ctx, "partial")
	if err != nil {
		return fmt.Errorf("failed to list partial loans: %w", err)
	}

	loans := append(openLoans, partialLoans...)

	now := time.Now()
	remindersCreated := 0

	for _, loan := range loans {
		// Skip loans without due date
		if loan.DueDate == nil {
			continue
		}

		// Remind 3 days before due date
		reminderTime := loan.DueDate.AddDate(0, 0, -3)

		// Check if it's time to send reminder
		if now.Before(reminderTime) || now.After(*loan.DueDate) {
			continue
		}

		// Check if reminder was already sent
		exists, err := s.sentReminders.Exists(ctx, loan.BorrowerID, "loan", loan.ID, "auto_scheduled")
		if err != nil {
			continue
		}
		if exists {
			continue
		}

		// Get lender name
		lender, err := s.users.GetByID(ctx, loan.LenderID)
		lenderName := "kogoś"
		if err == nil && lender != nil {
			lenderName = lender.Name
		}

		// Calculate remaining amount
		totalPaidStr, _ := s.loanPayments.SumByLoanID(ctx, loan.ID)
		totalPaid := utils.DecimalStringToFloat(totalPaidStr)
		loanAmount := utils.DecimalStringToFloat(loan.AmountPLN)
		remaining := loanAmount - totalPaid

		// Create notification
		if s.notificationService != nil {
			daysLeft := int(time.Until(*loan.DueDate).Hours() / 24)
			body := fmt.Sprintf("Pożyczka od %s (%.2f zł) - termin za %d dni", lenderName, remaining, daysLeft)
			if daysLeft <= 0 {
				body = fmt.Sprintf("Pożyczka od %s (%.2f zł) - termin minął!", lenderName, remaining)
			}

			_ = s.notificationService.CreateNotification(ctx, &models.Notification{
				UserID:     &loan.BorrowerID,
				TemplateID: "loan_due_reminder",
				Title:      "Przypomnienie o pożyczce",
				Body:       body,
			})
		}

		// Record sent reminder
		reminder := &models.SentReminder{
			UserID:       loan.BorrowerID,
			ResourceType: "loan",
			ResourceID:   loan.ID,
			ReminderType: "auto_scheduled",
		}
		if err := s.sentReminders.Create(ctx, reminder); err != nil {
			log.Printf("Failed to record loan reminder: %v", err)
		}
		remindersCreated++
	}

	if remindersCreated > 0 {
		log.Printf("Created %d loan reminders", remindersCreated)
	}
	return nil
}

// CheckLowSupplyReminders sends daily digest of low stock items
func (s *SchedulerService) CheckLowSupplyReminders(ctx context.Context) error {
	// Get all low stock items
	items, err := s.supplyItems.ListLowStock(ctx)
	if err != nil {
		return fmt.Errorf("failed to list low stock items: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	// Get all active users
	users, err := s.users.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	// Build item names for notification
	itemNames := ""
	for i, item := range items {
		if i > 0 {
			itemNames += ", "
		}
		itemNames += item.Name
		if i >= 4 {
			itemNames += fmt.Sprintf(" i jeszcze %d...", len(items)-5)
			break
		}
	}

	// Resource ID based on today's date to allow daily reminders
	resourceID := "daily_" + time.Now().Format("2006-01-02")
	remindersCreated := 0

	for _, user := range users {
		if !user.IsActive {
			continue
		}

		// Check if reminder was already sent today
		exists, err := s.sentReminders.Exists(ctx, user.ID, "supplies", resourceID, "auto_scheduled")
		if err != nil {
			continue
		}
		if exists {
			continue
		}

		// Create notification
		if s.notificationService != nil {
			_ = s.notificationService.CreateNotification(ctx, &models.Notification{
				UserID:     &user.ID,
				TemplateID: "low_supplies_daily",
				Title:      "Niskie stany magazynowe",
				Body:       fmt.Sprintf("Produkty wymagające uzupełnienia: %s", itemNames),
			})
		}

		// Record sent reminder
		reminder := &models.SentReminder{
			UserID:       user.ID,
			ResourceType: "supplies",
			ResourceID:   resourceID,
			ReminderType: "auto_scheduled",
		}
		if err := s.sentReminders.Create(ctx, reminder); err != nil {
			log.Printf("Failed to record supply reminder: %v", err)
		}
		remindersCreated++
	}

	if remindersCreated > 0 {
		log.Printf("Created %d low supply reminders for %d items", remindersCreated, len(items))
	}
	return nil
}

// getBillTypeName returns the display name for a bill type
func getBillTypeName(billType string, customType *string) string {
	switch billType {
	case "electricity":
		return "Prąd"
	case "gas":
		return "Gaz"
	case "internet":
		return "Internet"
	case "inne":
		if customType != nil {
			return *customType
		}
		return "Inne"
	default:
		return billType
	}
}
