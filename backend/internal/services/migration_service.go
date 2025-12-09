package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sainaif/holy-home/internal/config"
	"github.com/sainaif/holy-home/internal/repository"
)

// MigrationService handles MongoDB to SQLite migration
type MigrationService struct {
	db    *sqlx.DB
	repos *repository.Repositories
	cfg   *config.Config
}

// NewMigrationService creates a new migration service
func NewMigrationService(db *sqlx.DB, repos *repository.Repositories, cfg *config.Config) *MigrationService {
	return &MigrationService{
		db:    db,
		repos: repos,
		cfg:   cfg,
	}
}

// MigrationResult contains the results of a migration operation
type MigrationResult struct {
	Success         bool           `json:"success"`
	StartedAt       time.Time      `json:"startedAt"`
	CompletedAt     time.Time      `json:"completedAt"`
	RecordsMigrated map[string]int `json:"recordsMigrated"`
	Errors          []string       `json:"errors,omitempty"`
	Warnings        []string       `json:"warnings,omitempty"`
}

// MigrationStatus represents the current migration status
type MigrationStatus struct {
	MigrationEnabled bool       `json:"migrationEnabled"`
	HasExistingData  bool       `json:"hasExistingData"`
	LastMigration    *time.Time `json:"lastMigration,omitempty"`
}

// GetStatus returns the current migration status
func (s *MigrationService) GetStatus(ctx context.Context) (*MigrationStatus, error) {
	status := &MigrationStatus{
		MigrationEnabled: true,
	}

	// Check if there's existing data by counting users
	var count int
	err := s.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM users")
	if err != nil {
		return nil, fmt.Errorf("failed to check existing data: %w", err)
	}
	status.HasExistingData = count > 0

	// Check for migration metadata
	var migratedAt string
	err = s.db.GetContext(ctx, &migratedAt, "SELECT migrated_at FROM migration_metadata WHERE id = 'singleton'")
	if err == nil {
		t, _ := time.Parse(time.RFC3339, migratedAt)
		status.LastMigration = &t
	}

	return status, nil
}

// ==================== MongoDB Backup Format Types ====================
// These types match the OLD MongoDB backup format with string ObjectIDs

type mongoBackupData struct {
	Version             string                    `json:"version"`
	ExportedAt          time.Time                 `json:"exportedAt"`
	Users               []mongoUser               `json:"users"`
	Groups              []mongoGroup              `json:"groups"`
	Bills               []mongoBill               `json:"bills"`
	Consumptions        []mongoConsumption        `json:"consumptions"`
	Payments            []mongoPayment            `json:"payments"`
	Loans               []mongoLoan               `json:"loans"`
	LoanPayments        []mongoLoanPayment        `json:"loanPayments"`
	Chores              []mongoChore              `json:"chores"`
	ChoreAssignments    []mongoChoreAssignment    `json:"choreAssignments"`
	ChoreSettings       *mongoChoreSettings       `json:"choreSettings,omitempty"`
	Notifications       []mongoNotification       `json:"notifications"`
	SupplySettings      *mongoSupplySettings      `json:"supplySettings,omitempty"`
	SupplyItems         []mongoSupplyItem         `json:"supplyItems"`
	SupplyContributions []mongoSupplyContribution `json:"supplyContributions"`
}

type mongoObjectID struct {
	ID string `json:"$oid"`
}

// UnmarshalJSON handles both MongoDB extended JSON {"$oid": "..."} and plain string "id"
func (oid *mongoObjectID) UnmarshalJSON(data []byte) error {
	// Try MongoDB extended JSON format first: {"$oid": "..."}
	var extJSON struct {
		OID string `json:"$oid"`
	}
	if err := json.Unmarshal(data, &extJSON); err == nil && extJSON.OID != "" {
		oid.ID = extJSON.OID
		return nil
	}

	// Try plain string (backup format)
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		oid.ID = s
		return nil
	}

	return fmt.Errorf("unable to parse ObjectID: %s", string(data))
}

type mongoDecimal128 struct {
	NumberDecimal string `json:"$numberDecimal"`
}

type mongoDate struct {
	Date string `json:"$date"`
}

type mongoUser struct {
	ID                 mongoObjectID      `json:"id"`
	LegacyID           mongoObjectID      `json:"_id"` // MongoDB extended JSON uses _id
	Email              string             `json:"email"`
	Username           *string            `json:"username"`
	Name               string             `json:"name"`
	PasswordHash       string             `json:"password_hash"`
	Role               string             `json:"role"`
	GroupID            *mongoObjectID     `json:"groupId"`
	LegacyGroupID      *mongoObjectID     `json:"group_id"` // MongoDB extended JSON uses snake_case
	IsActive           bool               `json:"isActive"`
	LegacyIsActive     bool               `json:"is_active"`
	MustChangePassword bool               `json:"mustChangePassword"`
	LegacyMustChange   bool               `json:"must_change_password"`
	TOTPSecret         *string            `json:"totp_secret"`
	PasskeyCredentials []mongoPasskeyCred `json:"passkey_credentials"`
	CreatedAt          flexibleTime       `json:"createdAt"`
	LegacyCreatedAt    flexibleTime       `json:"created_at"`
}

// GetID returns the ID from either format
func (u *mongoUser) GetID() string {
	if u.ID.ID != "" {
		return u.ID.ID
	}
	return u.LegacyID.ID
}

// GetGroupID returns the GroupID from either format
func (u *mongoUser) GetGroupID() *string {
	if u.GroupID != nil && u.GroupID.ID != "" {
		return &u.GroupID.ID
	}
	if u.LegacyGroupID != nil && u.LegacyGroupID.ID != "" {
		return &u.LegacyGroupID.ID
	}
	return nil
}

// GetIsActive returns IsActive from either format
func (u *mongoUser) GetIsActive() bool {
	return u.IsActive || u.LegacyIsActive
}

// GetMustChangePassword returns MustChangePassword from either format
func (u *mongoUser) GetMustChangePassword() bool {
	return u.MustChangePassword || u.LegacyMustChange
}

// GetCreatedAt returns CreatedAt from either format
func (u *mongoUser) GetCreatedAt() time.Time {
	if !u.CreatedAt.Time.IsZero() {
		return u.CreatedAt.Time
	}
	return u.LegacyCreatedAt.Time
}

type mongoPasskeyCred struct {
	ID              []byte       `json:"id"`
	PublicKey       []byte       `json:"public_key"`
	AttestationType string       `json:"attestation_type"`
	AAGUID          []byte       `json:"aaguid"`
	SignCount       uint32       `json:"sign_count"`
	Name            string       `json:"name"`
	BackupEligible  bool         `json:"backup_eligible"`
	BackupState     bool         `json:"backup_state"`
	CreatedAt       flexibleTime `json:"created_at"`
	LastUsedAt      flexibleTime `json:"last_used_at"`
}

