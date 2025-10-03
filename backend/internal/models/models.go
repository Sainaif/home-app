package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a system user
type User struct {
	ID                 primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Email              string              `bson:"email" json:"email"`
	Name               string              `bson:"name" json:"name"`
	PasswordHash       string              `bson:"password_hash" json:"-"`
	Role               string              `bson:"role" json:"role"` // ADMIN, RESIDENT
	GroupID            *primitive.ObjectID `bson:"group_id,omitempty" json:"groupId,omitempty"`
	IsActive           bool                `bson:"is_active" json:"isActive"`
	MustChangePassword bool                `bson:"must_change_password" json:"mustChangePassword"`
	CreatedAt          time.Time           `bson:"created_at" json:"createdAt"`
	PasskeyCredentials []PasskeyCredential `bson:"passkey_credentials,omitempty" json:"-"`
}

// PasskeyCredential represents a WebAuthn credential
type PasskeyCredential struct {
	ID              []byte    `bson:"id" json:"id"`
	PublicKey       []byte    `bson:"public_key" json:"publicKey"`
	AttestationType string    `bson:"attestation_type" json:"attestationType"`
	AAGUID          []byte    `bson:"aaguid" json:"aaguid"`
	SignCount       uint32    `bson:"sign_count" json:"signCount"`
	Name            string    `bson:"name" json:"name"` // User-friendly name for the credential
	CreatedAt       time.Time `bson:"created_at" json:"createdAt"`
	LastUsedAt      time.Time `bson:"last_used_at" json:"lastUsedAt"`
	// Store backup flags to handle platform authenticators that sync credentials
	BackupEligible bool `bson:"backup_eligible" json:"backupEligible"`
	BackupState    bool `bson:"backup_state" json:"backupState"`
}

// Group represents a household group (e.g., couples)
type Group struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	Weight    float64            `bson:"weight" json:"weight"` // default 1.0
	CreatedAt time.Time          `bson:"created_at" json:"createdAt"`
}

// Bill represents a utility bill or shared expense
type Bill struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Type           string              `bson:"type" json:"type"` // electricity, gas, internet, inne
	CustomType     *string             `bson:"custom_type,omitempty" json:"customType,omitempty"` // used when Type is "inne"
	AllocationType *string             `bson:"allocation_type,omitempty" json:"allocationType,omitempty"` // simple (like gas) or metered (like electricity) - only for "inne"
	PeriodStart    time.Time           `bson:"period_start" json:"periodStart"`
	PeriodEnd      time.Time           `bson:"period_end" json:"periodEnd"`
	PaymentDeadline *time.Time          `bson:"payment_deadline,omitempty" json:"paymentDeadline,omitempty"` // optional deadline for payment
	TotalAmountPLN primitive.Decimal128 `bson:"total_amount_pln" json:"totalAmountPLN"`
	TotalUnits     primitive.Decimal128 `bson:"total_units,omitempty" json:"totalUnits,omitempty"`
	Notes          *string             `bson:"notes,omitempty" json:"notes,omitempty"`
	Status         string              `bson:"status" json:"status"` // draft, posted, closed
	ReopenedAt     *time.Time          `bson:"reopened_at,omitempty" json:"reopenedAt,omitempty"`
	ReopenReason   *string             `bson:"reopen_reason,omitempty" json:"reopenReason,omitempty"`
	ReopenedBy     *primitive.ObjectID `bson:"reopened_by,omitempty" json:"reopenedBy,omitempty"`
	CreatedAt      time.Time           `bson:"created_at" json:"createdAt"`
}

// Consumption represents individual usage readings
type Consumption struct {
	ID         primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
	BillID     primitive.ObjectID    `bson:"bill_id" json:"billId"`
	UserID     primitive.ObjectID    `bson:"user_id" json:"userId"`
	Units      primitive.Decimal128  `bson:"units" json:"units"`
	MeterValue *primitive.Decimal128 `bson:"meter_value,omitempty" json:"meterValue,omitempty"`
	RecordedAt time.Time             `bson:"recorded_at" json:"recordedAt"`
	Source     string                `bson:"source" json:"source"` // user, admin
}

