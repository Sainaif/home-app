package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const (
	tokenLength = 32 // 32 bytes = 256 bits
)

// GenerateSecureToken generates a cryptographically secure random token
// Returns a URL-safe base64 encoded string
func GenerateSecureToken() (string, error) {
	bytes := make([]byte, tokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}

	// Use URL-safe base64 encoding (no padding) for use in URLs
	token := base64.RawURLEncoding.EncodeToString(bytes)
	return token, nil
}

// HashToken hashes a token using SHA-256
// Returns the hex-encoded hash string
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}

// ValidateTokenFormat validates that a token string is properly formatted
// It should be a valid base64 URL-safe encoded string
func ValidateTokenFormat(token string) error {
	if len(token) == 0 {
		return fmt.Errorf("token is empty")
	}

	// Decode to verify it's valid base64 URL-safe encoding
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return fmt.Errorf("invalid token format: %w", err)
	}

	// Check if decoded bytes have reasonable length (should be tokenLength)
	if len(decoded) != tokenLength {
		return fmt.Errorf("invalid token length")
	}

	return nil
}
