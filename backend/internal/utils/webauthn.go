package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sainaif/holy-home/internal/models"
)

// WebAuthnUser wraps our User model to implement webauthn.User interface
type WebAuthnUser struct {
	User *models.User
}

// WebAuthnID returns the user's ID as bytes (required by webauthn.User)
func (u WebAuthnUser) WebAuthnID() []byte {
	return []byte(u.User.ID.Hex())
}

// WebAuthnName returns the user's email (required by webauthn.User)
func (u WebAuthnUser) WebAuthnName() string {
	return u.User.Email
}

// WebAuthnDisplayName returns the user's display name (required by webauthn.User)
func (u WebAuthnUser) WebAuthnDisplayName() string {
	if u.User.Name != "" {
		return u.User.Name
	}
	return u.User.Email
}

// WebAuthnCredentials returns the user's credentials (required by webauthn.User)
func (u WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	credentials := make([]webauthn.Credential, len(u.User.PasskeyCredentials))
	for i, cred := range u.User.PasskeyCredentials {
		credentials[i] = webauthn.Credential{
			ID:              cred.ID,
			PublicKey:       cred.PublicKey,
			AttestationType: cred.AttestationType,
			Authenticator: webauthn.Authenticator{
				AAGUID:    cred.AAGUID,
				SignCount: cred.SignCount,
			},
		}
	}
	return credentials
}

// WebAuthnIcon returns an optional icon URL (required by webauthn.User)
func (u WebAuthnUser) WebAuthnIcon() string {
	return ""
}

// NewWebAuthn creates a new WebAuthn instance
func NewWebAuthn(rpID, rpOrigin, rpName string) (*webauthn.WebAuthn, error) {
	// For localhost development, we need to handle port stripping
	// WebAuthn spec allows localhost for testing purposes
	origins := []string{rpOrigin}

	// Add additional localhost origins for development
	if rpID == "localhost" {
		origins = append(origins,
			"http://localhost:16161",  // Frontend
			"http://localhost:16162",  // API
			"http://localhost:3000",   // Local API
			"http://localhost:5173",   // Vite dev server
		)
	}

	wconfig := &webauthn.Config{
		RPDisplayName: rpName,
		RPID:          rpID,
		RPOrigins:     origins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			RequireResidentKey: protocol.ResidentKeyNotRequired(),
			ResidentKey:        protocol.ResidentKeyRequirementRequired, // Required for discoverable credentials
			UserVerification:   protocol.VerificationRequired,           // Required for better security
		},
		AttestationPreference: protocol.PreferNoAttestation,
	}

	return webauthn.New(wconfig)
}

// GenerateChallenge creates a random challenge for WebAuthn
func GenerateChallenge() ([]byte, error) {
	challenge := make([]byte, 32)
	_, err := rand.Read(challenge)
	if err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}
	return challenge, nil
}

// EncodeChallenge encodes a challenge to base64url
func EncodeChallenge(challenge []byte) string {
	return base64.RawURLEncoding.EncodeToString(challenge)
}

// DecodeChallenge decodes a base64url encoded challenge
func DecodeChallenge(encoded string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(encoded)
}

// ParseCredentialCreationResponse parses the credential creation response from the client
func ParseCredentialCreationResponse(body []byte) (*protocol.ParsedCredentialCreationData, error) {
	var ccr protocol.CredentialCreationResponse
	if err := json.Unmarshal(body, &ccr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credential creation response: %w", err)
	}

	parsed, err := ccr.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse credential creation response: %w", err)
	}

	return parsed, nil
}

// ParseCredentialRequestResponse parses the credential assertion response from the client
func ParseCredentialRequestResponse(body []byte) (*protocol.ParsedCredentialAssertionData, error) {
	var car protocol.CredentialAssertionResponse
	if err := json.Unmarshal(body, &car); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credential assertion response: %w", err)
	}

	parsed, err := car.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse credential assertion response: %w", err)
	}

	return parsed, nil
}

// ValidateOrigin checks if the origin is allowed
func ValidateOrigin(origin string, allowedOrigins []string) error {
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return nil
		}
	}
	return errors.New("origin not allowed")
}

// ConvertWebAuthnCredential converts a webauthn.Credential to our models.PasskeyCredential
func ConvertWebAuthnCredential(cred *webauthn.Credential, name string) models.PasskeyCredential {
	now := time.Now()
	return models.PasskeyCredential{
		ID:              cred.ID,
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		AAGUID:          cred.Authenticator.AAGUID,
		SignCount:       cred.Authenticator.SignCount,
		Name:            name,
		CreatedAt:       now,
		LastUsedAt:      now,
	}
}