// Payment represents a payment towards a bill
type Payment struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	BillID      primitive.ObjectID   `bson:"bill_id" json:"billId"`
	PayerUserID primitive.ObjectID   `bson:"payer_user_id" json:"payerUserId"`
	AmountPLN   primitive.Decimal128 `bson:"amount_pln" json:"amountPLN"`
	PaidAt      time.Time            `bson:"paid_at" json:"paidAt"`
	Method      *string              `bson:"method,omitempty" json:"method,omitempty"`
	Reference   *string              `bson:"reference,omitempty" json:"reference,omitempty"`
}

// Loan represents money lent between users
type Loan struct {
	ID         primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	LenderID   primitive.ObjectID   `bson:"lender_id" json:"lenderId"`
	BorrowerID primitive.ObjectID   `bson:"borrower_id" json:"borrowerId"`
	AmountPLN  primitive.Decimal128 `bson:"amount_pln" json:"amountPLN"`
	Note       *string              `bson:"note,omitempty" json:"note,omitempty"`
	DueDate    *time.Time           `bson:"due_date,omitempty" json:"dueDate,omitempty"`
	Status     string               `bson:"status" json:"status"` // open, partial, settled
	CreatedAt  time.Time            `bson:"created_at" json:"createdAt"`
}

// LoanPayment represents a partial or full loan repayment
type LoanPayment struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	LoanID    primitive.ObjectID   `bson:"loan_id" json:"loanId"`
	AmountPLN primitive.Decimal128 `bson:"amount_pln" json:"amountPLN"`
	PaidAt    time.Time            `bson:"paid_at" json:"paidAt"`
	Note      *string              `bson:"note,omitempty" json:"note,omitempty"`
}

// Chore represents a household task
type Chore struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name               string             `bson:"name" json:"name"`
	Description        *string            `bson:"description,omitempty" json:"description,omitempty"`
	Frequency          string             `bson:"frequency" json:"frequency"` // daily, weekly, monthly, custom, irregular
	CustomInterval     *int               `bson:"custom_interval,omitempty" json:"customInterval,omitempty"` // days for custom frequency
	Difficulty         int                `bson:"difficulty" json:"difficulty"` // 1-5 scale
	Priority           int                `bson:"priority" json:"priority"` // 1-5 scale
	AssignmentMode     string             `bson:"assignment_mode" json:"assignmentMode"` // manual, round_robin, random
	NotificationsEnabled bool             `bson:"notifications_enabled" json:"notificationsEnabled"`
	ReminderHours      *int               `bson:"reminder_hours,omitempty" json:"reminderHours,omitempty"` // hours before due
	IsActive           bool               `bson:"is_active" json:"isActive"`
	CreatedAt          time.Time          `bson:"created_at" json:"createdAt"`
}

// ChoreAssignment represents a chore assigned to a user
type ChoreAssignment struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ChoreID        primitive.ObjectID  `bson:"chore_id" json:"choreId"`
	AssigneeUserID primitive.ObjectID  `bson:"assignee_user_id" json:"assigneeUserId"`
	DueDate        time.Time           `bson:"due_date" json:"dueDate"`
	Status         string              `bson:"status" json:"status"` // pending, in_progress, done, overdue
	CompletedAt    *time.Time          `bson:"completed_at,omitempty" json:"completedAt,omitempty"`
	Points         int                 `bson:"points" json:"points"` // points earned for completion
	IsOnTime       bool                `bson:"is_on_time" json:"isOnTime"` // completed before due date
}

// ChoreSettings represents global chore system settings
type ChoreSettings struct {
	ID                     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	DefaultAssignmentMode  string             `bson:"default_assignment_mode" json:"defaultAssignmentMode"` // round_robin, random, manual
	GlobalNotifications    bool               `bson:"global_notifications" json:"globalNotifications"`
	DefaultReminderHours   int                `bson:"default_reminder_hours" json:"defaultReminderHours"`
	PointsEnabled          bool               `bson:"points_enabled" json:"pointsEnabled"`
	PointsMultiplier       float64            `bson:"points_multiplier" json:"pointsMultiplier"` // base points = difficulty * multiplier
	UpdatedAt              time.Time          `bson:"updated_at" json:"updatedAt"`
}

