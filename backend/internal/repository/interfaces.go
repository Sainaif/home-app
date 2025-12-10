package repository

import (
	"context"
	"time"

	"github.com/sainaif/holy-home/internal/models"
)

// UserRepository handles user data operations
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmailOrUsername(ctx context.Context, identifier string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.User, error)
	ListActive(ctx context.Context) ([]models.User, error)
	ListByGroupID(ctx context.Context, groupID string) ([]models.User, error)
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	SetMustChangePassword(ctx context.Context, id string, must bool) error
	UpdateTOTPSecret(ctx context.Context, id, secret string) error
}

// PasskeyCredentialRepository handles passkey credential operations
type PasskeyCredentialRepository interface {
	Create(ctx context.Context, userID string, cred *models.PasskeyCredential) error
	GetByUserID(ctx context.Context, userID string) ([]models.PasskeyCredential, error)
	GetByCredentialID(ctx context.Context, credID []byte) (*models.PasskeyCredential, string, error) // returns cred, userID, error
	UpdateSignCount(ctx context.Context, credID []byte, signCount uint32) error
	UpdateLastUsed(ctx context.Context, credID []byte, lastUsedAt time.Time) error
	Delete(ctx context.Context, userID string, credID []byte) error
	DeleteAllForUser(ctx context.Context, userID string) error
}

// GroupRepository handles group operations
type GroupRepository interface {
	Create(ctx context.Context, group *models.Group) error
	GetByID(ctx context.Context, id string) (*models.Group, error)
	Update(ctx context.Context, group *models.Group) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.Group, error)
}

// BillRepository handles bill operations
type BillRepository interface {
	Create(ctx context.Context, bill *models.Bill) error
	GetByID(ctx context.Context, id string) (*models.Bill, error)
	Update(ctx context.Context, bill *models.Bill) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.Bill, error)
	ListByStatus(ctx context.Context, status string) ([]models.Bill, error)
	ListByType(ctx context.Context, billType string) ([]models.Bill, error)
	ListByPeriod(ctx context.Context, start, end time.Time) ([]models.Bill, error)
	ListFiltered(ctx context.Context, billType *string, from, to *time.Time) ([]models.Bill, error)
	GetByRecurringTemplateID(ctx context.Context, templateID string) (*models.Bill, error)
}

// RecurringBillTemplateRepository handles recurring bill template operations
type RecurringBillTemplateRepository interface {
	Create(ctx context.Context, template *models.RecurringBillTemplate) error
	GetByID(ctx context.Context, id string) (*models.RecurringBillTemplate, error)
	Update(ctx context.Context, template *models.RecurringBillTemplate) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.RecurringBillTemplate, error)
	ListActive(ctx context.Context) ([]models.RecurringBillTemplate, error)
	ListDueBefore(ctx context.Context, date time.Time) ([]models.RecurringBillTemplate, error)
}

// RecurringBillAllocationRepository handles recurring bill allocation operations
type RecurringBillAllocationRepository interface {
	Create(ctx context.Context, templateID string, alloc *models.RecurringBillAllocation) error
	GetByTemplateID(ctx context.Context, templateID string) ([]models.RecurringBillAllocation, error)
	DeleteByTemplateID(ctx context.Context, templateID string) error
	ReplaceForTemplate(ctx context.Context, templateID string, allocs []models.RecurringBillAllocation) error
	List(ctx context.Context) ([]models.RecurringBillAllocation, error)
}

// ConsumptionRepository handles consumption/meter reading operations
type ConsumptionRepository interface {
	Create(ctx context.Context, consumption *models.Consumption) error
	GetByID(ctx context.Context, id string) (*models.Consumption, error)
	Update(ctx context.Context, consumption *models.Consumption) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.Consumption, error)
	ListByBillID(ctx context.Context, billID string) ([]models.Consumption, error)
	ListBySubject(ctx context.Context, subjectType, subjectID string) ([]models.Consumption, error)
	ListFiltered(ctx context.Context, subjectType, subjectID *string, from, to *time.Time) ([]models.Consumption, error)
	DeleteByBillID(ctx context.Context, billID string) error
}

// AllocationRepository handles bill allocation operations
type AllocationRepository interface {
	Create(ctx context.Context, billID, subjectType, subjectID, allocatedPLN string) error
	GetByBillID(ctx context.Context, billID string) ([]Allocation, error)
	DeleteByBillID(ctx context.Context, billID string) error
	List(ctx context.Context) ([]Allocation, error)
}

// Allocation represents a calculated cost allocation (not in models, stored only)
type Allocation struct {
	ID           string `db:"id"`
	BillID       string `db:"bill_id"`
	SubjectType  string `db:"subject_type"`
	SubjectID    string `db:"subject_id"`
	AllocatedPLN string `db:"allocated_pln"`
}