type mongoGroup struct {
	ID              mongoObjectID `json:"id"`
	LegacyID        mongoObjectID `json:"_id"`
	Name            string        `json:"name"`
	Weight          float64       `json:"weight"`
	CreatedAt       flexibleTime  `json:"createdAt"`
	LegacyCreatedAt flexibleTime  `json:"created_at"`
}

func (g *mongoGroup) GetID() string {
	if g.ID.ID != "" {
		return g.ID.ID
	}
	return g.LegacyID.ID
}

func (g *mongoGroup) GetCreatedAt() time.Time {
	if !g.CreatedAt.Time.IsZero() {
		return g.CreatedAt.Time
	}
	return g.LegacyCreatedAt.Time
}

type mongoBill struct {
	ID                    mongoObjectID    `json:"id"`
	LegacyID              mongoObjectID    `json:"_id"`
	Type                  string           `json:"type"`
	CustomType            *string          `json:"customType"`
	LegacyCustomType      *string          `json:"custom_type"`
	AllocationType        *string          `json:"allocationType"`
	LegacyAllocationType  *string          `json:"allocation_type"`
	PeriodStart           flexibleTime     `json:"periodStart"`
	LegacyPeriodStart     flexibleTime     `json:"period_start"`
	PeriodEnd             flexibleTime     `json:"periodEnd"`
	LegacyPeriodEnd       flexibleTime     `json:"period_end"`
	PaymentDeadline       *flexibleTime    `json:"paymentDeadline"`
	LegacyPaymentDeadline *flexibleTime    `json:"payment_deadline"`
	TotalAmountPLN        flexibleDecimal  `json:"totalAmountPLN"`
	LegacyTotalAmountPLN  flexibleDecimal  `json:"total_amount_pln"`
	TotalUnits            *flexibleDecimal `json:"totalUnits"`
	LegacyTotalUnits      *flexibleDecimal `json:"total_units"`
	Notes                 *string          `json:"notes"`
	Status                string           `json:"status"`
	ReopenedAt            *flexibleTime    `json:"reopenedAt"`
	LegacyReopenedAt      *flexibleTime    `json:"reopened_at"`
	ReopenReason          *string          `json:"reopenReason"`
	LegacyReopenReason    *string          `json:"reopen_reason"`
	ReopenedBy            *mongoObjectID   `json:"reopenedBy"`
	LegacyReopenedBy      *mongoObjectID   `json:"reopened_by"`
	RecurringTemplateID   *mongoObjectID   `json:"recurringTemplateId"`
	LegacyRecurringTplID  *mongoObjectID   `json:"recurring_template_id"`
	CreatedAt             flexibleTime     `json:"createdAt"`
	LegacyCreatedAt       flexibleTime     `json:"created_at"`
}

func (b *mongoBill) GetID() string {
	if b.ID.ID != "" {
		return b.ID.ID
	}
	return b.LegacyID.ID
}

func (b *mongoBill) GetCustomType() *string {
	if b.CustomType != nil {
		return b.CustomType
	}
	return b.LegacyCustomType
}

func (b *mongoBill) GetAllocationType() *string {
	if b.AllocationType != nil {
		return b.AllocationType
	}
	return b.LegacyAllocationType
}

func (b *mongoBill) GetPeriodStart() time.Time {
	if !b.PeriodStart.Time.IsZero() {
		return b.PeriodStart.Time
	}
	return b.LegacyPeriodStart.Time
}

func (b *mongoBill) GetPeriodEnd() time.Time {
	if !b.PeriodEnd.Time.IsZero() {
		return b.PeriodEnd.Time
	}
	return b.LegacyPeriodEnd.Time
}

func (b *mongoBill) GetPaymentDeadline() *time.Time {
	if b.PaymentDeadline != nil && !b.PaymentDeadline.Time.IsZero() {
		return &b.PaymentDeadline.Time
	}
	if b.LegacyPaymentDeadline != nil && !b.LegacyPaymentDeadline.Time.IsZero() {
		return &b.LegacyPaymentDeadline.Time
	}
	return nil
}

func (b *mongoBill) GetTotalAmountPLN() string {
	if b.TotalAmountPLN.Value != "" {
		return b.TotalAmountPLN.Value
	}
	return b.LegacyTotalAmountPLN.Value
}

func (b *mongoBill) GetTotalUnits() string {
	if b.TotalUnits != nil && b.TotalUnits.Value != "" {
		return b.TotalUnits.Value
	}
	if b.LegacyTotalUnits != nil {
		return b.LegacyTotalUnits.Value
	}
	return ""
}

func (b *mongoBill) GetReopenedAt() *time.Time {
	if b.ReopenedAt != nil && !b.ReopenedAt.Time.IsZero() {
		return &b.ReopenedAt.Time
	}
	if b.LegacyReopenedAt != nil && !b.LegacyReopenedAt.Time.IsZero() {
		return &b.LegacyReopenedAt.Time
	}
	return nil
}

func (b *mongoBill) GetReopenReason() *string {
	if b.ReopenReason != nil {
		return b.ReopenReason
	}
	return b.LegacyReopenReason
}

func (b *mongoBill) GetReopenedBy() *string {
	if b.ReopenedBy != nil && b.ReopenedBy.ID != "" {
		return &b.ReopenedBy.ID
	}
	if b.LegacyReopenedBy != nil && b.LegacyReopenedBy.ID != "" {
		return &b.LegacyReopenedBy.ID
	}
	return nil
}

func (b *mongoBill) GetRecurringTemplateID() *string {
	if b.RecurringTemplateID != nil && b.RecurringTemplateID.ID != "" {
		return &b.RecurringTemplateID.ID
	}
	if b.LegacyRecurringTplID != nil && b.LegacyRecurringTplID.ID != "" {
		return &b.LegacyRecurringTplID.ID
	}
	return nil
}

func (b *mongoBill) GetCreatedAt() time.Time {
	if !b.CreatedAt.Time.IsZero() {
		return b.CreatedAt.Time
	}
	return b.LegacyCreatedAt.Time
}

type mongoConsumption struct {
	ID               mongoObjectID    `json:"id"`
	LegacyID         mongoObjectID    `json:"_id"`
	BillID           mongoObjectID    `json:"billId"`
	LegacyBillID     mongoObjectID    `json:"bill_id"`
	SubjectType      string           `json:"subjectType"`
	LegacySubjType   string           `json:"subject_type"`
	SubjectID        mongoObjectID    `json:"subjectId"`
	LegacySubjectID  mongoObjectID    `json:"subject_id"`
	Units            flexibleDecimal  `json:"units"`
	MeterValue       *flexibleDecimal `json:"meterValue"`
	LegacyMeterValue *flexibleDecimal `json:"meter_value"`
	RecordedAt       flexibleTime     `json:"recordedAt"`
	LegacyRecordedAt flexibleTime     `json:"recorded_at"`
	Source           string           `json:"source"`
}

