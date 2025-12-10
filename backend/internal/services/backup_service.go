package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
	"github.com/sainaif/holy-home/internal/utils"
)

type BackupService struct {
	db                       *sqlx.DB
	users                    repository.UserRepository
	groups                   repository.GroupRepository
	bills                    repository.BillRepository
	consumptions             repository.ConsumptionRepository
	allocations              repository.AllocationRepository
	payments                 repository.PaymentRepository
	loans                    repository.LoanRepository
	loanPayments             repository.LoanPaymentRepository
	chores                   repository.ChoreRepository
	choreAssignments         repository.ChoreAssignmentRepository
	choreSettings            repository.ChoreSettingsRepository
	notifications            repository.NotificationRepository
	supplySettings           repository.SupplySettingsRepository
	supplyItems              repository.SupplyItemRepository
	supplyContributions      repository.SupplyContributionRepository
	recurringBillTemplates   repository.RecurringBillTemplateRepository
	recurringBillAllocations repository.RecurringBillAllocationRepository
}

func NewBackupService(
	db *sqlx.DB,
	users repository.UserRepository,
	groups repository.GroupRepository,
	bills repository.BillRepository,
	consumptions repository.ConsumptionRepository,
	allocations repository.AllocationRepository,
	payments repository.PaymentRepository,
	loans repository.LoanRepository,
	loanPayments repository.LoanPaymentRepository,
	chores repository.ChoreRepository,
	choreAssignments repository.ChoreAssignmentRepository,
	choreSettings repository.ChoreSettingsRepository,
	notifications repository.NotificationRepository,
	supplySettings repository.SupplySettingsRepository,
	supplyItems repository.SupplyItemRepository,
	supplyContributions repository.SupplyContributionRepository,
	recurringBillTemplates repository.RecurringBillTemplateRepository,
	recurringBillAllocations repository.RecurringBillAllocationRepository,
) *BackupService {
	return &BackupService{
		db:                       db,
		users:                    users,
		groups:                   groups,
		bills:                    bills,
		consumptions:             consumptions,
		allocations:              allocations,
		payments:                 payments,
		loans:                    loans,
		loanPayments:             loanPayments,
		chores:                   chores,
		choreAssignments:         choreAssignments,
		choreSettings:            choreSettings,
		notifications:            notifications,
		supplySettings:           supplySettings,
		supplyItems:              supplyItems,
		supplyContributions:      supplyContributions,
		recurringBillTemplates:   recurringBillTemplates,
		recurringBillAllocations: recurringBillAllocations,
	}
}

