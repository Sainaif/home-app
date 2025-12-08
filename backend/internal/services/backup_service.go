package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
)

type BackupService struct {
	users               repository.UserRepository
	groups              repository.GroupRepository
	bills               repository.BillRepository
	consumptions        repository.ConsumptionRepository
	payments            repository.PaymentRepository
	loans               repository.LoanRepository
	loanPayments        repository.LoanPaymentRepository
	chores              repository.ChoreRepository
	choreAssignments    repository.ChoreAssignmentRepository
	choreSettings       repository.ChoreSettingsRepository
	notifications       repository.NotificationRepository
	supplySettings      repository.SupplySettingsRepository
	supplyItems         repository.SupplyItemRepository
	supplyContributions repository.SupplyContributionRepository
}

func NewBackupService(
	users repository.UserRepository,
	groups repository.GroupRepository,
	bills repository.BillRepository,
	consumptions repository.ConsumptionRepository,
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
) *BackupService {
	return &BackupService{
		users:               users,
		groups:              groups,
		bills:               bills,
		consumptions:        consumptions,
		payments:            payments,
		loans:               loans,
		loanPayments:        loanPayments,
		chores:              chores,
		choreAssignments:    choreAssignments,
		choreSettings:       choreSettings,
		notifications:       notifications,
		supplySettings:      supplySettings,
		supplyItems:         supplyItems,
		supplyContributions: supplyContributions,
	}
}

// BackupData represents a complete system backup
type BackupData struct {
	Version             string                      `json:"version"`
	ExportedAt          time.Time                   `json:"exportedAt"`
	Users               []models.User               `json:"users"`
	Groups              []models.Group              `json:"groups"`
	Bills               []models.Bill               `json:"bills"`
	Consumptions        []models.Consumption        `json:"consumptions"`
	Payments            []models.Payment            `json:"payments"`
	Loans               []models.Loan               `json:"loans"`
	LoanPayments        []models.LoanPayment        `json:"loanPayments"`
	Chores              []models.Chore              `json:"chores"`
	ChoreAssignments    []models.ChoreAssignment    `json:"choreAssignments"`
	ChoreSettings       *models.ChoreSettings       `json:"choreSettings,omitempty"`
	Notifications       []models.Notification       `json:"notifications"`
	SupplySettings      *models.SupplySettings      `json:"supplySettings,omitempty"`
	SupplyItems         []models.SupplyItem         `json:"supplyItems"`
	SupplyContributions []models.SupplyContribution `json:"supplyContributions"`
}

// ExportAll exports all data from all collections
func (s *BackupService) ExportAll(ctx context.Context) (*BackupData, error) {
	backup := &BackupData{
		Version:    "1.0",
		ExportedAt: time.Now(),
	}

	// Export users
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	backup.Users = users

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
// NOTE: Import functionality will be handled by the migration service for SQLite
// This method is kept for API compatibility but the actual import logic
// should use the migration service for proper transaction handling
func (s *BackupService) ImportJSON(ctx context.Context, jsonData []byte) error {
	var backup BackupData
	if err := json.Unmarshal(jsonData, &backup); err != nil {
		return fmt.Errorf("failed to unmarshal JSON backup: %w", err)
	}

	// For SQLite, import should be handled by migration service with proper transactions
	// This is a placeholder that returns an error directing users to use migration
	return fmt.Errorf("import functionality is handled by the migration service - use /api/migrate/import endpoint")
}
