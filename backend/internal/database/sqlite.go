package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

// SQLiteDB wraps the SQLite database connection
type SQLiteDB struct {
	DB *sqlx.DB
}

// NewSQLiteDB creates a new SQLite database connection
func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
	}

	// Connection string with pragmas for performance and safety
	dsn := fmt.Sprintf("%s?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=10000", dbPath)

	db, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	sqlite := &SQLiteDB{DB: db}

	// Initialize schema
	if err := sqlite.initSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Println("SQLite database initialized successfully")
	return sqlite, nil
}

// initSchema creates all tables if they don't exist
func (s *SQLiteDB) initSchema(ctx context.Context) error {
	schemaSQL, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema.sql: %w", err)
	}

	_, err = s.DB.ExecContext(ctx, string(schemaSQL))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// Run migrations for existing databases
	if err := s.runMigrations(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// runMigrations applies incremental migrations for existing databases
func (s *SQLiteDB) runMigrations(ctx context.Context) error {
	// Migration: Add reminder_rate_limit_per_hour column to app_settings if not exists
	var count int
	err := s.DB.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM pragma_table_info('app_settings')
		WHERE name = 'reminder_rate_limit_per_hour'
	`)
	if err != nil {
		return fmt.Errorf("failed to check app_settings column: %w", err)
	}
	if count == 0 {
		_, err = s.DB.ExecContext(ctx, `
			ALTER TABLE app_settings ADD COLUMN reminder_rate_limit_per_hour INTEGER NOT NULL DEFAULT 1
		`)
		if err != nil {
			return fmt.Errorf("failed to add reminder_rate_limit_per_hour column: %w", err)
		}
		log.Println("Migration: Added reminder_rate_limit_per_hour column to app_settings")
	}

	return nil
}

// Close closes the database connection
func (s *SQLiteDB) Close() error {
	return s.DB.Close()
}

// BeginTx starts a new transaction
func (s *SQLiteDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	return s.DB.BeginTxx(ctx, opts)
}

// Exec executes a query without returning rows
func (s *SQLiteDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.DB.ExecContext(ctx, query, args...)
}

// Get retrieves a single row
func (s *SQLiteDB) Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return s.DB.GetContext(ctx, dest, query, args...)
}

// Select retrieves multiple rows
func (s *SQLiteDB) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return s.DB.SelectContext(ctx, dest, query, args...)
}

// NamedExec executes a named query
func (s *SQLiteDB) NamedExec(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	return s.DB.NamedExecContext(ctx, query, arg)
}

// HasData checks if the database has any user data
func (s *SQLiteDB) HasData(ctx context.Context) bool {
	var count int
	err := s.DB.GetContext(ctx, &count, "SELECT COUNT(*) FROM users")
	if err != nil {
		return false
	}
	return count > 0
}