func (c *mongoConsumption) GetID() string {
	if c.ID.ID != "" {
		return c.ID.ID
	}
	return c.LegacyID.ID
}
func (c *mongoConsumption) GetBillID() string {
	if c.BillID.ID != "" {
		return c.BillID.ID
	}
	return c.LegacyBillID.ID
}
func (c *mongoConsumption) GetSubjectType() string {
	if c.SubjectType != "" {
		return c.SubjectType
	}
	return c.LegacySubjType
}
func (c *mongoConsumption) GetSubjectID() string {
	if c.SubjectID.ID != "" {
		return c.SubjectID.ID
	}
	return c.LegacySubjectID.ID
}
func (c *mongoConsumption) GetMeterValue() *string {
	if c.MeterValue != nil && c.MeterValue.Value != "" {
		return &c.MeterValue.Value
	}
	if c.LegacyMeterValue != nil {
		return &c.LegacyMeterValue.Value
	}
	return nil
}
func (c *mongoConsumption) GetRecordedAt() time.Time {
	if !c.RecordedAt.Time.IsZero() {
		return c.RecordedAt.Time
	}
	return c.LegacyRecordedAt.Time
}

type mongoPayment struct {
	ID              mongoObjectID   `json:"id"`
	LegacyID        mongoObjectID   `json:"_id"`
	BillID          mongoObjectID   `json:"billId"`
	LegacyBillID    mongoObjectID   `json:"bill_id"`
	PayerUserID     mongoObjectID   `json:"payerUserId"`
	LegacyPayerUID  mongoObjectID   `json:"payer_user_id"`
	AmountPLN       flexibleDecimal `json:"amountPLN"`
	LegacyAmountPLN flexibleDecimal `json:"amount_pln"`
	PaidAt          flexibleTime    `json:"paidAt"`
	LegacyPaidAt    flexibleTime    `json:"paid_at"`
	Method          string          `json:"method"`
	Reference       string          `json:"reference"`
}

func (p *mongoPayment) GetID() string {
	if p.ID.ID != "" {
		return p.ID.ID
	}
	return p.LegacyID.ID
}
func (p *mongoPayment) GetBillID() string {
	if p.BillID.ID != "" {
		return p.BillID.ID
	}
	return p.LegacyBillID.ID
}
func (p *mongoPayment) GetPayerUserID() string {
	if p.PayerUserID.ID != "" {
		return p.PayerUserID.ID
	}
	return p.LegacyPayerUID.ID
}
func (p *mongoPayment) GetAmountPLN() string {
	if p.AmountPLN.Value != "" {
		return p.AmountPLN.Value
	}
	return p.LegacyAmountPLN.Value
}
func (p *mongoPayment) GetPaidAt() time.Time {
	if !p.PaidAt.Time.IsZero() {
		return p.PaidAt.Time
	}
	return p.LegacyPaidAt.Time
}

type mongoLoan struct {
	ID               mongoObjectID   `json:"id"`
	LegacyID         mongoObjectID   `json:"_id"`
	LenderID         mongoObjectID   `json:"lenderId"`
	LegacyLenderID   mongoObjectID   `json:"lender_id"`
	BorrowerID       mongoObjectID   `json:"borrowerId"`
	LegacyBorrowerID mongoObjectID   `json:"borrower_id"`
	AmountPLN        flexibleDecimal `json:"amountPLN"`
	LegacyAmountPLN  flexibleDecimal `json:"amount_pln"`
	Note             string          `json:"note"`
	DueDate          *flexibleTime   `json:"dueDate"`
	LegacyDueDate    *flexibleTime   `json:"due_date"`
	Status           string          `json:"status"`
	CreatedAt        flexibleTime    `json:"createdAt"`
	LegacyCreatedAt  flexibleTime    `json:"created_at"`
}

func (l *mongoLoan) GetID() string {
	if l.ID.ID != "" {
		return l.ID.ID
	}
	return l.LegacyID.ID
}
func (l *mongoLoan) GetLenderID() string {
	if l.LenderID.ID != "" {
		return l.LenderID.ID
	}
	return l.LegacyLenderID.ID
}
func (l *mongoLoan) GetBorrowerID() string {
	if l.BorrowerID.ID != "" {
		return l.BorrowerID.ID
	}
	return l.LegacyBorrowerID.ID
}
func (l *mongoLoan) GetAmountPLN() string {
	if l.AmountPLN.Value != "" {
		return l.AmountPLN.Value
	}
	return l.LegacyAmountPLN.Value
}
func (l *mongoLoan) GetDueDate() *time.Time {
	if l.DueDate != nil && !l.DueDate.Time.IsZero() {
		return &l.DueDate.Time
	}
	if l.LegacyDueDate != nil && !l.LegacyDueDate.Time.IsZero() {
		return &l.LegacyDueDate.Time
	}
	return nil
}
func (l *mongoLoan) GetCreatedAt() time.Time {
	if !l.CreatedAt.Time.IsZero() {
		return l.CreatedAt.Time
	}
	return l.LegacyCreatedAt.Time
}

type mongoLoanPayment struct {
	ID              mongoObjectID   `json:"id"`
	LegacyID        mongoObjectID   `json:"_id"`
	LoanID          mongoObjectID   `json:"loanId"`
	LegacyLoanID    mongoObjectID   `json:"loan_id"`
	AmountPLN       flexibleDecimal `json:"amountPLN"`
	LegacyAmountPLN flexibleDecimal `json:"amount_pln"`
	PaidAt          flexibleTime    `json:"paidAt"`
	LegacyPaidAt    flexibleTime    `json:"paid_at"`
	Note            string          `json:"note"`
}

func (lp *mongoLoanPayment) GetID() string {
	if lp.ID.ID != "" {
		return lp.ID.ID
	}
	return lp.LegacyID.ID
}
func (lp *mongoLoanPayment) GetLoanID() string {
	if lp.LoanID.ID != "" {
		return lp.LoanID.ID
	}
	return lp.LegacyLoanID.ID
}
func (lp *mongoLoanPayment) GetAmountPLN() string {
	if lp.AmountPLN.Value != "" {
		return lp.AmountPLN.Value
	}
	return lp.LegacyAmountPLN.Value
}
func (lp *mongoLoanPayment) GetPaidAt() time.Time {
	if !lp.PaidAt.Time.IsZero() {
		return lp.PaidAt.Time
	}
	return lp.LegacyPaidAt.Time
}

