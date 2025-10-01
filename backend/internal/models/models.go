package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a system user
type User struct {
	ID                primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Email             string              `bson:"email" json:"email"`
	Name              string              `bson:"name" json:"name"`
	PasswordHash      string              `bson:"password_hash" json:"-"`
	Role              string              `bson:"role" json:"role"` // ADMIN, RESIDENT
	GroupID           *primitive.ObjectID `bson:"group_id,omitempty" json:"groupId,omitempty"`
	IsActive          bool                `bson:"is_active" json:"isActive"`
	MustChangePassword bool               `bson:"must_change_password" json:"mustChangePassword"`
	CreatedAt         time.Time           `bson:"created_at" json:"createdAt"`
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
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type           string             `bson:"type" json:"type"` // electricity, gas, internet, inne
	CustomType     *string            `bson:"custom_type,omitempty" json:"customType,omitempty"` // used when Type is "inne"
	PeriodStart    time.Time          `bson:"period_start" json:"periodStart"`
	PeriodEnd      time.Time          `bson:"period_end" json:"periodEnd"`
	TotalAmountPLN primitive.Decimal128 `bson:"total_amount_pln" json:"totalAmountPLN"`
	TotalUnits     primitive.Decimal128 `bson:"total_units,omitempty" json:"totalUnits,omitempty"`
	Notes          *string            `bson:"notes,omitempty" json:"notes,omitempty"`
	Status         string             `bson:"status" json:"status"` // draft, posted, closed
	CreatedAt      time.Time          `bson:"created_at" json:"createdAt"`
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

// Allocation represents cost allocation to users/groups
type Allocation struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	BillID      primitive.ObjectID   `bson:"bill_id" json:"billId"`
	SubjectType string               `bson:"subject_type" json:"subjectType"` // user, group
	SubjectID   primitive.ObjectID   `bson:"subject_id" json:"subjectId"`
	AmountPLN   primitive.Decimal128 `bson:"amount_pln" json:"amountPLN"`
	Units       primitive.Decimal128 `bson:"units" json:"units"`
	Method      string               `bson:"method" json:"method"` // proportional, equal, weight, override
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

// Prediction represents a forecast result
type Prediction struct {
	ID                  primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Target              string               `bson:"target" json:"target"` // electricity, gas, shared_budget
	PeriodStart         time.Time            `bson:"period_start" json:"periodStart"`
	PeriodEnd           time.Time            `bson:"period_end" json:"periodEnd"`
	HorizonMonths       int                  `bson:"horizon_months" json:"horizonMonths"`
	PredictedUnits      primitive.Decimal128 `bson:"predicted_units" json:"predictedUnits"`
	PredictedAmountPLN  primitive.Decimal128 `bson:"predicted_amount_pln" json:"predictedAmountPLN"`
	Model               ModelInfo            `bson:"model" json:"model"`
	CreatedFrom         string               `bson:"created_from" json:"createdFrom"` // bills, consumptions
	CreatedAt           time.Time            `bson:"created_at" json:"createdAt"`
}

type ModelInfo struct {
	Name    string `bson:"name" json:"name"`
	Version string `bson:"version" json:"version"`
}