package models

import (
	"time"
)

// User represents a system user
type User struct {
	ID                 string              `db:"id" json:"id"`
	Email              string              `db:"email" json:"email"`
	Username           string              `db:"username" json:"username,omitempty"` // Optional unique username for login
	Name               string              `db:"name" json:"name"`
	PasswordHash       string              `db:"password_hash" json:"-"`
	Role               string              `db:"role" json:"role"` // ADMIN, RESIDENT
	GroupID            *string             `db:"group_id" json:"groupId,omitempty"`
	IsActive           bool                `db:"is_active" json:"isActive"`
	MustChangePassword bool                `db:"must_change_password" json:"mustChangePassword"`
	TOTPSecret         string              `db:"totp_secret" json:"-"` // Encrypted TOTP secret for 2FA
	CreatedAt          time.Time           `db:"created_at" json:"createdAt"`
	PasskeyCredentials []PasskeyCredential `db:"-" json:"-"` // Loaded separately from passkey_credentials table
}

// PasskeyCredential represents a WebAuthn credential
type PasskeyCredential struct {
	ID              []byte    `db:"id" json:"id"`
	UserID          string    `db:"user_id" json:"-"`
	PublicKey       []byte    `db:"public_key" json:"publicKey"`
	AttestationType string    `db:"attestation_type" json:"attestationType"`
	AAGUID          []byte    `db:"aaguid" json:"aaguid"`
	SignCount       uint32    `db:"sign_count" json:"signCount"`
	Name            string    `db:"name" json:"name"` // User-friendly name for the credential
	CreatedAt       time.Time `db:"created_at" json:"createdAt"`
	LastUsedAt      time.Time `db:"last_used_at" json:"lastUsedAt"`
	// Store backup flags to handle platform authenticators that sync credentials
	BackupEligible bool `db:"backup_eligible" json:"backupEligible"`
	BackupState    bool `db:"backup_state" json:"backupState"`
}

// Group represents a household group (e.g., couples)
type Group struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Weight    float64   `db:"weight" json:"weight"` // default 1.0
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
}

// Bill represents a utility bill or shared expense
type Bill struct {
	ID                  string     `db:"id" json:"id"`
	Type                string     `db:"type" json:"type"`                                // electricity, gas, internet, inne
	CustomType          *string    `db:"custom_type" json:"customType,omitempty"`         // used when Type is "inne"
	AllocationType      *string    `db:"allocation_type" json:"allocationType,omitempty"` // simple (like gas) or metered (like electricity) - only for "inne"
	PeriodStart         time.Time  `db:"period_start" json:"periodStart"`
	PeriodEnd           time.Time  `db:"period_end" json:"periodEnd"`
	PaymentDeadline     *time.Time `db:"payment_deadline" json:"paymentDeadline,omitempty"` // optional deadline for payment
	TotalAmountPLN      string     `db:"total_amount_pln" json:"totalAmountPLN"`            // Decimal as string
	TotalUnits          string     `db:"total_units" json:"totalUnits,omitempty"`           // Decimal as string
	Notes               *string    `db:"notes" json:"notes,omitempty"`
	Status              string     `db:"status" json:"status"` // draft, posted, closed
	ReopenedAt          *time.Time `db:"reopened_at" json:"reopenedAt,omitempty"`
	ReopenReason        *string    `db:"reopen_reason" json:"reopenReason,omitempty"`
	ReopenedBy          *string    `db:"reopened_by" json:"reopenedBy,omitempty"`
	RecurringTemplateID *string    `db:"recurring_template_id" json:"recurringTemplateId,omitempty"` // link to recurring template if generated
	CreatedAt           time.Time  `db:"created_at" json:"createdAt"`
}