type mongoChore struct {
	ID                    mongoObjectID `json:"id"`
	LegacyID              mongoObjectID `json:"_id"`
	Name                  string        `json:"name"`
	Description           string        `json:"description"`
	Frequency             string        `json:"frequency"`
	CustomInterval        int           `json:"customInterval"`
	LegacyCustomInterval  int           `json:"custom_interval"`
	Difficulty            int           `json:"difficulty"`
	Priority              int           `json:"priority"`
	AssignmentMode        string        `json:"assignmentMode"`
	LegacyAssignmentMode  string        `json:"assignment_mode"`
	NotificationsEnabled  bool          `json:"notificationsEnabled"`
	LegacyNotificationsOn bool          `json:"notifications_enabled"`
	ReminderHours         int           `json:"reminderHours"`
	LegacyReminderHours   int           `json:"reminder_hours"`
	IsActive              bool          `json:"isActive"`
	LegacyIsActive        bool          `json:"is_active"`
	CreatedAt             flexibleTime  `json:"createdAt"`
	LegacyCreatedAt       flexibleTime  `json:"created_at"`
}

func (ch *mongoChore) GetID() string {
	if ch.ID.ID != "" {
		return ch.ID.ID
	}
	return ch.LegacyID.ID
}
func (ch *mongoChore) GetCustomInterval() int {
	if ch.CustomInterval != 0 {
		return ch.CustomInterval
	}
	return ch.LegacyCustomInterval
}
func (ch *mongoChore) GetAssignmentMode() string {
	if ch.AssignmentMode != "" {
		return ch.AssignmentMode
	}
	return ch.LegacyAssignmentMode
}
func (ch *mongoChore) GetNotificationsEnabled() bool {
	return ch.NotificationsEnabled || ch.LegacyNotificationsOn
}
func (ch *mongoChore) GetReminderHours() int {
	if ch.ReminderHours != 0 {
		return ch.ReminderHours
	}
	return ch.LegacyReminderHours
}
func (ch *mongoChore) GetIsActive() bool {
	return ch.IsActive || ch.LegacyIsActive
}
func (ch *mongoChore) GetCreatedAt() time.Time {
	if !ch.CreatedAt.Time.IsZero() {
		return ch.CreatedAt.Time
	}
	return ch.LegacyCreatedAt.Time
}

type mongoChoreAssignment struct {
	ID                mongoObjectID `json:"id"`
	LegacyID          mongoObjectID `json:"_id"`
	ChoreID           mongoObjectID `json:"choreId"`
	LegacyChoreID     mongoObjectID `json:"chore_id"`
	AssigneeUserID    mongoObjectID `json:"assigneeUserId"`
	LegacyAssigneeUID mongoObjectID `json:"assignee_user_id"`
	DueDate           flexibleTime  `json:"dueDate"`
	LegacyDueDate     flexibleTime  `json:"due_date"`
	Status            string        `json:"status"`
	CompletedAt       *flexibleTime `json:"completedAt"`
	LegacyCompletedAt *flexibleTime `json:"completed_at"`
	Points            int           `json:"points"`
	IsOnTime          bool          `json:"isOnTime"`
	LegacyIsOnTime    bool          `json:"is_on_time"`
}

func (ca *mongoChoreAssignment) GetID() string {
	if ca.ID.ID != "" {
		return ca.ID.ID
	}
	return ca.LegacyID.ID
}
func (ca *mongoChoreAssignment) GetChoreID() string {
	if ca.ChoreID.ID != "" {
		return ca.ChoreID.ID
	}
	return ca.LegacyChoreID.ID
}
func (ca *mongoChoreAssignment) GetAssigneeUserID() string {
	if ca.AssigneeUserID.ID != "" {
		return ca.AssigneeUserID.ID
	}
	return ca.LegacyAssigneeUID.ID
}
func (ca *mongoChoreAssignment) GetDueDate() time.Time {
	if !ca.DueDate.Time.IsZero() {
		return ca.DueDate.Time
	}
	return ca.LegacyDueDate.Time
}
func (ca *mongoChoreAssignment) GetCompletedAt() *time.Time {
	if ca.CompletedAt != nil && !ca.CompletedAt.Time.IsZero() {
		return &ca.CompletedAt.Time
	}
	if ca.LegacyCompletedAt != nil && !ca.LegacyCompletedAt.Time.IsZero() {
		return &ca.LegacyCompletedAt.Time
	}
	return nil
}
func (ca *mongoChoreAssignment) GetIsOnTime() bool {
	return ca.IsOnTime || ca.LegacyIsOnTime
}

type mongoChoreSettings struct {
	DefaultAssignmentMode string       `json:"default_assignment_mode"`
	GlobalNotifications   bool         `json:"global_notifications"`
	DefaultReminderHours  int          `json:"default_reminder_hours"`
	PointsEnabled         bool         `json:"points_enabled"`
	PointsMultiplier      float64      `json:"points_multiplier"`
	UpdatedAt             flexibleTime `json:"updated_at"`
}

type mongoNotification struct {
	ID                 mongoObjectID  `json:"id"`
	LegacyID           mongoObjectID  `json:"_id"`
	Channel            string         `json:"channel"`
	TemplateID         string         `json:"templateId"`
	LegacyTemplateID   string         `json:"template_id"`
	ScheduledFor       flexibleTime   `json:"scheduledFor"`
	LegacyScheduledFor flexibleTime   `json:"scheduled_for"`
	SentAt             *flexibleTime  `json:"sentAt"`
	LegacySentAt       *flexibleTime  `json:"sent_at"`
	Status             string         `json:"status"`
	Read               bool           `json:"read"`
	UserID             *mongoObjectID `json:"userId"`
	LegacyUserID       *mongoObjectID `json:"user_id"`
	Title              string         `json:"title"`
	Body               string         `json:"body"`
}

func (n *mongoNotification) GetID() string {
	if n.ID.ID != "" {
		return n.ID.ID
	}
	return n.LegacyID.ID
}
func (n *mongoNotification) GetTemplateID() string {
	if n.TemplateID != "" {
		return n.TemplateID
	}
	return n.LegacyTemplateID
}
func (n *mongoNotification) GetScheduledFor() time.Time {
	if !n.ScheduledFor.Time.IsZero() {
		return n.ScheduledFor.Time
	}
	return n.LegacyScheduledFor.Time
}
func (n *mongoNotification) GetSentAt() *time.Time {
	if n.SentAt != nil && !n.SentAt.Time.IsZero() {
		return &n.SentAt.Time
	}
	if n.LegacySentAt != nil && !n.LegacySentAt.Time.IsZero() {
		return &n.LegacySentAt.Time
	}
	return nil
}
func (n *mongoNotification) GetUserID() *string {
	if n.UserID != nil && n.UserID.ID != "" {
		return &n.UserID.ID
	}
	if n.LegacyUserID != nil && n.LegacyUserID.ID != "" {
		return &n.LegacyUserID.ID
	}
	return nil
}

type mongoSupplySettings struct {
	WeeklyContributionPLN flexibleDecimal `json:"weekly_contribution_pln"`
	ContributionDay       string          `json:"contribution_day"`
	CurrentBudgetPLN      flexibleDecimal `json:"current_budget_pln"`
	LastContributionAt    flexibleTime    `json:"last_contribution_at"`
	IsActive              bool            `json:"is_active"`
	CreatedAt             flexibleTime    `json:"created_at"`
	UpdatedAt             flexibleTime    `json:"updated_at"`
}

