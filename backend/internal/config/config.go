package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	App     AppConfig
	JWT     JWTConfig
	Admin   AdminConfig
	Auth    AuthConfig
	Mongo   MongoConfig
	Logging LogConfig
	VAPID   VAPIDConfig
}

type VAPIDConfig struct {
	PublicKey  string
	PrivateKey string
}

type AppConfig struct {
	Name    string
	Env     string
	Host    string
	Port    string
	BaseURL string
	Domain  string // For WebAuthn (e.g., "localhost" or "holyhome.app")
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
}

type MongoConfig struct {
	URI      string
	Database string
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
			Name:    getEnv("APP_NAME", "Holy Home"),
			Env:     getEnv("APP_ENV", "production"),
			Host:    getEnv("APP_HOST", "0.0.0.0"),
			Port:    getEnv("APP_PORT", "8080"),
			BaseURL: getEnv("APP_BASE_URL", "http://localhost:8080"),
			Domain:  getEnv("APP_DOMAIN", "localhost"),
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
			TwoFAEnabled:      getEnv("AUTH_2FA_ENABLED", "false") == "true",
			TOTPEncryptionKey: getEnv("TOTP_ENCRYPTION_KEY", ""),
		},
		Mongo: MongoConfig{
			URI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
			Database: getEnv("MONGO_DB", "holyhome"),
		},
		Logging: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		VAPID: VAPIDConfig{
			PublicKey:  getEnv("VAPID_PUBLIC_KEY", ""),
			PrivateKey: getEnv("VAPID_PRIVATE_KEY", ""),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
