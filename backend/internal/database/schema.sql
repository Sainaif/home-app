-- Holy Home SQLite Schema
-- Version: 1.5 (Bridge Release)
-- This schema migrates from MongoDB document store to SQLite relational database

-- ============================================
-- CORE ENTITIES
-- ============================================

-- Groups table (must be created before users due to FK)
CREATE TABLE IF NOT EXISTS groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    weight REAL NOT NULL DEFAULT 1.0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE COLLATE NOCASE,
    username TEXT UNIQUE COLLATE NOCASE,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'RESIDENT',
    group_id TEXT REFERENCES groups(id) ON DELETE SET NULL,
    is_active INTEGER NOT NULL DEFAULT 1,
    must_change_password INTEGER NOT NULL DEFAULT 0,
    totp_secret TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_group_id ON users(group_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Passkey credentials (extracted from MongoDB embedded array)
CREATE TABLE IF NOT EXISTS passkey_credentials (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key BLOB NOT NULL,
    attestation_type TEXT NOT NULL,
    aaguid BLOB NOT NULL,
    sign_count INTEGER NOT NULL DEFAULT 0,
    name TEXT NOT NULL,
    backup_eligible INTEGER NOT NULL DEFAULT 0,
    backup_state INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    last_used_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_passkey_user_id ON passkey_credentials(user_id);

-- ============================================
-- BILLS & FINANCIAL
-- ============================================

-- Recurring bill templates (must be created before bills due to FK)
CREATE TABLE IF NOT EXISTS recurring_bill_templates (
    id TEXT PRIMARY KEY,
    custom_type TEXT NOT NULL,
    frequency TEXT NOT NULL,
    amount TEXT NOT NULL,
    day_of_month INTEGER NOT NULL,
    start_date TEXT NOT NULL,
    notes TEXT,
    is_active INTEGER NOT NULL DEFAULT 1,
    current_bill_id TEXT,
    next_due_date TEXT NOT NULL,
    last_generated_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Bills table
CREATE TABLE IF NOT EXISTS bills (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    custom_type TEXT,
    allocation_type TEXT,
    period_start TEXT NOT NULL,
    period_end TEXT NOT NULL,
    payment_deadline TEXT,
    total_amount_pln TEXT NOT NULL,
    total_units TEXT,
    notes TEXT,
    status TEXT NOT NULL DEFAULT 'draft',
    reopened_at TEXT,
    reopen_reason TEXT,
    reopened_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    recurring_template_id TEXT REFERENCES recurring_bill_templates(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_bills_type_period ON bills(type, period_start);
CREATE INDEX IF NOT EXISTS idx_bills_status ON bills(status);
CREATE INDEX IF NOT EXISTS idx_bills_recurring_template ON bills(recurring_template_id);

-- Add FK from recurring_bill_templates to bills after bills table exists
-- (SQLite doesn't support adding FK after table creation, so current_bill_id is just TEXT)

-- Recurring bill allocations (extracted from MongoDB embedded array)
CREATE TABLE IF NOT EXISTS recurring_bill_allocations (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL REFERENCES recurring_bill_templates(id) ON DELETE CASCADE,
    subject_type TEXT NOT NULL,
    subject_id TEXT NOT NULL,
    allocation_type TEXT NOT NULL,
    percentage REAL,
    fraction_numerator INTEGER,
    fraction_denominator INTEGER,
    fixed_amount TEXT
);

CREATE INDEX IF NOT EXISTS idx_recurring_alloc_template ON recurring_bill_allocations(template_id);

-- Consumptions (meter readings)
CREATE TABLE IF NOT EXISTS consumptions (
    id TEXT PRIMARY KEY,
    bill_id TEXT NOT NULL REFERENCES bills(id) ON DELETE CASCADE,
    subject_type TEXT NOT NULL,
    subject_id TEXT NOT NULL,
    units TEXT NOT NULL,
    meter_value TEXT,
    recorded_at TEXT NOT NULL DEFAULT (datetime('now')),
    source TEXT NOT NULL DEFAULT 'user'
);

CREATE INDEX IF NOT EXISTS idx_consumptions_bill ON consumptions(bill_id);
CREATE INDEX IF NOT EXISTS idx_consumptions_subject ON consumptions(subject_type, subject_id, recorded_at);

-- Allocations (calculated cost splits)
CREATE TABLE IF NOT EXISTS allocations (
    id TEXT PRIMARY KEY,
    bill_id TEXT NOT NULL REFERENCES bills(id) ON DELETE CASCADE,
    subject_type TEXT NOT NULL,
    subject_id TEXT NOT NULL,
    allocated_pln TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_allocations_bill_subject ON allocations(bill_id, subject_type, subject_id);

-- Payments
CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    bill_id TEXT NOT NULL REFERENCES bills(id) ON DELETE CASCADE,
    payer_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_pln TEXT NOT NULL,
    paid_at TEXT NOT NULL DEFAULT (datetime('now')),
    method TEXT,
    reference TEXT
);

CREATE INDEX IF NOT EXISTS idx_payments_bill ON payments(bill_id);
CREATE INDEX IF NOT EXISTS idx_payments_payer ON payments(payer_user_id);

-- ============================================
-- LOANS
-- ============================================

CREATE TABLE IF NOT EXISTS loans (
    id TEXT PRIMARY KEY,
    lender_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    borrower_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_pln TEXT NOT NULL,
    note TEXT,
    due_date TEXT,
    status TEXT NOT NULL DEFAULT 'open',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_loans_lender ON loans(lender_id);
CREATE INDEX IF NOT EXISTS idx_loans_borrower ON loans(borrower_id);
CREATE INDEX IF NOT EXISTS idx_loans_status ON loans(status);

CREATE TABLE IF NOT EXISTS loan_payments (
    id TEXT PRIMARY KEY,
    loan_id TEXT NOT NULL REFERENCES loans(id) ON DELETE CASCADE,
    amount_pln TEXT NOT NULL,
    paid_at TEXT NOT NULL DEFAULT (datetime('now')),
    note TEXT
);

CREATE INDEX IF NOT EXISTS idx_loan_payments_loan ON loan_payments(loan_id);

-- ============================================
-- CHORES
-- ============================================

CREATE TABLE IF NOT EXISTS chores (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    frequency TEXT NOT NULL,
    custom_interval INTEGER,
    difficulty INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 1,
    assignment_mode TEXT NOT NULL DEFAULT 'manual',
    notifications_enabled INTEGER NOT NULL DEFAULT 1,
    reminder_hours INTEGER,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS chore_assignments (
    id TEXT PRIMARY KEY,
    chore_id TEXT NOT NULL REFERENCES chores(id) ON DELETE CASCADE,
    assignee_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    due_date TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    completed_at TEXT,
    points INTEGER NOT NULL DEFAULT 0,
    is_on_time INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_chore_assign_chore ON chore_assignments(chore_id);
CREATE INDEX IF NOT EXISTS idx_chore_assign_user ON chore_assignments(assignee_user_id);
CREATE INDEX IF NOT EXISTS idx_chore_assign_due ON chore_assignments(due_date);
CREATE INDEX IF NOT EXISTS idx_chore_assign_status ON chore_assignments(status);

-- Chore settings (singleton - one row max)
CREATE TABLE IF NOT EXISTS chore_settings (
    id TEXT PRIMARY KEY DEFAULT 'singleton',
    default_assignment_mode TEXT NOT NULL DEFAULT 'round_robin',
    global_notifications INTEGER NOT NULL DEFAULT 1,
    default_reminder_hours INTEGER NOT NULL DEFAULT 24,
    points_enabled INTEGER NOT NULL DEFAULT 1,
    points_multiplier REAL NOT NULL DEFAULT 1.0,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ============================================
-- SUPPLIES
-- ============================================

-- Supply settings (singleton)
CREATE TABLE IF NOT EXISTS supply_settings (
    id TEXT PRIMARY KEY DEFAULT 'singleton',
    weekly_contribution_pln TEXT NOT NULL DEFAULT '0',
    contribution_day TEXT NOT NULL DEFAULT 'monday',
    current_budget_pln TEXT NOT NULL DEFAULT '0',
    last_contribution_at TEXT NOT NULL DEFAULT (datetime('now')),
    is_active INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS supply_items (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    current_quantity INTEGER NOT NULL DEFAULT 0,
    min_quantity INTEGER NOT NULL DEFAULT 0,
    unit TEXT NOT NULL DEFAULT 'pcs',
    priority INTEGER NOT NULL DEFAULT 1,
    added_by_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    added_at TEXT NOT NULL DEFAULT (datetime('now')),
    last_restocked_at TEXT,
    last_restocked_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    last_restock_amount_pln TEXT,
    needs_refund INTEGER NOT NULL DEFAULT 0,
    notes TEXT
);

CREATE INDEX IF NOT EXISTS idx_supply_items_category ON supply_items(category);
CREATE INDEX IF NOT EXISTS idx_supply_items_priority ON supply_items(priority);

CREATE TABLE IF NOT EXISTS supply_contributions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_pln TEXT NOT NULL,
    period_start TEXT NOT NULL,
    period_end TEXT NOT NULL,
    type TEXT NOT NULL,
    notes TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_supply_contrib_user_period ON supply_contributions(user_id, period_start);

CREATE TABLE IF NOT EXISTS supply_item_history (
    id TEXT PRIMARY KEY,
    supply_item_id TEXT NOT NULL REFERENCES supply_items(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    quantity_delta INTEGER NOT NULL,
    old_quantity INTEGER NOT NULL,
    new_quantity INTEGER NOT NULL,
    cost_pln TEXT,
    notes TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_supply_history_item ON supply_item_history(supply_item_id);

-- ============================================
-- SESSIONS & AUTH
-- ============================================

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token TEXT NOT NULL,
    name TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    user_agent TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    last_used_at TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    used INTEGER NOT NULL DEFAULT 0,
    used_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    created_by_admin_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_reset_tokens_user ON password_reset_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_reset_tokens_expires ON password_reset_tokens(expires_at);

-- ============================================
-- NOTIFICATIONS
-- ============================================

CREATE TABLE IF NOT EXISTS notifications (
    id TEXT PRIMARY KEY,
    channel TEXT NOT NULL DEFAULT 'app',
    template_id TEXT NOT NULL,
    scheduled_for TEXT NOT NULL,
    sent_at TEXT,
    status TEXT NOT NULL DEFAULT 'queued',
    read INTEGER NOT NULL DEFAULT 0,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);

CREATE TABLE IF NOT EXISTS notification_preferences (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    preferences TEXT NOT NULL DEFAULT '{}',
    all_enabled INTEGER NOT NULL DEFAULT 1,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS web_push_subscriptions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL UNIQUE,
    expiration_time TEXT,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_web_push_user ON web_push_subscriptions(user_id);

-- ============================================
-- PERMISSIONS & ROLES
-- ============================================

CREATE TABLE IF NOT EXISTS permissions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    category TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_permissions_category ON permissions(category);

CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    is_system INTEGER NOT NULL DEFAULT 0,
    permissions TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ============================================
-- AUDIT & APPROVALS
-- ============================================

CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    user_email TEXT NOT NULL,
    user_name TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    details TEXT,
    ip_address TEXT NOT NULL,
    user_agent TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at);

CREATE TABLE IF NOT EXISTS approval_requests (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_email TEXT NOT NULL,
    user_name TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    details TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    reviewed_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TEXT,
    review_notes TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_approval_status ON approval_requests(status);
CREATE INDEX IF NOT EXISTS idx_approval_user ON approval_requests(user_id);

-- ============================================
-- APP SETTINGS (singleton)
-- ============================================

CREATE TABLE IF NOT EXISTS app_settings (
    id TEXT PRIMARY KEY DEFAULT 'singleton',
    app_name TEXT NOT NULL DEFAULT 'Holy Home',
    default_language TEXT NOT NULL DEFAULT 'en',
    disable_auto_detect INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ============================================
-- MIGRATION METADATA (v1.5 only)
-- ============================================

CREATE TABLE IF NOT EXISTS migration_metadata (
    id TEXT PRIMARY KEY DEFAULT 'singleton',
    source_version TEXT NOT NULL,
    migrated_at TEXT NOT NULL DEFAULT (datetime('now')),
    mongodb_export_date TEXT,
    records_migrated INTEGER NOT NULL DEFAULT 0
);
