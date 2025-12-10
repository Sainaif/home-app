package sqlite

import (
	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/repository"
)

// NewRepositories creates all SQLite repository implementations
func NewRepositories(db *sqlx.DB) *repository.Repositories {
	return &repository.Repositories{
		Users:                    NewUserRepository(db),
		PasskeyCredentials:       NewPasskeyCredentialRepository(db),
		Groups:                   NewGroupRepository(db),
		Bills:                    NewBillRepository(db),
		RecurringBillTemplates:   NewRecurringBillTemplateRepository(db),
		RecurringBillAllocations: NewRecurringBillAllocationRepository(db),
		Consumptions:             NewConsumptionRepository(db),
		Allocations:              NewAllocationRepository(db),
		Payments:                 NewPaymentRepository(db),
		Loans:                    NewLoanRepository(db),
		LoanPayments:             NewLoanPaymentRepository(db),
		Chores:                   NewChoreRepository(db),
		ChoreAssignments:         NewChoreAssignmentRepository(db),
		ChoreSettings:            NewChoreSettingsRepository(db),
		SupplySettings:           NewSupplySettingsRepository(db),
		SupplyItems:              NewSupplyItemRepository(db),
		SupplyContributions:      NewSupplyContributionRepository(db),
		SupplyItemHistory:        NewSupplyItemHistoryRepository(db),
		Sessions:                 NewSessionRepository(db),
		PasswordResetTokens:      NewPasswordResetTokenRepository(db),
		Notifications:            NewNotificationRepository(db),
		NotificationPreferences:  NewNotificationPreferenceRepository(db),
		WebPushSubscriptions:     NewWebPushSubscriptionRepository(db),
		Permissions:              NewPermissionRepository(db),
		Roles:                    NewRoleRepository(db),
		AuditLogs:                NewAuditLogRepository(db),
		ApprovalRequests:         NewApprovalRequestRepository(db),
		AppSettings:              NewAppSettingsRepository(db),
		SentReminders:            NewSentReminderRepository(db),
	}
}
