package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
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

// HasData checks if the database has any user data (for migration check)
func (s *SQLiteDB) HasData(ctx context.Context) bool {
	var count int
	err := s.DB.GetContext(ctx, &count, "SELECT COUNT(*) FROM users")
	if err != nil {
		return false
	}
	return count > 0
}

// GetMigrationMetadata returns migration info if exists
func (s *SQLiteDB) GetMigrationMetadata(ctx context.Context) (*MigrationMetadata, error) {
	var meta MigrationMetadata
	err := s.DB.GetContext(ctx, &meta, "SELECT * FROM migration_metadata WHERE id = 'singleton'")
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// MigrationMetadata stores info about the MongoDB migration
type MigrationMetadata struct {
	ID                string  `db:"id"`
	SourceVersion     string  `db:"source_version"`
	MigratedAt        string  `db:"migrated_at"`
	MongoDBExportDate *string `db:"mongodb_export_date"`
	RecordsMigrated   int     `db:"records_migrated"`
}
