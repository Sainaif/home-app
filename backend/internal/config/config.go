package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	App           AppConfig
	JWT           JWTConfig
	Admin         AdminConfig
	Auth          AuthConfig
	Mongo         MongoConfig
	SQLite        SQLiteConfig
	Logging       LogConfig
	VAPID         VAPIDConfig
	MigrationMode bool // v1.5 bridge release: enables MongoDB->SQLite migration UI
}

type VAPIDConfig struct {
	PublicKey  string
	PrivateKey string
}

type AppConfig struct {
	Name           string
	Env            string
	Host           string
	Port           string
	BaseURL        string
	Domain         string // For WebAuthn (e.g., "localhost" or "holyhome.app")
	AllowedOrigins string // CORS allowed origins, defaults to "*" if not set
}

type JWTConfig struct {
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	Secret        string
	RefreshSecret string
}

type AdminConfig struct {
	Email        string
	PasswordHash string
}

type AuthConfig struct {
	TwoFAEnabled      bool
	TOTPEncryptionKey string // 32-byte key for AES-256 encryption of TOTP secrets
	// Login identifier options
	AllowEmailLogin    bool // Allow login with email (default: true)
	AllowUsernameLogin bool // Allow login with username (default: false)
	// Registration options
	RequireUsername bool // Require username during registration (default: false)
}

type MongoConfig struct {
	URI      string
	Database string
}

type SQLiteConfig struct {
	DatabasePath string // Path to SQLite database file
}

type LogConfig struct {
	Level  string
	Format string
}

func Load() (*Config, error) {
	accessTTL, err := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}

	refreshTTL, err := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "720h")) // 30 days
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
	}

	return &Config{
		App: AppConfig{
			Name:           getEnv("APP_NAME", "Holy Home"),
			Env:            getEnv("APP_ENV", "production"),
			Host:           getEnv("APP_HOST", "0.0.0.0"),
			Port:           getEnv("APP_PORT", "8080"),
			BaseURL:        getEnv("APP_BASE_URL", "http://localhost:8080"),
			Domain:         getEnv("APP_DOMAIN", "localhost"),
			AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
		},
		JWT: JWTConfig{
			AccessTTL:     accessTTL,
			RefreshTTL:    refreshTTL,
			Secret:        getEnv("JWT_SECRET", ""),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", ""),
		},
		Admin: AdminConfig{
			Email:        getEnv("ADMIN_EMAIL", ""),
			PasswordHash: getEnv("ADMIN_PASSWORD_HASH", getEnv("ADMIN_PASSWORD", "")),
		},
		Auth: AuthConfig{
			TwoFAEnabled:       getEnv("AUTH_2FA_ENABLED", "false") == "true",
			TOTPEncryptionKey:  getEnv("TOTP_ENCRYPTION_KEY", ""),
			AllowEmailLogin:    getEnv("AUTH_ALLOW_EMAIL_LOGIN", "true") == "true",
			AllowUsernameLogin: getEnv("AUTH_ALLOW_USERNAME_LOGIN", "false") == "true",
			RequireUsername:    getEnv("AUTH_REQUIRE_USERNAME", "false") == "true",
		},
		Mongo: MongoConfig{
			URI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
			Database: getEnv("MONGO_DB", "holyhome"),
		},
		SQLite: SQLiteConfig{
			DatabasePath: getEnv("DATABASE_PATH", "./holyhome.db"),
		},
		Logging: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		VAPID: VAPIDConfig{
			PublicKey:  getEnv("VAPID_PUBLIC_KEY", ""),
			PrivateKey: getEnv("VAPID_PRIVATE_KEY", ""),
		},
		MigrationMode: getEnv("MIGRATION_MODE", "false") == "true",
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