// Notification represents an in-app notification
type Notification struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Channel      string              `bson:"channel" json:"channel"` // app
	TemplateID   string              `bson:"template_id" json:"templateId"`
	ScheduledFor time.Time           `bson:"scheduled_for" json:"scheduledFor"`
	SentAt       *time.Time          `bson:"sent_at,omitempty" json:"sentAt,omitempty"`
	Status       string              `bson:"status" json:"status"` // queued, sent
	UserID       *primitive.ObjectID `bson:"user_id,omitempty" json:"userId,omitempty"`
}

// SupplySettings represents household supply budget settings (singleton)
type SupplySettings struct {
	ID                    primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	WeeklyContributionPLN primitive.Decimal128 `bson:"weekly_contribution_pln" json:"weeklyContributionPLN"` // per person
	ContributionDay       string               `bson:"contribution_day" json:"contributionDay"` // monday, tuesday, etc
	CurrentBudgetPLN      primitive.Decimal128 `bson:"current_budget_pln" json:"currentBudgetPLN"` // grows over time
	LastContributionAt    time.Time            `bson:"last_contribution_at" json:"lastContributionAt"`
	IsActive              bool                 `bson:"is_active" json:"isActive"`
	CreatedAt             time.Time            `bson:"created_at" json:"createdAt"`
	UpdatedAt             time.Time            `bson:"updated_at" json:"updatedAt"`
}

// SupplyItem represents a household supply with inventory tracking
type SupplyItem struct {
	ID                     primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
	Name                   string                `bson:"name" json:"name"`
	Category               string                `bson:"category" json:"category"` // groceries, cleaning, toiletries, other
	CurrentQuantity        int                   `bson:"current_quantity" json:"currentQuantity"` // How much is in stock now
	MinQuantity            int                   `bson:"min_quantity" json:"minQuantity"` // Threshold for low stock warning
	Unit                   string                `bson:"unit" json:"unit"` // pcs, kg, L, bottles, boxes, etc.
	Priority               int                   `bson:"priority" json:"priority"` // 1-5 (1=low, 5=urgent)
	AddedByUserID          primitive.ObjectID    `bson:"added_by_user_id" json:"addedByUserId"`
	AddedAt                time.Time             `bson:"added_at" json:"addedAt"`
	LastRestockedAt        *time.Time            `bson:"last_restocked_at,omitempty" json:"lastRestockedAt,omitempty"`
	LastRestockedByUserID  *primitive.ObjectID   `bson:"last_restocked_by_user_id,omitempty" json:"lastRestockedByUserId,omitempty"`
	LastRestockAmountPLN   *primitive.Decimal128 `bson:"last_restock_amount_pln,omitempty" json:"lastRestockAmountPLN,omitempty"`
	NeedsRefund            bool                  `bson:"needs_refund" json:"needsRefund"` // If last restock awaits reimbursement
	Notes                  *string               `bson:"notes,omitempty" json:"notes,omitempty"`
}