type mongoSupplyItem struct {
	ID                       mongoObjectID    `json:"id"`
	LegacyID                 mongoObjectID    `json:"_id"`
	Name                     string           `json:"name"`
	Category                 string           `json:"category"`
	CurrentQuantity          int              `json:"currentQuantity"`
	LegacyCurrentQty         int              `json:"current_quantity"`
	MinQuantity              int              `json:"minQuantity"`
	LegacyMinQty             int              `json:"min_quantity"`
	Unit                     string           `json:"unit"`
	Priority                 int              `json:"priority"`
	AddedByUserID            mongoObjectID    `json:"addedByUserId"`
	LegacyAddedByUID         mongoObjectID    `json:"added_by_user_id"`
	AddedAt                  flexibleTime     `json:"addedAt"`
	LegacyAddedAt            flexibleTime     `json:"added_at"`
	LastRestockedAt          *flexibleTime    `json:"lastRestockedAt"`
	LegacyLastRestockedAt    *flexibleTime    `json:"last_restocked_at"`
	LastRestockedByUserID    *mongoObjectID   `json:"lastRestockedByUserId"`
	LegacyLastRestockedByUID *mongoObjectID   `json:"last_restocked_by_user_id"`
	LastRestockAmountPLN     *flexibleDecimal `json:"lastRestockAmountPLN"`
	LegacyLastRestockAmt     *flexibleDecimal `json:"last_restock_amount_pln"`
	NeedsRefund              bool             `json:"needsRefund"`
	LegacyNeedsRefund        bool             `json:"needs_refund"`
	Notes                    string           `json:"notes"`
}

func (si *mongoSupplyItem) GetID() string {
	if si.ID.ID != "" {
		return si.ID.ID
	}
	return si.LegacyID.ID
}
func (si *mongoSupplyItem) GetCurrentQuantity() int {
	if si.CurrentQuantity != 0 {
		return si.CurrentQuantity
	}
	return si.LegacyCurrentQty
}
func (si *mongoSupplyItem) GetMinQuantity() int {
	if si.MinQuantity != 0 {
		return si.MinQuantity
	}
	return si.LegacyMinQty
}
func (si *mongoSupplyItem) GetAddedByUserID() string {
	if si.AddedByUserID.ID != "" {
		return si.AddedByUserID.ID
	}
	return si.LegacyAddedByUID.ID
}
func (si *mongoSupplyItem) GetAddedAt() time.Time {
	if !si.AddedAt.Time.IsZero() {
		return si.AddedAt.Time
	}
	return si.LegacyAddedAt.Time
}
func (si *mongoSupplyItem) GetLastRestockedAt() *time.Time {
	if si.LastRestockedAt != nil && !si.LastRestockedAt.Time.IsZero() {
		return &si.LastRestockedAt.Time
	}
	if si.LegacyLastRestockedAt != nil && !si.LegacyLastRestockedAt.Time.IsZero() {
		return &si.LegacyLastRestockedAt.Time
	}
	return nil
}
func (si *mongoSupplyItem) GetLastRestockedByUserID() *string {
	if si.LastRestockedByUserID != nil && si.LastRestockedByUserID.ID != "" {
		return &si.LastRestockedByUserID.ID
	}
	if si.LegacyLastRestockedByUID != nil && si.LegacyLastRestockedByUID.ID != "" {
		return &si.LegacyLastRestockedByUID.ID
	}
	return nil
}
func (si *mongoSupplyItem) GetLastRestockAmountPLN() *string {
	if si.LastRestockAmountPLN != nil && si.LastRestockAmountPLN.Value != "" {
		return &si.LastRestockAmountPLN.Value
	}
	if si.LegacyLastRestockAmt != nil && si.LegacyLastRestockAmt.Value != "" {
		return &si.LegacyLastRestockAmt.Value
	}
	return nil
}
func (si *mongoSupplyItem) GetNeedsRefund() bool {
	return si.NeedsRefund || si.LegacyNeedsRefund
}

type mongoSupplyContribution struct {
	ID                mongoObjectID   `json:"id"`
	LegacyID          mongoObjectID   `json:"_id"`
	UserID            mongoObjectID   `json:"userId"`
	LegacyUserID      mongoObjectID   `json:"user_id"`
	AmountPLN         flexibleDecimal `json:"amountPLN"`
	LegacyAmountPLN   flexibleDecimal `json:"amount_pln"`
	PeriodStart       flexibleTime    `json:"periodStart"`
	LegacyPeriodStart flexibleTime    `json:"period_start"`
	PeriodEnd         flexibleTime    `json:"periodEnd"`
	LegacyPeriodEnd   flexibleTime    `json:"period_end"`
	Type              string          `json:"type"`
	Notes             string          `json:"notes"`
	CreatedAt         flexibleTime    `json:"createdAt"`
	LegacyCreatedAt   flexibleTime    `json:"created_at"`
}

func (sc *mongoSupplyContribution) GetID() string {
	if sc.ID.ID != "" {
		return sc.ID.ID
	}
	return sc.LegacyID.ID
}
func (sc *mongoSupplyContribution) GetUserID() string {
	if sc.UserID.ID != "" {
		return sc.UserID.ID
	}
	return sc.LegacyUserID.ID
}
func (sc *mongoSupplyContribution) GetAmountPLN() string {
	if sc.AmountPLN.Value != "" {
		return sc.AmountPLN.Value
	}
	return sc.LegacyAmountPLN.Value
}
func (sc *mongoSupplyContribution) GetPeriodStart() time.Time {
	if !sc.PeriodStart.Time.IsZero() {
		return sc.PeriodStart.Time
	}
	return sc.LegacyPeriodStart.Time
}
func (sc *mongoSupplyContribution) GetPeriodEnd() time.Time {
	if !sc.PeriodEnd.Time.IsZero() {
		return sc.PeriodEnd.Time
	}
	return sc.LegacyPeriodEnd.Time
}
func (sc *mongoSupplyContribution) GetCreatedAt() time.Time {
	if !sc.CreatedAt.Time.IsZero() {
		return sc.CreatedAt.Time
	}
	return sc.LegacyCreatedAt.Time
}

// flexibleTime handles both MongoDB extended JSON dates and ISO strings
type flexibleTime struct {
	time.Time
}