// BackupUser is a User struct with all fields exported for backup purposes
// (models.User has json:"-" on PasswordHash and TOTPSecret)
type BackupUser struct {
	ID                 string    `json:"id"`
	Email              string    `json:"email"`
	Username           string    `json:"username,omitempty"`
	Name               string    `json:"name"`
	PasswordHash       string    `json:"passwordHash"`
	Role               string    `json:"role"`
	GroupID            *string   `json:"groupId,omitempty"`
	IsActive           bool      `json:"isActive"`
	MustChangePassword bool      `json:"mustChangePassword"`
	TOTPSecret         string    `json:"totpSecret,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
}

// ImportResult contains information about the import operation
type ImportResult struct {
	UsersWithResetPasswords []string `json:"usersWithResetPasswords"` // Email addresses of users who got default passwords
	DefaultPassword         string   `json:"defaultPassword"`         // The default password assigned (only if there are users with reset passwords)
}

// BackupData represents a complete system backup
type BackupData struct {
	Version                  string                           `json:"version"`
	ExportedAt               time.Time                        `json:"exportedAt"`
	Users                    []BackupUser                     `json:"users"`
	Groups                   []models.Group                   `json:"groups"`
	Bills                    []models.Bill                    `json:"bills"`
	Consumptions             []models.Consumption             `json:"consumptions"`
	Allocations              []repository.Allocation          `json:"allocations"`
	Payments                 []models.Payment                 `json:"payments"`
	Loans                    []models.Loan                    `json:"loans"`
	LoanPayments             []models.LoanPayment             `json:"loanPayments"`
	Chores                   []models.Chore                   `json:"chores"`
	ChoreAssignments         []models.ChoreAssignment         `json:"choreAssignments"`
	ChoreSettings            *models.ChoreSettings            `json:"choreSettings,omitempty"`
	Notifications            []models.Notification            `json:"notifications"`
	SupplySettings           *models.SupplySettings           `json:"supplySettings,omitempty"`
	SupplyItems              []models.SupplyItem              `json:"supplyItems"`
	SupplyContributions      []models.SupplyContribution      `json:"supplyContributions"`
	RecurringBillTemplates   []models.RecurringBillTemplate   `json:"recurringBillTemplates"`
	RecurringBillAllocations []models.RecurringBillAllocation `json:"recurringBillAllocations"`
}

// ExportAll exports all data from all collections
func (s *BackupService) ExportAll(ctx context.Context) (*BackupData, error) {
	backup := &BackupData{
		Version:    "1.0",
		ExportedAt: time.Now(),
	}

	// Export users (convert to BackupUser to include password hashes)
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	backup.Users = make([]BackupUser, len(users))
	for i, u := range users {
		backup.Users[i] = BackupUser{
			ID:                 u.ID,
			Email:              u.Email,
			Username:           u.Username,
			Name:               u.Name,
			PasswordHash:       u.PasswordHash,
			Role:               u.Role,
			GroupID:            u.GroupID,
			IsActive:           u.IsActive,
			MustChangePassword: u.MustChangePassword,
			TOTPSecret:         u.TOTPSecret,
			CreatedAt:          u.CreatedAt,
		}
	}

	// Export groups
	groups, err := s.groups.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch groups: %w", err)
	}
	backup.Groups = groups

	// Export bills
	bills, err := s.bills.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bills: %w", err)
	}
	backup.Bills = bills

	// Export consumptions
	consumptions, err := s.consumptions.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch consumptions: %w", err)
	}
	backup.Consumptions = consumptions

	// Export payments
	payments, err := s.payments.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payments: %w", err)
	}
	backup.Payments = payments

	// Export loans
	loans, err := s.loans.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch loans: %w", err)
	}
	backup.Loans = loans

	// Export loan payments
	loanPayments, err := s.loanPayments.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch loan payments: %w", err)
	}
	backup.LoanPayments = loanPayments

	// Export chores
	chores, err := s.chores.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chores: %w", err)
	}
	backup.Chores = chores

	// Export chore assignments
	choreAssignments, err := s.choreAssignments.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chore assignments: %w", err)
	}
	backup.ChoreAssignments = choreAssignments

	// Export chore settings (singleton)
	choreSettings, err := s.choreSettings.Get(ctx)
	if err == nil && choreSettings != nil {
		backup.ChoreSettings = choreSettings
	}

	// Export notifications
	notifications, err := s.notifications.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch notifications: %w", err)
	}
	backup.Notifications = notifications

	// Export supply settings (singleton)
	supplySettings, err := s.supplySettings.Get(ctx)
	if err == nil && supplySettings != nil {
		backup.SupplySettings = supplySettings
	}

	// Export supply items
	supplyItems, err := s.supplyItems.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch supply items: %w", err)
	}
	backup.SupplyItems = supplyItems

	// Export supply contributions
	supplyContributions, err := s.supplyContributions.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch supply contributions: %w", err)
	}
	backup.SupplyContributions = supplyContributions

	// Export allocations
	allocations, err := s.allocations.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch allocations: %w", err)
	}
	backup.Allocations = allocations

	// Export recurring bill templates
	recurringBillTemplates, err := s.recurringBillTemplates.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recurring bill templates: %w", err)
	}
	backup.RecurringBillTemplates = recurringBillTemplates

	// Export recurring bill allocations
	recurringBillAllocations, err := s.recurringBillAllocations.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recurring bill allocations: %w", err)
	}
	backup.RecurringBillAllocations = recurringBillAllocations

	return backup, nil
}

// ExportJSON exports backup data as JSON string
func (s *BackupService) ExportJSON(ctx context.Context) ([]byte, error) {
	backup, err := s.ExportAll(ctx)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup to JSON: %w", err)
	}

	return jsonData, nil
}

// ImportJSON imports backup data from JSON string
// WARNING: This is a destructive operation that replaces all existing data
// Returns ImportResult with information about users who got default passwords
func (s *BackupService) ImportJSON(ctx context.Context, jsonData []byte) (*ImportResult, error) {
	var backup BackupData
	if err := json.Unmarshal(jsonData, &backup); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON backup: %w", err)
	}

	result := &ImportResult{
		UsersWithResetPasswords: []string{},
	}

	// Default password for users with missing password hashes
	const defaultPassword = "ChangeMe123!"

	// Disable foreign key checks BEFORE starting transaction (PRAGMA must be outside transaction)
	if _, err := s.db.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return nil, fmt.Errorf("failed to disable foreign keys: %w", err)
	}

	// Start a transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		// Re-enable foreign keys before returning
		s.db.ExecContext(ctx, "PRAGMA foreign_keys = ON")
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		tx.Rollback()
		// Re-enable foreign keys on any exit path
		s.db.ExecContext(ctx, "PRAGMA foreign_keys = ON")
	}()

	// Delete existing data in reverse dependency order
	tablesToClear := []string{
		"loan_payments",
		"payments",
		"consumptions",
		"allocations",
		"chore_assignments",
		"supply_contributions",
		"supply_item_history",
		"notifications",
		"web_push_subscriptions",
		"notification_preferences",
		"bills",
		"recurring_bill_allocations",
		"recurring_bill_templates",
		"supply_items",
		"loans",
		"chores",
		"chore_settings",
		"supply_settings",
		"sessions",
		"password_reset_tokens",
		"passkey_credentials",
		"users",
		"groups",
	}

	for _, table := range tablesToClear {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return nil, fmt.Errorf("failed to clear table %s: %w", table, err)
		}
	}

	// Import groups
	for _, group := range backup.Groups {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO groups (id, name, weight, created_at) VALUES (?, ?, ?, ?)`,
			group.ID, group.Name, group.Weight, group.CreatedAt.UTC().Format(time.RFC3339))
		if err != nil {
			return nil, fmt.Errorf("failed to import group %s: %w", group.ID, err)
		}
	}

	// Import users - handle empty password hashes
	for _, user := range backup.Users {
		var username, totpSecret, groupID *string
		if user.Username != "" {
			username = &user.Username
		}
		if user.TOTPSecret != "" {
			totpSecret = &user.TOTPSecret
		}
		if user.GroupID != nil {
			groupID = user.GroupID
		}

		isActive := 0
		if user.IsActive {
			isActive = 1
		}
		mustChange := 0
		if user.MustChangePassword {
			mustChange = 1
		}

		// Check if password hash is empty or invalid
		passwordHash := user.PasswordHash
		if strings.TrimSpace(passwordHash) == "" {
			// Generate a default password hash for users with missing passwords
			hash, err := utils.HashPassword(defaultPassword)
			if err != nil {
				return nil, fmt.Errorf("failed to hash default password for user %s: %w", user.Email, err)
			}
			passwordHash = hash
			mustChange = 1 // Force password change on first login
			result.UsersWithResetPasswords = append(result.UsersWithResetPasswords, user.Email)
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO users (id, email, username, name, password_hash, role, group_id, is_active, must_change_password, totp_secret, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			user.ID, user.Email, username, user.Name, passwordHash, user.Role, groupID, isActive, mustChange, totpSecret, user.CreatedAt.UTC().Format(time.RFC3339))
		if err != nil {
			return nil, fmt.Errorf("failed to import user %s: %w", user.ID, err)
		}
	}

	// Import bills
	for _, bill := range backup.Bills {
		var paymentDeadline, reopenedAt, totalUnits *string
		if bill.PaymentDeadline != nil {
			pd := bill.PaymentDeadline.UTC().Format(time.RFC3339)
			paymentDeadline = &pd
		}
		if bill.ReopenedAt != nil {
			ra := bill.ReopenedAt.UTC().Format(time.RFC3339)
			reopenedAt = &ra
		}
		if bill.TotalUnits != "" {
			totalUnits = &bill.TotalUnits
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO bills (id, type, custom_type, allocation_type, period_start, period_end, payment_deadline,
				total_amount_pln, total_units, notes, status, reopened_at, reopen_reason, reopened_by, recurring_template_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			bill.ID, bill.Type, bill.CustomType, bill.AllocationType,
			bill.PeriodStart.UTC().Format(time.RFC3339), bill.PeriodEnd.UTC().Format(time.RFC3339),
			paymentDeadline, bill.TotalAmountPLN, totalUnits, bill.Notes, bill.Status,
			reopenedAt, bill.ReopenReason, bill.ReopenedBy, bill.RecurringTemplateID,
			bill.CreatedAt.UTC().Format(time.RFC3339))
		if err != nil {
			return nil, fmt.Errorf("failed to import bill %s: %w", bill.ID, err)
		}
	}

	// Import consumptions
	for _, consumption := range backup.Consumptions {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO consumptions (id, bill_id, subject_type, subject_id, units, meter_value, recorded_at, source)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			consumption.ID, consumption.BillID, consumption.SubjectType, consumption.SubjectID,
			consumption.Units, consumption.MeterValue, consumption.RecordedAt.UTC().Format(time.RFC3339), consumption.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to import consumption %s: %w", consumption.ID, err)
		}
	}

	// Import payments
	for _, payment := range backup.Payments {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO payments (id, bill_id, payer_user_id, amount_pln, paid_at, method, reference)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			payment.ID, payment.BillID, payment.PayerUserID, payment.AmountPLN,
			payment.PaidAt.UTC().Format(time.RFC3339), payment.Method, payment.Reference)
		if err != nil {
			return nil, fmt.Errorf("failed to import payment %s: %w", payment.ID, err)
		}
	}

	// Import loans
	for _, loan := range backup.Loans {
		var dueDate *string
		if loan.DueDate != nil {
			dd := loan.DueDate.UTC().Format(time.RFC3339)
			dueDate = &dd
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO loans (id, lender_id, borrower_id, amount_pln, note, due_date, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			loan.ID, loan.LenderID, loan.BorrowerID, loan.AmountPLN, loan.Note, dueDate, loan.Status,
			loan.CreatedAt.UTC().Format(time.RFC3339))
		if err != nil {
			return nil, fmt.Errorf("failed to import loan %s: %w", loan.ID, err)
		}
	}

	// Import loan payments
	for _, lp := range backup.LoanPayments {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO loan_payments (id, loan_id, amount_pln, paid_at, note)
			VALUES (?, ?, ?, ?, ?)`,
			lp.ID, lp.LoanID, lp.AmountPLN, lp.PaidAt.UTC().Format(time.RFC3339), lp.Note)
		if err != nil {
			return nil, fmt.Errorf("failed to import loan payment %s: %w", lp.ID, err)
		}
	}

	// Import chores
	for _, chore := range backup.Chores {
		isActive := 0
		if chore.IsActive {
			isActive = 1
		}
		notificationsEnabled := 0
		if chore.NotificationsEnabled {
			notificationsEnabled = 1
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO chores (id, name, description, frequency, custom_interval, difficulty, priority, assignment_mode, notifications_enabled, reminder_hours, is_active, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			chore.ID, chore.Name, chore.Description, chore.Frequency, chore.CustomInterval,
			chore.Difficulty, chore.Priority, chore.AssignmentMode, notificationsEnabled,
			chore.ReminderHours, isActive, chore.CreatedAt.UTC().Format(time.RFC3339))
		if err != nil {
			return nil, fmt.Errorf("failed to import chore %s: %w", chore.ID, err)
		}
	}

	// Import chore assignments
	for _, ca := range backup.ChoreAssignments {
		var completedAt *string
		if ca.CompletedAt != nil {
			ct := ca.CompletedAt.UTC().Format(time.RFC3339)
			completedAt = &ct
		}
		isOnTime := 0
		if ca.IsOnTime {
			isOnTime = 1
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO chore_assignments (id, chore_id, assignee_user_id, due_date, status, completed_at, points, is_on_time)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			ca.ID, ca.ChoreID, ca.AssigneeUserID, ca.DueDate.UTC().Format(time.RFC3339),
			ca.Status, completedAt, ca.Points, isOnTime)
		if err != nil {
			return nil, fmt.Errorf("failed to import chore assignment %s: %w", ca.ID, err)
		}
	}

	// Import chore settings
	if backup.ChoreSettings != nil {
		cs := backup.ChoreSettings
		globalNotifications := 0
		if cs.GlobalNotifications {
			globalNotifications = 1
		}
		pointsEnabled := 0
		if cs.PointsEnabled {
			pointsEnabled = 1
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO chore_settings (id, default_assignment_mode, global_notifications, default_reminder_hours, points_enabled, points_multiplier, updated_at)
			VALUES ('singleton', ?, ?, ?, ?, ?, ?)`,
			cs.DefaultAssignmentMode, globalNotifications, cs.DefaultReminderHours,
			pointsEnabled, cs.PointsMultiplier, cs.UpdatedAt.UTC().Format(time.RFC3339))
		if err != nil {
			return nil, fmt.Errorf("failed to import chore settings: %w", err)
		}
	}

	// Import notifications
	for _, n := range backup.Notifications {
		var sentAt *string
		if n.SentAt != nil {
			sa := n.SentAt.UTC().Format(time.RFC3339)
			sentAt = &sa
		}
		read := 0
		if n.Read {
			read = 1
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO notifications (id, channel, template_id, scheduled_for, sent_at, status, read, user_id, title, body)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			n.ID, n.Channel, n.TemplateID, n.ScheduledFor.UTC().Format(time.RFC3339),
			sentAt, n.Status, read, n.UserID, n.Title, n.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to import notification %s: %w", n.ID, err)
		}
	}

	// Import supply settings
	if backup.SupplySettings != nil {
		ss := backup.SupplySettings
		isActive := 0
		if ss.IsActive {
			isActive = 1
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO supply_settings (id, weekly_contribution_pln, contribution_day, current_budget_pln, last_contribution_at, is_active, created_at, updated_at)
			VALUES ('singleton', ?, ?, ?, ?, ?, ?, ?)`,
			ss.WeeklyContributionPLN, ss.ContributionDay, ss.CurrentBudgetPLN,
			ss.LastContributionAt.UTC().Format(time.RFC3339), isActive,
			ss.CreatedAt.UTC().Format(time.RFC3339), ss.UpdatedAt.UTC().Format(time.RFC3339))
		if err != nil {
			return nil, fmt.Errorf("failed to import supply settings: %w", err)
		}
	}

	// Import supply items
	for _, item := range backup.SupplyItems {
		var lastRestockedAt, lastRestockedByUserID, lastRestockAmountPLN *string
		if item.LastRestockedAt != nil {
			lra := item.LastRestockedAt.UTC().Format(time.RFC3339)
			lastRestockedAt = &lra
		}
		if item.LastRestockedByUserID != nil {
			lastRestockedByUserID = item.LastRestockedByUserID
		}
		if item.LastRestockAmountPLN != nil {
			lastRestockAmountPLN = item.LastRestockAmountPLN
		}
		needsRefund := 0
		if item.NeedsRefund {
			needsRefund = 1
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO supply_items (id, name, category, current_quantity, min_quantity, unit, priority, added_by_user_id, added_at, last_restocked_at, last_restocked_by_user_id, last_restock_amount_pln, needs_refund, notes)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.Name, item.Category, item.CurrentQuantity, item.MinQuantity,
			item.Unit, item.Priority, item.AddedByUserID, item.AddedAt.UTC().Format(time.RFC3339),
			lastRestockedAt, lastRestockedByUserID, lastRestockAmountPLN, needsRefund, item.Notes)
		if err != nil {
			return nil, fmt.Errorf("failed to import supply item %s: %w", item.ID, err)
		}
	}

	// Import supply contributions
	for _, sc := range backup.SupplyContributions {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO supply_contributions (id, user_id, amount_pln, period_start, period_end, type, notes, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			sc.ID, sc.UserID, sc.AmountPLN, sc.PeriodStart.UTC().Format(time.RFC3339),
			sc.PeriodEnd.UTC().Format(time.RFC3339), sc.Type, sc.Notes,
			sc.CreatedAt.UTC().Format(time.RFC3339))
		if err != nil {
			return nil, fmt.Errorf("failed to import supply contribution %s: %w", sc.ID, err)
		}
	}

	// Import allocations (bill cost splits)
	for _, alloc := range backup.Allocations {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO allocations (id, bill_id, subject_type, subject_id, allocated_pln)
			VALUES (?, ?, ?, ?, ?)`,
			alloc.ID, alloc.BillID, alloc.SubjectType, alloc.SubjectID, alloc.AllocatedPLN)
		if err != nil {
			return nil, fmt.Errorf("failed to import allocation %s: %w", alloc.ID, err)
		}
	}

	// Import recurring bill templates
	for _, template := range backup.RecurringBillTemplates {
		isActive := 0
		if template.IsActive {
			isActive = 1
		}
		var lastGeneratedAt *string
		if template.LastGeneratedAt != nil {
			lga := template.LastGeneratedAt.UTC().Format(time.RFC3339)
			lastGeneratedAt = &lga
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO recurring_bill_templates (id, custom_type, frequency, amount, day_of_month, start_date, notes, is_active, current_bill_id, next_due_date, last_generated_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			template.ID, template.CustomType, template.Frequency, template.Amount, template.DayOfMonth,
			template.StartDate.UTC().Format(time.RFC3339), template.Notes, isActive, template.CurrentBillID,
			template.NextDueDate.UTC().Format(time.RFC3339), lastGeneratedAt,
			template.CreatedAt.UTC().Format(time.RFC3339), template.UpdatedAt.UTC().Format(time.RFC3339))
		if err != nil {
			return nil, fmt.Errorf("failed to import recurring bill template %s: %w", template.ID, err)
		}
	}

	// Import recurring bill allocations
	for _, alloc := range backup.RecurringBillAllocations {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO recurring_bill_allocations (id, template_id, subject_type, subject_id, allocation_type, percentage, fraction_numerator, fraction_denominator, fixed_amount)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			alloc.ID, alloc.TemplateID, alloc.SubjectType, alloc.SubjectID, alloc.AllocationType,
			alloc.Percentage, alloc.FractionNum, alloc.FractionDenom, alloc.FixedAmount)
		if err != nil {
			return nil, fmt.Errorf("failed to import recurring bill allocation %s: %w", alloc.ID, err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Re-enable foreign keys after successful commit (defer will also call this, but that's fine)
	s.db.ExecContext(ctx, "PRAGMA foreign_keys = ON")

	// Set default password in result only if there are users with reset passwords
	if len(result.UsersWithResetPasswords) > 0 {
		result.DefaultPassword = defaultPassword
	}

	return result, nil
}