// RecurringBillTemplate represents a template for auto-generating bills
type RecurringBillTemplate struct {
	ID              string                    `db:"id" json:"id"`
	CustomType      string                    `db:"custom_type" json:"customType"`  // name of the bill (e.g., "Netflix", "Rent")
	Frequency       string                    `db:"frequency" json:"frequency"`     // monthly, quarterly, yearly
	Amount          string                    `db:"amount" json:"amount"`           // fixed amount per period (decimal as string)
	DayOfMonth      int                       `db:"day_of_month" json:"dayOfMonth"` // 1-31, day when bill is due
	StartDate       time.Time                 `db:"start_date" json:"startDate"`    // required start date for first bill
	Allocations     []RecurringBillAllocation `db:"-" json:"allocations"`           // Loaded separately
	Notes           *string                   `db:"notes" json:"notes,omitempty"`
	IsActive        bool                      `db:"is_active" json:"isActive"`
	CurrentBillID   *string                   `db:"current_bill_id" json:"currentBillId,omitempty"` // ID of the current active bill
	NextDueDate     time.Time                 `db:"next_due_date" json:"nextDueDate"`               // when next bill should be generated
	LastGeneratedAt *time.Time                `db:"last_generated_at" json:"lastGeneratedAt,omitempty"`
	CreatedAt       time.Time                 `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time                 `db:"updated_at" json:"updatedAt"`
}

// RecurringBillAllocation represents predefined cost split for recurring bills
type RecurringBillAllocation struct {
	ID             string   `db:"id" json:"id"`
	TemplateID     string   `db:"template_id" json:"-"`
	SubjectType    string   `db:"subject_type" json:"subjectType"`                           // user or group
	SubjectID      string   `db:"subject_id" json:"subjectId"`                               // user ID or group ID
	AllocationType string   `db:"allocation_type" json:"allocationType"`                     // "percentage", "fraction", "fixed"
	Percentage     *float64 `db:"percentage" json:"percentage,omitempty"`                    // 0-100, for percentage type
	FractionNum    *int     `db:"fraction_numerator" json:"fractionNumerator,omitempty"`     // numerator for fraction (e.g., 1 in 1/3)
	FractionDenom  *int     `db:"fraction_denominator" json:"fractionDenominator,omitempty"` // denominator for fraction (e.g., 3 in 1/3)
	FixedAmount    *string  `db:"fixed_amount" json:"fixedAmount,omitempty"`                 // fixed PLN amount (decimal as string)
}

// Consumption represents individual usage readings
type Consumption struct {
	ID          string    `db:"id" json:"id"`
	BillID      string    `db:"bill_id" json:"billId"`
	SubjectType string    `db:"subject_type" json:"subjectType"` // "user" or "group"
	SubjectID   string    `db:"subject_id" json:"subjectId"`     // user ID or group ID
	Units       string    `db:"units" json:"units"`              // Decimal as string
	MeterValue  *string   `db:"meter_value" json:"meterValue,omitempty"`
	RecordedAt  time.Time `db:"recorded_at" json:"recordedAt"`
	Source      string    `db:"source" json:"source"` // user, admin
}

// Payment represents a payment towards a bill
type Payment struct {
	ID          string    `db:"id" json:"id"`
	BillID      string    `db:"bill_id" json:"billId"`
	PayerUserID string    `db:"payer_user_id" json:"payerUserId"`
	AmountPLN   string    `db:"amount_pln" json:"amountPLN"` // Decimal as string
	PaidAt      time.Time `db:"paid_at" json:"paidAt"`
	Method      *string   `db:"method" json:"method,omitempty"`
	Reference   *string   `db:"reference" json:"reference,omitempty"`
}

// Loan represents money lent between users
type Loan struct {
	ID         string     `db:"id" json:"id"`
	LenderID   string     `db:"lender_id" json:"lenderId"`
	BorrowerID string     `db:"borrower_id" json:"borrowerId"`
	AmountPLN  string     `db:"amount_pln" json:"amountPLN"` // Decimal as string
	Note       *string    `db:"note" json:"note,omitempty"`
	DueDate    *time.Time `db:"due_date" json:"dueDate,omitempty"`
	Status     string     `db:"status" json:"status"` // open, partial, settled
	CreatedAt  time.Time  `db:"created_at" json:"createdAt"`
}

// LoanPayment represents a partial or full loan repayment
type LoanPayment struct {
	ID        string    `db:"id" json:"id"`
	LoanID    string    `db:"loan_id" json:"loanId"`
	AmountPLN string    `db:"amount_pln" json:"amountPLN"` // Decimal as string
	PaidAt    time.Time `db:"paid_at" json:"paidAt"`
	Note      *string   `db:"note" json:"note,omitempty"`
}

// Chore represents a household task
type Chore struct {
	ID                   string    `db:"id" json:"id"`
	Name                 string    `db:"name" json:"name"`
	Description          *string   `db:"description" json:"description,omitempty"`
	Frequency            string    `db:"frequency" json:"frequency"`                      // daily, weekly, monthly, custom, irregular
	CustomInterval       *int      `db:"custom_interval" json:"customInterval,omitempty"` // days for custom frequency
	Difficulty           int       `db:"difficulty" json:"difficulty"`                    // 1-5 scale
	Priority             int       `db:"priority" json:"priority"`                        // 1-5 scale
	AssignmentMode       string    `db:"assignment_mode" json:"assignmentMode"`           // manual, round_robin, random
	NotificationsEnabled bool      `db:"notifications_enabled" json:"notificationsEnabled"`
	ReminderHours        *int      `db:"reminder_hours" json:"reminderHours,omitempty"` // hours before due
	IsActive             bool      `db:"is_active" json:"isActive"`
	CreatedAt            time.Time `db:"created_at" json:"createdAt"`
}

// ChoreAssignment represents a chore assigned to a user
type ChoreAssignment struct {
	ID             string     `db:"id" json:"id"`
	ChoreID        string     `db:"chore_id" json:"choreId"`
	AssigneeUserID string     `db:"assignee_user_id" json:"assigneeUserId"`
	DueDate        time.Time  `db:"due_date" json:"dueDate"`
	Status         string     `db:"status" json:"status"` // pending, in_progress, done, overdue
	CompletedAt    *time.Time `db:"completed_at" json:"completedAt,omitempty"`
	Points         int        `db:"points" json:"points"`       // points earned for completion
	IsOnTime       bool       `db:"is_on_time" json:"isOnTime"` // completed before due date
}

// ChoreSettings represents global chore system settings
type ChoreSettings struct {
	ID                    string    `db:"id" json:"id"`
	DefaultAssignmentMode string    `db:"default_assignment_mode" json:"defaultAssignmentMode"` // round_robin, random, manual
	GlobalNotifications   bool      `db:"global_notifications" json:"globalNotifications"`
	DefaultReminderHours  int       `db:"default_reminder_hours" json:"defaultReminderHours"`
	PointsEnabled         bool      `db:"points_enabled" json:"pointsEnabled"`
	PointsMultiplier      float64   `db:"points_multiplier" json:"pointsMultiplier"` // base points = difficulty * multiplier
	UpdatedAt             time.Time `db:"updated_at" json:"updatedAt"`
}

// Notification represents an in-app notification
type Notification struct {
	ID           string     `db:"id" json:"id"`
	Channel      string     `db:"channel" json:"channel"` // app
	TemplateID   string     `db:"template_id" json:"templateId"`
	ScheduledFor time.Time  `db:"scheduled_for" json:"scheduledFor"`
	SentAt       *time.Time `db:"sent_at" json:"sentAt,omitempty"`
	Status       string     `db:"status" json:"status"` // queued, sent
	Read         bool       `db:"read" json:"read"`
	UserID       *string    `db:"user_id" json:"userId,omitempty"`
	Title        string     `db:"title" json:"title"`
	Body         string     `db:"body" json:"body"`
}

// SupplySettings represents household supply budget settings (singleton)
type SupplySettings struct {
	ID                    string    `db:"id" json:"id"`
	WeeklyContributionPLN string    `db:"weekly_contribution_pln" json:"weeklyContributionPLN"` // per person (decimal as string)
	ContributionDay       string    `db:"contribution_day" json:"contributionDay"`              // monday, tuesday, etc
	CurrentBudgetPLN      string    `db:"current_budget_pln" json:"currentBudgetPLN"`           // grows over time (decimal as string)
	LastContributionAt    time.Time `db:"last_contribution_at" json:"lastContributionAt"`
	IsActive              bool      `db:"is_active" json:"isActive"`
	CreatedAt             time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt             time.Time `db:"updated_at" json:"updatedAt"`
}

// AppSettings represents application branding/customization settings (singleton)
type AppSettings struct {
	ID                       string    `db:"id" json:"id"`
	AppName                  string    `db:"app_name" json:"appName"`
	DefaultLanguage          string    `db:"default_language" json:"defaultLanguage"`                    // Default locale code (e.g., "en", "pl")
	DisableAutoDetect        bool      `db:"disable_auto_detect" json:"disableAutoDetect"`               // If true, always use default language
	ReminderRateLimitPerHour int       `db:"reminder_rate_limit_per_hour" json:"reminderRateLimitPerHour"` // Max reminders per user per hour (0 = unlimited)
	UpdatedAt                time.Time `db:"updated_at" json:"updatedAt"`
}

// SupplyItem represents a household supply with inventory tracking
type SupplyItem struct {
	ID                    string     `db:"id" json:"id"`
	Name                  string     `db:"name" json:"name"`
	Category              string     `db:"category" json:"category"`                // groceries, cleaning, toiletries, other
	CurrentQuantity       int        `db:"current_quantity" json:"currentQuantity"` // How much is in stock now
	MinQuantity           int        `db:"min_quantity" json:"minQuantity"`         // Threshold for low stock warning
	Unit                  string     `db:"unit" json:"unit"`                        // pcs, kg, L, bottles, boxes, etc.
	Priority              int        `db:"priority" json:"priority"`                // 1-5 (1=low, 5=urgent)
	AddedByUserID         string     `db:"added_by_user_id" json:"addedByUserId"`
	AddedAt               time.Time  `db:"added_at" json:"addedAt"`
	LastRestockedAt       *time.Time `db:"last_restocked_at" json:"lastRestockedAt,omitempty"`
	LastRestockedByUserID *string    `db:"last_restocked_by_user_id" json:"lastRestockedByUserId,omitempty"`
	LastRestockAmountPLN  *string    `db:"last_restock_amount_pln" json:"lastRestockAmountPLN,omitempty"` // decimal as string
	NeedsRefund           bool       `db:"needs_refund" json:"needsRefund"`                               // If last restock awaits reimbursement
	Notes                 *string    `db:"notes" json:"notes,omitempty"`
}

// SupplyContribution represents a budget contribution
type SupplyContribution struct {
	ID          string    `db:"id" json:"id"`
	UserID      string    `db:"user_id" json:"userId"`
	AmountPLN   string    `db:"amount_pln" json:"amountPLN"` // decimal as string
	PeriodStart time.Time `db:"period_start" json:"periodStart"`
	PeriodEnd   time.Time `db:"period_end" json:"periodEnd"`
	Type        string    `db:"type" json:"type"` // weekly_auto, manual, adjustment
	Notes       *string   `db:"notes" json:"notes,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}

// SupplyItemHistory tracks changes to supply items (purchases, usage, etc.)
type SupplyItemHistory struct {
	ID            string    `db:"id" json:"id"`
	SupplyItemID  string    `db:"supply_item_id" json:"supplyItemId"`
	UserID        string    `db:"user_id" json:"userId"`
	Action        string    `db:"action" json:"action"`                // add, remove, restock, purchase, adjust
	QuantityDelta int       `db:"quantity_delta" json:"quantityDelta"` // +/- amount changed
	OldQuantity   int       `db:"old_quantity" json:"oldQuantity"`
	NewQuantity   int       `db:"new_quantity" json:"newQuantity"`
	CostPLN       *string   `db:"cost_pln" json:"costPLN,omitempty"` // for purchases (decimal as string)
	Notes         *string   `db:"notes" json:"notes,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"createdAt"`
}

// Session represents an active user session with a refresh token
type Session struct {
	ID           string    `db:"id" json:"id"`
	UserID       string    `db:"user_id" json:"userId"`
	RefreshToken string    `db:"refresh_token" json:"-"` // Hashed token
	Name         string    `db:"name" json:"name"`       // User-friendly name (e.g., "Chrome on Windows")
	IPAddress    string    `db:"ip_address" json:"ipAddress"`
	UserAgent    string    `db:"user_agent" json:"userAgent"`
	CreatedAt    time.Time `db:"created_at" json:"createdAt"`
	LastUsedAt   time.Time `db:"last_used_at" json:"lastUsedAt"`
	ExpiresAt    time.Time `db:"expires_at" json:"expiresAt"`
}

// PasswordResetToken represents a password reset token for users
type PasswordResetToken struct {
	ID               string     `db:"id" json:"id"`
	UserID           string     `db:"user_id" json:"userId"`
	TokenHash        string     `db:"token_hash" json:"-"` // SHA-256 hash of the token
	ExpiresAt        time.Time  `db:"expires_at" json:"expiresAt"`
	Used             bool       `db:"used" json:"used"`
	UsedAt           *time.Time `db:"used_at" json:"usedAt,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"createdAt"`
	CreatedByAdminID string     `db:"created_by_admin_id" json:"createdByAdminId"`
}

// AuditLog represents a log entry for user/admin actions
type AuditLog struct {
	ID           string                 `db:"id" json:"id"`
	UserID       string                 `db:"user_id" json:"userId"`
	UserEmail    string                 `db:"user_email" json:"userEmail"`
	UserName     string                 `db:"user_name" json:"userName"`
	Action       string                 `db:"action" json:"action"`              // e.g., "user.create", "bill.post", "chore.delete"
	ResourceType string                 `db:"resource_type" json:"resourceType"` // e.g., "user", "bill", "chore"
	ResourceID   *string                `db:"resource_id" json:"resourceId,omitempty"`
	Details      map[string]interface{} `db:"-" json:"details,omitempty"` // Additional context (stored as JSON)
	DetailsJSON  string                 `db:"details" json:"-"`           // JSON string for DB storage
	IPAddress    string                 `db:"ip_address" json:"ipAddress"`
	UserAgent    string                 `db:"user_agent" json:"userAgent"`
	Status       string                 `db:"status" json:"status"` // "success", "failure"
	CreatedAt    time.Time              `db:"created_at" json:"createdAt"`
}

// Permission represents a granular permission for an action
type Permission struct {
	ID          string `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`               // e.g., "chores.create", "bills.post"
	Description string `db:"description" json:"description"` // Human-readable description
	Category    string `db:"category" json:"category"`       // e.g., "chores", "bills", "users"
}

// Role represents a role with associated permissions
type Role struct {
	ID              string    `db:"id" json:"id"`
	Name            string    `db:"name" json:"name"` // e.g., "ADMIN", "RESIDENT", "CUSTOM_ROLE_1"
	DisplayName     string    `db:"display_name" json:"displayName"`
	IsSystem        bool      `db:"is_system" json:"isSystem"` // true for ADMIN/RESIDENT, false for custom roles
	Permissions     []string  `db:"-" json:"permissions"`      // Array of permission names
	PermissionsJSON string    `db:"permissions" json:"-"`      // JSON string for DB storage
	CreatedAt       time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time `db:"updated_at" json:"updatedAt"`
}

// ApprovalRequest represents a pending approval for an action requiring admin approval
type ApprovalRequest struct {
	ID           string                 `db:"id" json:"id"`
	UserID       string                 `db:"user_id" json:"userId"`
	UserEmail    string                 `db:"user_email" json:"userEmail"`
	UserName     string                 `db:"user_name" json:"userName"`
	Action       string                 `db:"action" json:"action"` // e.g., "chore.delete"
	ResourceType string                 `db:"resource_type" json:"resourceType"`
	ResourceID   *string                `db:"resource_id" json:"resourceId,omitempty"`
	Details      map[string]interface{} `db:"-" json:"details,omitempty"`
	DetailsJSON  string                 `db:"details" json:"-"`     // JSON string for DB storage
	Status       string                 `db:"status" json:"status"` // "pending", "approved", "rejected"
	ReviewedBy   *string                `db:"reviewed_by" json:"reviewedBy,omitempty"`
	ReviewedAt   *time.Time             `db:"reviewed_at" json:"reviewedAt,omitempty"`
	ReviewNotes  *string                `db:"review_notes" json:"reviewNotes,omitempty"`
	CreatedAt    time.Time              `db:"created_at" json:"createdAt"`
}

// NotificationPreference represents a user's notification preferences
type NotificationPreference struct {
	ID              string          `db:"id" json:"id"`
	UserID          string          `db:"user_id" json:"userId"`
	Preferences     map[string]bool `db:"-" json:"preferences"`
	PreferencesJSON string          `db:"preferences" json:"-"` // JSON string for DB storage
	AllEnabled      bool            `db:"all_enabled" json:"allEnabled"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updatedAt"`
}

// SentReminder tracks sent reminders to avoid duplicates and for rate limiting
type SentReminder struct {
	ID           string    `db:"id" json:"id"`
	UserID       string    `db:"user_id" json:"userId"`
	ResourceType string    `db:"resource_type" json:"resourceType"` // chore_assignment, bill, loan, supply
	ResourceID   string    `db:"resource_id" json:"resourceId"`
	ReminderType string    `db:"reminder_type" json:"reminderType"` // auto_scheduled, manual
	SentAt       time.Time `db:"sent_at" json:"sentAt"`
}