// PaymentRepository handles payment operations
type PaymentRepository interface {
	Create(ctx context.Context, payment *models.Payment) error
	GetByID(ctx context.Context, id string) (*models.Payment, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.Payment, error)
	ListByBillID(ctx context.Context, billID string) ([]models.Payment, error)
	ListByPayerID(ctx context.Context, payerID string) ([]models.Payment, error)
	SumByBillID(ctx context.Context, billID string) (string, error) // Returns decimal string
}

// LoanRepository handles loan operations
type LoanRepository interface {
	Create(ctx context.Context, loan *models.Loan) error
	GetByID(ctx context.Context, id string) (*models.Loan, error)
	Update(ctx context.Context, loan *models.Loan) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.Loan, error)
	ListByLenderID(ctx context.Context, lenderID string) ([]models.Loan, error)
	ListByBorrowerID(ctx context.Context, borrowerID string) ([]models.Loan, error)
	ListByStatus(ctx context.Context, status string) ([]models.Loan, error)
	ListOpenBetweenUsers(ctx context.Context, userA, userB string) ([]models.Loan, error)
}

// LoanPaymentRepository handles loan payment operations
type LoanPaymentRepository interface {
	Create(ctx context.Context, payment *models.LoanPayment) error
	GetByID(ctx context.Context, id string) (*models.LoanPayment, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.LoanPayment, error)
	ListByLoanID(ctx context.Context, loanID string) ([]models.LoanPayment, error)
	SumByLoanID(ctx context.Context, loanID string) (string, error)
}

// ChoreRepository handles chore operations
type ChoreRepository interface {
	Create(ctx context.Context, chore *models.Chore) error
	GetByID(ctx context.Context, id string) (*models.Chore, error)
	Update(ctx context.Context, chore *models.Chore) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.Chore, error)
	ListActive(ctx context.Context) ([]models.Chore, error)
}

// ChoreAssignmentRepository handles chore assignment operations
type ChoreAssignmentRepository interface {
	Create(ctx context.Context, assignment *models.ChoreAssignment) error
	GetByID(ctx context.Context, id string) (*models.ChoreAssignment, error)
	Update(ctx context.Context, assignment *models.ChoreAssignment) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.ChoreAssignment, error)
	ListByChoreID(ctx context.Context, choreID string) ([]models.ChoreAssignment, error)
	ListByAssigneeID(ctx context.Context, assigneeID string) ([]models.ChoreAssignment, error)
	ListByStatus(ctx context.Context, status string) ([]models.ChoreAssignment, error)
	ListFiltered(ctx context.Context, assigneeID, status *string) ([]models.ChoreAssignment, error)
	ListPendingByAssignee(ctx context.Context, assigneeID string) ([]models.ChoreAssignment, error)
	GetLatestByChoreID(ctx context.Context, choreID string) (*models.ChoreAssignment, error)
}

// ChoreSettingsRepository handles chore settings (singleton)
type ChoreSettingsRepository interface {
	Get(ctx context.Context) (*models.ChoreSettings, error)
	Upsert(ctx context.Context, settings *models.ChoreSettings) error
}

// SupplySettingsRepository handles supply settings (singleton)
type SupplySettingsRepository interface {
	Get(ctx context.Context) (*models.SupplySettings, error)
	Upsert(ctx context.Context, settings *models.SupplySettings) error
}

// SupplyItemRepository handles supply item operations
type SupplyItemRepository interface {
	Create(ctx context.Context, item *models.SupplyItem) error
	GetByID(ctx context.Context, id string) (*models.SupplyItem, error)
	Update(ctx context.Context, item *models.SupplyItem) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.SupplyItem, error)
	ListByCategory(ctx context.Context, category string) ([]models.SupplyItem, error)
	ListLowStock(ctx context.Context) ([]models.SupplyItem, error)
}

// SupplyContributionRepository handles supply contribution operations
type SupplyContributionRepository interface {
	Create(ctx context.Context, contribution *models.SupplyContribution) error
	GetByID(ctx context.Context, id string) (*models.SupplyContribution, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.SupplyContribution, error)
	ListByUserID(ctx context.Context, userID string) ([]models.SupplyContribution, error)
	ListByPeriod(ctx context.Context, start, end time.Time) ([]models.SupplyContribution, error)
	SumByUserID(ctx context.Context, userID string) (string, error)
}

// SupplyItemHistoryRepository handles supply item history operations
type SupplyItemHistoryRepository interface {
	Create(ctx context.Context, history *models.SupplyItemHistory) error
	ListBySupplyItemID(ctx context.Context, supplyItemID string) ([]models.SupplyItemHistory, error)
	ListByUserID(ctx context.Context, userID string) ([]models.SupplyItemHistory, error)
}