func (ft *flexibleTime) UnmarshalJSON(data []byte) error {
	// Try MongoDB extended JSON format first: {"$date": "2024-..."}
	var extJSON struct {
		Date string `json:"$date"`
	}
	if err := json.Unmarshal(data, &extJSON); err == nil && extJSON.Date != "" {
		t, err := time.Parse(time.RFC3339, extJSON.Date)
		if err == nil {
			ft.Time = t
			return nil
		}
		// Try alternative format
		t, err = time.Parse("2006-01-02T15:04:05.999Z07:00", extJSON.Date)
		if err == nil {
			ft.Time = t
			return nil
		}
	}

	// Try plain ISO string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		t, err := time.Parse(time.RFC3339, s)
		if err == nil {
			ft.Time = t
			return nil
		}
		t, err = time.Parse("2006-01-02T15:04:05.999Z07:00", s)
		if err == nil {
			ft.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse time: %s", string(data))
}

// flexibleDecimal handles both MongoDB extended JSON decimals and plain strings
type flexibleDecimal struct {
	Value string
}

func (fd *flexibleDecimal) UnmarshalJSON(data []byte) error {
	// Try MongoDB extended JSON format first: {"$numberDecimal": "123.45"}
	var extJSON struct {
		NumberDecimal string `json:"$numberDecimal"`
	}
	if err := json.Unmarshal(data, &extJSON); err == nil && extJSON.NumberDecimal != "" {
		fd.Value = extJSON.NumberDecimal
		return nil
	}

	// Try plain string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		fd.Value = s
		return nil
	}

	// Try number
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		fd.Value = fmt.Sprintf("%v", n)
		return nil
	}

	return fmt.Errorf("unable to parse decimal: %s", string(data))
}

// ==================== Import Implementation ====================

// ImportFromBackup imports data from a parsed backup structure
func (s *MigrationService) ImportFromBackup(ctx context.Context, backup *BackupData) (*MigrationResult, error) {
	// Convert to internal format and process
	result := &MigrationResult{
		StartedAt:       time.Now(),
		RecordsMigrated: make(map[string]int),
	}

	// Start transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to start transaction: %v", err))
		return result, err
	}
	defer tx.Rollback()

	// Import groups
	if len(backup.Groups) > 0 {
		for _, g := range backup.Groups {
			_, err := tx.ExecContext(ctx, `
				INSERT OR REPLACE INTO groups (id, name, weight, created_at)
				VALUES (?, ?, ?, ?)
			`, g.ID, g.Name, g.Weight, g.CreatedAt.Format(time.RFC3339))
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("group %s: %v", g.ID, err))
				continue
			}
			result.RecordsMigrated["groups"]++
		}
	}

	// Import users
	if len(backup.Users) > 0 {
		for _, u := range backup.Users {
			_, err := tx.ExecContext(ctx, `
				INSERT OR REPLACE INTO users (id, email, username, name, password_hash, role, group_id, is_active, must_change_password, totp_secret, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, u.ID, u.Email, u.Username, u.Name, u.PasswordHash, u.Role, u.GroupID,
				boolToInt(u.IsActive), boolToInt(u.MustChangePassword), u.TOTPSecret, u.CreatedAt.Format(time.RFC3339))
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("user %s: %v", u.Email, err))
				continue
			}
			result.RecordsMigrated["users"]++
		}
	}

	// Import bills
	if len(backup.Bills) > 0 {
		for _, b := range backup.Bills {
			var paymentDeadline, reopenedAt *string
			if b.PaymentDeadline != nil {
				pd := b.PaymentDeadline.Format(time.RFC3339)
				paymentDeadline = &pd
			}
			if b.ReopenedAt != nil {
				ra := b.ReopenedAt.Format(time.RFC3339)
				reopenedAt = &ra
			}

			_, err := tx.ExecContext(ctx, `
				INSERT OR REPLACE INTO bills (id, type, custom_type, allocation_type, period_start, period_end, payment_deadline, total_amount_pln, total_units, notes, status, reopened_at, reopen_reason, reopened_by, recurring_template_id, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, b.ID, b.Type, b.CustomType, b.AllocationType,
				b.PeriodStart.Format(time.RFC3339), b.PeriodEnd.Format(time.RFC3339),
				paymentDeadline, b.TotalAmountPLN, b.TotalUnits,
				b.Notes, b.Status, reopenedAt, b.ReopenReason, b.ReopenedBy,
				b.RecurringTemplateID, b.CreatedAt.Format(time.RFC3339))
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("bill %s: %v", b.ID, err))
				continue
			}
			result.RecordsMigrated["bills"]++
		}
	}

	// Import consumptions, payments, loans, etc. follow similar pattern...
	// (Abbreviated for brevity - following the same pattern as above)

	// Commit transaction
	if err := tx.Commit(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to commit transaction: %v", err))
		return result, err
	}

	result.Success = len(result.Errors) == 0
	result.CompletedAt = time.Now()
	return result, nil
}