// SupplyContribution represents a budget contribution
type SupplyContribution struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID   `bson:"user_id" json:"userId"`
	AmountPLN   primitive.Decimal128 `bson:"amount_pln" json:"amountPLN"`
	PeriodStart time.Time            `bson:"period_start" json:"periodStart"`
	PeriodEnd   time.Time            `bson:"period_end" json:"periodEnd"`
	Type        string               `bson:"type" json:"type"` // weekly_auto, manual, adjustment
	Notes       *string              `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt   time.Time            `bson:"created_at" json:"createdAt"`
}

// SupplyItemHistory tracks changes to supply items (purchases, usage, etc.)
type SupplyItemHistory struct {
	ID            primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
	SupplyItemID  primitive.ObjectID    `bson:"supply_item_id" json:"supplyItemId"`
	UserID        primitive.ObjectID    `bson:"user_id" json:"userId"`
	Action        string                `bson:"action" json:"action"` // add, remove, restock, purchase, adjust
	QuantityDelta int                   `bson:"quantity_delta" json:"quantityDelta"` // +/- amount changed
	OldQuantity   int                   `bson:"old_quantity" json:"oldQuantity"`
	NewQuantity   int                   `bson:"new_quantity" json:"newQuantity"`
	CostPLN       *primitive.Decimal128 `bson:"cost_pln,omitempty" json:"costPLN,omitempty"` // for purchases
	Notes         *string               `bson:"notes,omitempty" json:"notes,omitempty"`
	CreatedAt     time.Time             `bson:"created_at" json:"createdAt"`
}

// Session represents an active user session with a refresh token
type Session struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       primitive.ObjectID `bson:"user_id" json:"userId"`
	RefreshToken string             `bson:"refresh_token" json:"-"` // Hashed token
	Name         string             `bson:"name" json:"name"`        // User-friendly name (e.g., "Chrome on Windows")
	IPAddress    string             `bson:"ip_address" json:"ipAddress"`
	UserAgent    string             `bson:"user_agent" json:"userAgent"`
	CreatedAt    time.Time          `bson:"created_at" json:"createdAt"`
	LastUsedAt   time.Time          `bson:"last_used_at" json:"lastUsedAt"`
	ExpiresAt    time.Time          `bson:"expires_at" json:"expiresAt"`
}

// AuditLog represents a log entry for user/admin actions
type AuditLog struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID     `bson:"user_id" json:"userId"`
	UserEmail   string                 `bson:"user_email" json:"userEmail"`
	UserName    string                 `bson:"user_name" json:"userName"`
	Action      string                 `bson:"action" json:"action"`           // e.g., "user.create", "bill.post", "chore.delete"
	ResourceType string                `bson:"resource_type" json:"resourceType"` // e.g., "user", "bill", "chore"
	ResourceID  *primitive.ObjectID    `bson:"resource_id,omitempty" json:"resourceId,omitempty"`
	Details     map[string]interface{} `bson:"details,omitempty" json:"details,omitempty"` // Additional context
	IPAddress   string                 `bson:"ip_address" json:"ipAddress"`
	UserAgent   string                 `bson:"user_agent" json:"userAgent"`
	Status      string                 `bson:"status" json:"status"` // "success", "failure"
	CreatedAt   time.Time              `bson:"created_at" json:"createdAt"`
}

// Permission represents a granular permission for an action
type Permission struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`               // e.g., "chores.create", "bills.post"
	Description string             `bson:"description" json:"description"` // Human-readable description
	Category    string             `bson:"category" json:"category"`       // e.g., "chores", "bills", "users"
}

// Role represents a role with associated permissions
type Role struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name        string               `bson:"name" json:"name"` // e.g., "ADMIN", "RESIDENT", "CUSTOM_ROLE_1"
	DisplayName string               `bson:"display_name" json:"displayName"`
	IsSystem    bool                 `bson:"is_system" json:"isSystem"` // true for ADMIN/RESIDENT, false for custom roles
	Permissions []string             `bson:"permissions" json:"permissions"` // Array of permission names
	CreatedAt   time.Time            `bson:"created_at" json:"createdAt"`
	UpdatedAt   time.Time            `bson:"updated_at" json:"updatedAt"`
}

// ApprovalRequest represents a pending approval for an action requiring admin approval
type ApprovalRequest struct {
	ID           primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID       primitive.ObjectID     `bson:"user_id" json:"userId"`
	UserEmail    string                 `bson:"user_email" json:"userEmail"`
	UserName     string                 `bson:"user_name" json:"userName"`
	Action       string                 `bson:"action" json:"action"` // e.g., "chore.delete"
	ResourceType string                 `bson:"resource_type" json:"resourceType"`
	ResourceID   *primitive.ObjectID    `bson:"resource_id,omitempty" json:"resourceId,omitempty"`
	Details      map[string]interface{} `bson:"details,omitempty" json:"details,omitempty"`
	Status       string                 `bson:"status" json:"status"` // "pending", "approved", "rejected"
	ReviewedBy   *primitive.ObjectID    `bson:"reviewed_by,omitempty" json:"reviewedBy,omitempty"`
	ReviewedAt   *time.Time             `bson:"reviewed_at,omitempty" json:"reviewedAt,omitempty"`
	ReviewNotes  *string                `bson:"review_notes,omitempty" json:"reviewNotes,omitempty"`
	CreatedAt    time.Time              `bson:"created_at" json:"createdAt"`
}