// SessionRepository handles session operations
type SessionRepository interface {
	Create(ctx context.Context, session *models.Session) error
	GetByID(ctx context.Context, id string) (*models.Session, error)
	GetByRefreshToken(ctx context.Context, tokenHash string) (*models.Session, error)
	Update(ctx context.Context, session *models.Session) error
	Delete(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
	ListByUserID(ctx context.Context, userID string) ([]models.Session, error)
}

// PasswordResetTokenRepository handles password reset token operations
type PasswordResetTokenRepository interface {
	Create(ctx context.Context, token *models.PasswordResetToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error)
	MarkUsed(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
}

// NotificationRepository handles notification operations
type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	GetByID(ctx context.Context, id string) (*models.Notification, error)
	Update(ctx context.Context, notification *models.Notification) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.Notification, error)
	ListByUserID(ctx context.Context, userID string, limit int) ([]models.Notification, error)
	ListUnreadByUserID(ctx context.Context, userID string) ([]models.Notification, error)
	MarkAsRead(ctx context.Context, id string) error
	MarkAllAsReadForUser(ctx context.Context, userID string) error
}

// NotificationPreferenceRepository handles notification preference operations
type NotificationPreferenceRepository interface {
	GetByUserID(ctx context.Context, userID string) (*models.NotificationPreference, error)
	Upsert(ctx context.Context, pref *models.NotificationPreference) error
}

// WebPushSubscriptionRepository handles web push subscription operations
type WebPushSubscriptionRepository interface {
	Create(ctx context.Context, sub *models.WebPushSubscription) error
	GetByEndpoint(ctx context.Context, endpoint string) (*models.WebPushSubscription, error)
	Delete(ctx context.Context, endpoint string) error
	ListByUserID(ctx context.Context, userID string) ([]models.WebPushSubscription, error)
}

// PermissionRepository handles permission operations
type PermissionRepository interface {
	Create(ctx context.Context, permission *models.Permission) error
	GetByName(ctx context.Context, name string) (*models.Permission, error)
	List(ctx context.Context) ([]models.Permission, error)
	ListByCategory(ctx context.Context, category string) ([]models.Permission, error)
}

// RoleRepository handles role operations
type RoleRepository interface {
	Create(ctx context.Context, role *models.Role) error
	GetByID(ctx context.Context, id string) (*models.Role, error)
	GetByName(ctx context.Context, name string) (*models.Role, error)
	Update(ctx context.Context, role *models.Role) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.Role, error)
}

// AuditLogRepository handles audit log operations
type AuditLogRepository interface {
	Create(ctx context.Context, log *models.AuditLog) error
	List(ctx context.Context, limit, offset int) ([]models.AuditLog, error)
	ListByUserID(ctx context.Context, userID string, limit int) ([]models.AuditLog, error)
	ListByAction(ctx context.Context, action string, limit int) ([]models.AuditLog, error)
	ListByResourceType(ctx context.Context, resourceType string, limit int) ([]models.AuditLog, error)
}

// ApprovalRequestRepository handles approval request operations
type ApprovalRequestRepository interface {
	Create(ctx context.Context, request *models.ApprovalRequest) error
	GetByID(ctx context.Context, id string) (*models.ApprovalRequest, error)
	Update(ctx context.Context, request *models.ApprovalRequest) error
	List(ctx context.Context) ([]models.ApprovalRequest, error)
	ListPending(ctx context.Context) ([]models.ApprovalRequest, error)
	ListByUserID(ctx context.Context, userID string) ([]models.ApprovalRequest, error)
}

// AppSettingsRepository handles app settings (singleton)
type AppSettingsRepository interface {
	Get(ctx context.Context) (*models.AppSettings, error)
	Upsert(ctx context.Context, settings *models.AppSettings) error
}

// Repositories aggregates all repository interfaces
type Repositories struct {
	Users                    UserRepository
	PasskeyCredentials       PasskeyCredentialRepository
	Groups                   GroupRepository
	Bills                    BillRepository
	RecurringBillTemplates   RecurringBillTemplateRepository
	RecurringBillAllocations RecurringBillAllocationRepository
	Consumptions             ConsumptionRepository
	Allocations              AllocationRepository
	Payments                 PaymentRepository
	Loans                    LoanRepository
	LoanPayments             LoanPaymentRepository
	Chores                   ChoreRepository
	ChoreAssignments         ChoreAssignmentRepository
	ChoreSettings            ChoreSettingsRepository
	SupplySettings           SupplySettingsRepository
	SupplyItems              SupplyItemRepository
	SupplyContributions      SupplyContributionRepository
	SupplyItemHistory        SupplyItemHistoryRepository
	Sessions                 SessionRepository
	PasswordResetTokens      PasswordResetTokenRepository
	Notifications            NotificationRepository
	NotificationPreferences  NotificationPreferenceRepository
	WebPushSubscriptions     WebPushSubscriptionRepository
	Permissions              PermissionRepository
	Roles                    RoleRepository
	AuditLogs                AuditLogRepository
	ApprovalRequests         ApprovalRequestRepository
	AppSettings              AppSettingsRepository
}