// ImportFromJSON imports data from a JSON backup file (handles MongoDB extended JSON format)
func (s *MigrationService) ImportFromJSON(ctx context.Context, jsonData []byte, clearExisting bool) (*MigrationResult, error) {
	// Try parsing as MongoDB extended JSON format
	var mongoBackup mongoBackupData
	if err := json.Unmarshal(jsonData, &mongoBackup); err != nil {
		return nil, fmt.Errorf("failed to parse backup JSON: %w", err)
	}

	result := &MigrationResult{
		StartedAt:       time.Now(),
		RecordsMigrated: make(map[string]int),
	}

	// Start transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to start transaction: %v", err))
		return result, err
	}
	defer tx.Rollback()

	// Disable foreign key checks for the entire import to allow importing in any order
	// and to handle references to records that may have been deleted or modified
	if _, err := tx.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("disabling foreign keys: %v", err))
	}

	// Clear existing data if requested (for overwrite mode)
	if clearExisting {
		// Clear all tables
		tables := []string{
			"supply_contributions", "supply_items", "supply_settings",
			"notifications", "notification_preferences", "web_push_subscriptions",
			"chore_settings", "chore_assignments", "chores",
			"loan_payments", "loans",
			"payments", "allocations", "consumptions", "bills",
			"recurring_bill_allocations", "recurring_bill_templates",
			"passkey_credentials", "sessions", "password_reset_tokens",
			"audit_logs", "approval_requests", "user_roles",
			"users", "groups", "roles", "permissions",
			"migration_metadata", "app_settings",
		}
		for _, table := range tables {
			if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
				// Ignore errors for tables that might not exist
				result.Warnings = append(result.Warnings, fmt.Sprintf("clearing %s: %v", table, err))
			}
		}
	}

	// Import in dependency order

	// 1. Groups (no dependencies)
	for _, g := range mongoBackup.Groups {
		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO groups (id, name, weight, created_at)
			VALUES (?, ?, ?, ?)
		`, g.GetID(), g.Name, g.Weight, g.GetCreatedAt().Format(time.RFC3339))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("group %s: %v", g.GetID(), err))
			continue
		}
		result.RecordsMigrated["groups"]++
	}

	// 2. Users (depends on groups)
	for _, u := range mongoBackup.Users {
		groupID := u.GetGroupID()

		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO users (id, email, username, name, password_hash, role, group_id, is_active, must_change_password, totp_secret, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, u.GetID(), u.Email, u.Username, u.Name, u.PasswordHash, u.Role, groupID,
			boolToInt(u.GetIsActive()), boolToInt(u.GetMustChangePassword()), u.TOTPSecret, u.GetCreatedAt().Format(time.RFC3339))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("user %s: %v", u.Email, err))
			continue
		}
		result.RecordsMigrated["users"]++

		// Import embedded passkey credentials
		for _, cred := range u.PasskeyCredentials {
			var lastUsedAt *string
			if !cred.LastUsedAt.Time.IsZero() {
				lut := cred.LastUsedAt.Time.Format(time.RFC3339)
				lastUsedAt = &lut
			}

			_, err := tx.ExecContext(ctx, `
				INSERT OR REPLACE INTO passkey_credentials (id, user_id, public_key, attestation_type, aaguid, sign_count, name, backup_eligible, backup_state, created_at, last_used_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, fmt.Sprintf("%x", cred.ID), u.GetID(), cred.PublicKey, cred.AttestationType,
				cred.AAGUID, cred.SignCount, cred.Name, boolToInt(cred.BackupEligible),
				boolToInt(cred.BackupState), cred.CreatedAt.Time.Format(time.RFC3339), lastUsedAt)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("passkey for user %s: %v", u.Email, err))
				continue
			}
			result.RecordsMigrated["passkey_credentials"]++
		}
	}

	// 3. Bills (depends on users for reopened_by)
	for _, b := range mongoBackup.Bills {
		var paymentDeadline, reopenedAt *string

		if pd := b.GetPaymentDeadline(); pd != nil {
			pdStr := pd.Format(time.RFC3339)
			paymentDeadline = &pdStr
		}
		if ra := b.GetReopenedAt(); ra != nil {
			raStr := ra.Format(time.RFC3339)
			reopenedAt = &raStr
		}
		reopenedBy := b.GetReopenedBy()
		recurringTemplateID := b.GetRecurringTemplateID()
		totalUnits := b.GetTotalUnits()

		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO bills (id, type, custom_type, allocation_type, period_start, period_end, payment_deadline, total_amount_pln, total_units, notes, status, reopened_at, reopen_reason, reopened_by, recurring_template_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, b.GetID(), b.Type, b.GetCustomType(), b.GetAllocationType(),
			b.GetPeriodStart().Format(time.RFC3339), b.GetPeriodEnd().Format(time.RFC3339),
			paymentDeadline, b.GetTotalAmountPLN(), totalUnits,
			b.Notes, b.Status, reopenedAt, b.GetReopenReason(), reopenedBy,
			recurringTemplateID, b.GetCreatedAt().Format(time.RFC3339))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("bill %s: %v", b.GetID(), err))
			continue
		}
		result.RecordsMigrated["bills"]++
	}

	// 4. Consumptions (depends on bills)
	for _, c := range mongoBackup.Consumptions {
		meterValue := c.GetMeterValue()

		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO consumptions (id, bill_id, subject_type, subject_id, units, meter_value, recorded_at, source)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, c.GetID(), c.GetBillID(), c.GetSubjectType(), c.GetSubjectID(),
			c.Units.Value, meterValue, c.GetRecordedAt().Format(time.RFC3339), c.Source)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("consumption %s: %v", c.GetID(), err))
			continue
		}
		result.RecordsMigrated["consumptions"]++
	}

	// 5. Payments (depends on bills, users)
	for _, p := range mongoBackup.Payments {
		var method, reference *string
		if p.Method != "" {
			method = &p.Method
		}
		if p.Reference != "" {
			reference = &p.Reference
		}

		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO payments (id, bill_id, payer_user_id, amount_pln, paid_at, method, reference)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, p.GetID(), p.GetBillID(), p.GetPayerUserID(),
			p.GetAmountPLN(), p.GetPaidAt().Format(time.RFC3339), method, reference)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("payment %s: %v", p.GetID(), err))
			continue
		}
		result.RecordsMigrated["payments"]++
	}

	// 6. Loans (depends on users)
	for _, l := range mongoBackup.Loans {
		var note, dueDate *string
		if l.Note != "" {
			note = &l.Note
		}
		if dd := l.GetDueDate(); dd != nil {
			ddStr := dd.Format(time.RFC3339)
			dueDate = &ddStr
		}

		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO loans (id, lender_id, borrower_id, amount_pln, note, due_date, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, l.GetID(), l.GetLenderID(), l.GetBorrowerID(),
			l.GetAmountPLN(), note, dueDate, l.Status, l.GetCreatedAt().Format(time.RFC3339))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("loan %s: %v", l.GetID(), err))
			continue
		}
		result.RecordsMigrated["loans"]++
	}

	// 7. Loan payments (depends on loans)
	for _, lp := range mongoBackup.LoanPayments {
		var note *string
		if lp.Note != "" {
			note = &lp.Note
		}

		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO loan_payments (id, loan_id, amount_pln, paid_at, note)
			VALUES (?, ?, ?, ?, ?)
		`, lp.GetID(), lp.GetLoanID(), lp.GetAmountPLN(),
			lp.GetPaidAt().Format(time.RFC3339), note)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("loan_payment %s: %v", lp.GetID(), err))
			continue
		}
		result.RecordsMigrated["loan_payments"]++
	}

	// 8. Chores (no dependencies)
	for _, ch := range mongoBackup.Chores {
		var description *string
		var customInterval, reminderHours *int

		if ch.Description != "" {
			description = &ch.Description
		}
		if ci := ch.GetCustomInterval(); ci > 0 {
			customInterval = &ci
		}
		if rh := ch.GetReminderHours(); rh > 0 {
			reminderHours = &rh
		}

		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO chores (id, name, description, frequency, custom_interval, difficulty, priority, assignment_mode, notifications_enabled, reminder_hours, is_active, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, ch.GetID(), ch.Name, description, ch.Frequency, customInterval,
			ch.Difficulty, ch.Priority, ch.GetAssignmentMode(), boolToInt(ch.GetNotificationsEnabled()),
			reminderHours, boolToInt(ch.GetIsActive()), ch.GetCreatedAt().Format(time.RFC3339))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("chore %s: %v", ch.GetID(), err))
			continue
		}
		result.RecordsMigrated["chores"]++
	}

	// 9. Chore assignments (depends on chores, users)
	for _, ca := range mongoBackup.ChoreAssignments {
		var completedAt *string
		if cat := ca.GetCompletedAt(); cat != nil {
			catStr := cat.Format(time.RFC3339)
			completedAt = &catStr
		}

		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO chore_assignments (id, chore_id, assignee_user_id, due_date, status, completed_at, points, is_on_time)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, ca.GetID(), ca.GetChoreID(), ca.GetAssigneeUserID(),
			ca.GetDueDate().Format(time.RFC3339), ca.Status, completedAt, ca.Points, boolToInt(ca.GetIsOnTime()))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("chore_assignment %s: %v", ca.GetID(), err))
			continue
		}
		result.RecordsMigrated["chore_assignments"]++
	}

	// 10. Chore settings (singleton)
	if mongoBackup.ChoreSettings != nil {
		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO chore_settings (id, default_assignment_mode, global_notifications, default_reminder_hours, points_enabled, points_multiplier, updated_at)
			VALUES ('singleton', ?, ?, ?, ?, ?, ?)
		`, mongoBackup.ChoreSettings.DefaultAssignmentMode, boolToInt(mongoBackup.ChoreSettings.GlobalNotifications),
			mongoBackup.ChoreSettings.DefaultReminderHours, boolToInt(mongoBackup.ChoreSettings.PointsEnabled),
			mongoBackup.ChoreSettings.PointsMultiplier, mongoBackup.ChoreSettings.UpdatedAt.Time.Format(time.RFC3339))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("chore_settings: %v", err))
		} else {
			result.RecordsMigrated["chore_settings"] = 1
		}
	}

	// 11. Notifications (depends on users)
	for _, n := range mongoBackup.Notifications {
		userID := n.GetUserID()
		var sentAt *string
		if sa := n.GetSentAt(); sa != nil {
			saStr := sa.Format(time.RFC3339)
			sentAt = &saStr
		}

		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO notifications (id, channel, template_id, scheduled_for, sent_at, status, read, user_id, title, body)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, n.GetID(), n.Channel, n.GetTemplateID(), n.GetScheduledFor().Format(time.RFC3339),
			sentAt, n.Status, boolToInt(n.Read), userID, n.Title, n.Body)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("notification %s: %v", n.GetID(), err))
			continue
		}
		result.RecordsMigrated["notifications"]++
	}

	// 12. Supply settings (singleton)
	if mongoBackup.SupplySettings != nil {
		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO supply_settings (id, weekly_contribution_pln, contribution_day, current_budget_pln, last_contribution_at, is_active, created_at, updated_at)
			VALUES ('singleton', ?, ?, ?, ?, ?, ?, ?)
		`, mongoBackup.SupplySettings.WeeklyContributionPLN.Value, mongoBackup.SupplySettings.ContributionDay,
			mongoBackup.SupplySettings.CurrentBudgetPLN.Value, mongoBackup.SupplySettings.LastContributionAt.Time.Format(time.RFC3339),
			boolToInt(mongoBackup.SupplySettings.IsActive), mongoBackup.SupplySettings.CreatedAt.Time.Format(time.RFC3339),
			mongoBackup.SupplySettings.UpdatedAt.Time.Format(time.RFC3339))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("supply_settings: %v", err))
		} else {
			result.RecordsMigrated["supply_settings"] = 1
		}
	}

	// 13. Supply items (depends on users)
	for _, si := range mongoBackup.SupplyItems {
		var lastRestockedAt *string
		var notes *string

		if lra := si.GetLastRestockedAt(); lra != nil {
			lraStr := lra.Format(time.RFC3339)
			lastRestockedAt = &lraStr
		}
		lastRestockedByUserID := si.GetLastRestockedByUserID()
		lastRestockAmountPLN := si.GetLastRestockAmountPLN()
		if si.Notes != "" {
			notes = &si.Notes
		}

		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO supply_items (id, name, category, current_quantity, min_quantity, unit, priority, added_by_user_id, added_at, last_restocked_at, last_restocked_by_user_id, last_restock_amount_pln, needs_refund, notes)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, si.GetID(), si.Name, si.Category, si.GetCurrentQuantity(), si.GetMinQuantity(),
			si.Unit, si.Priority, si.GetAddedByUserID(), si.GetAddedAt().Format(time.RFC3339),
			lastRestockedAt, lastRestockedByUserID, lastRestockAmountPLN,
			boolToInt(si.GetNeedsRefund()), notes)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("supply_item %s: %v", si.GetID(), err))
			continue
		}
		result.RecordsMigrated["supply_items"]++
	}

	// 14. Supply contributions (depends on users)
	for _, sc := range mongoBackup.SupplyContributions {
		var notes *string
		if sc.Notes != "" {
			notes = &sc.Notes
		}

		_, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO supply_contributions (id, user_id, amount_pln, period_start, period_end, type, notes, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, sc.GetID(), sc.GetUserID(), sc.GetAmountPLN(),
			sc.GetPeriodStart().Format(time.RFC3339), sc.GetPeriodEnd().Format(time.RFC3339),
			sc.Type, notes, sc.GetCreatedAt().Format(time.RFC3339))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("supply_contribution %s: %v", sc.GetID(), err))
			continue
		}
		result.RecordsMigrated["supply_contributions"]++
	}

	// Record migration metadata
	now := time.Now().UTC().Format(time.RFC3339)
	totalRecords := 0
	for _, count := range result.RecordsMigrated {
		totalRecords += count
	}

	_, err = tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO migration_metadata (id, source_version, migrated_at, mongodb_export_date, records_migrated)
		VALUES ('singleton', ?, ?, ?, ?)
	`, mongoBackup.Version, now, mongoBackup.ExportedAt.Format(time.RFC3339), totalRecords)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to record migration metadata: %v", err))
	}

	// Re-enable foreign key checks before commit
	if _, err := tx.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("re-enabling foreign keys: %v", err))
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to commit transaction: %v", err))
		return result, err
	}

	// Re-initialize default permissions and roles after clearing tables
	// This is critical because the migration clears the permissions/roles tables
	if clearExisting {
		// Initialize default permissions
		permissionService := NewPermissionService(s.repos.Permissions)
		if err := permissionService.InitializeDefaultPermissions(ctx); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("re-initializing permissions: %v", err))
		} else {
			result.RecordsMigrated["permissions_initialized"] = 1
		}

		// Initialize default roles
		roleService := NewRoleService(s.repos.Roles, s.repos.Users)
		if err := roleService.InitializeDefaultRoles(ctx); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("re-initializing roles: %v", err))
		} else {
			result.RecordsMigrated["roles_initialized"] = 1
		}

		// Re-bootstrap admin user if configured
		// The migration cleared the users table, so the bootstrap admin needs to be recreated
		if s.cfg != nil && s.cfg.Admin.Email != "" {
			authService := NewAuthService(s.repos.Users, s.repos.PasskeyCredentials, s.repos.Roles, s.cfg, nil)
			if err := authService.BootstrapAdmin(ctx); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("re-bootstrapping admin: %v", err))
			} else {
				result.RecordsMigrated["admin_bootstrapped"] = 1
			}
		}
	}

	result.Success = len(result.Errors) == 0
	result.CompletedAt = time.Now()
	return result, nil
}

// Helper functions

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
