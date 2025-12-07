package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sainaif/holy-home/internal/config"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/utils"
)

type AuthService struct {
	db             *mongo.Database
	cfg            *config.Config
	webAuthn       *webauthn.WebAuthn
	sessions       map[string]*webauthn.SessionData // In-memory session storage (use Redis in production)
	sessionsMu     sync.RWMutex                     // Mutex to protect concurrent access to sessions map
	sessionService *SessionService
}

func NewAuthService(db *mongo.Database, cfg *config.Config, sessionService *SessionService) *AuthService {
	// Initialize WebAuthn with configuration
	wa, err := utils.NewWebAuthn(
		cfg.App.Domain,
		cfg.App.BaseURL,
		cfg.App.Name,
	)
	if err != nil {
		// Log error but don't fail - passkeys are optional
		fmt.Printf("Warning: Failed to initialize WebAuthn: %v\n", err)
	}

	return &AuthService{
		db:             db,
		cfg:            cfg,
		webAuthn:       wa,
		sessions:       make(map[string]*webauthn.SessionData),
		sessionService: sessionService,
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TOTPCode string `json:"totpCode,omitempty"` // Required if 2FA is enabled for the user
}

type TokenResponse struct {
	Access             string `json:"access"`
	Refresh            string `json:"refresh"`
	MustChangePassword bool   `json:"mustChangePassword"`
	Requires2FA        bool   `json:"requires2FA,omitempty"` // True if 2FA is required but no code provided
}

// Login authenticates a user and returns JWT tokens
func (s *AuthService) Login(ctx context.Context, req LoginRequest, ipAddress, userAgent string) (*TokenResponse, error) {
	email := strings.TrimSpace(req.Email)
	if email == "" {
		return nil, errors.New("invalid credentials")
	}

	// Find user by email
	var user models.User
	findOpts := options.FindOne().SetCollation(caseInsensitiveEmailCollation)
	err := s.db.Collection("users").FindOne(ctx, bson.M{"email": email}, findOpts).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("invalid credentials")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if !user.IsActive {
		return nil, errors.New("user account is disabled")
	}

	// Verify password
	valid, err := utils.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("password verification error: %w", err)
	}
	if !valid {
		return nil, errors.New("invalid credentials")
	}

	// Check if 2FA is enabled for this user
	if user.TOTPSecret != "" {
		// 2FA is enabled - require TOTP code
		if req.TOTPCode == "" {
			// Return indicator that 2FA is required
			return &TokenResponse{Requires2FA: true}, nil
		}

		// Decrypt the TOTP secret
		decryptedSecret := user.TOTPSecret
		if s.cfg.Auth.TOTPEncryptionKey != "" {
			decrypted, err := utils.DecryptTOTPSecret(user.TOTPSecret, s.cfg.Auth.TOTPEncryptionKey)
			if err != nil {
				// If decryption fails, assume it's an old unencrypted secret
				decryptedSecret = user.TOTPSecret
			} else {
				decryptedSecret = decrypted
			}
		}

		// Validate TOTP code
		if !utils.ValidateTOTP(req.TOTPCode, decryptedSecret) {
			return nil, errors.New("invalid 2FA code")
		}
	}

	// Generate tokens
	accessToken, err := utils.GenerateAccessToken(
		user.ID,
		user.Email,
		user.Role,
		s.cfg.JWT.Secret,
		s.cfg.JWT.AccessTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := utils.GenerateRefreshToken(
		user.ID,
		s.cfg.JWT.RefreshSecret,
		s.cfg.JWT.RefreshTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session record (best effort - don't fail login if session creation fails)
	if s.sessionService != nil {
		expiresAt := time.Now().Add(s.cfg.JWT.RefreshTTL)
		_ = s.sessionService.CreateSession(ctx, user.ID, refreshToken, "Web Browser", ipAddress, userAgent, expiresAt)
	}

	return &TokenResponse{
		Access:             accessToken,
		Refresh:            refreshToken,
		MustChangePassword: user.MustChangePassword,
	}, nil
}

// RefreshTokens generates new tokens from a valid refresh token
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string, ipAddress, userAgent string) (*TokenResponse, error) {
	// Validate refresh token
	userID, err := utils.ValidateRefreshToken(refreshToken, s.cfg.JWT.RefreshSecret)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Validate session exists (if session service is available)
	// Note: This is optional for backward compatibility - if session doesn't exist, we'll create one
	var sessionExists bool
	if s.sessionService != nil {
		_, err := s.sessionService.ValidateSession(ctx, refreshToken)
		sessionExists = (err == nil)
	}

	// Find user
	var user models.User
	err = s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if !user.IsActive {
		return nil, errors.New("user account is disabled")
	}

	// Generate new tokens
	accessToken, err := utils.GenerateAccessToken(
		user.ID,
		user.Email,
		user.Role,
		s.cfg.JWT.Secret,
		s.cfg.JWT.AccessTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := utils.GenerateRefreshToken(
		user.ID,
		s.cfg.JWT.RefreshSecret,
		s.cfg.JWT.RefreshTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Update the session with new refresh token
	if s.sessionService != nil {
		if sessionExists {
			// Revoke old session only if it existed
			_ = s.sessionService.RevokeSession(ctx, refreshToken)
		}

		// Create new session with new refresh token
		expiresAt := time.Now().Add(s.cfg.JWT.RefreshTTL)
		_ = s.sessionService.CreateSession(ctx, user.ID, newRefreshToken, "Web Browser", ipAddress, userAgent, expiresAt)
	}

	return &TokenResponse{
		Access:  accessToken,
		Refresh: newRefreshToken,
	}, nil
}

// Enable2FA generates a new TOTP secret for the user
func (s *AuthService) Enable2FA(ctx context.Context, userID primitive.ObjectID) (string, string, error) {
	// Get user
	var user models.User
	err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return "", "", fmt.Errorf("user not found: %w", err)
	}

	// Generate TOTP secret
	secret, err := utils.GenerateTOTPSecret()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Encrypt the secret if encryption key is configured
	secretToStore := secret
	if s.cfg.Auth.TOTPEncryptionKey != "" {
		encrypted, err := utils.EncryptTOTPSecret(secret, s.cfg.Auth.TOTPEncryptionKey)
		if err != nil {
			return "", "", fmt.Errorf("failed to encrypt TOTP secret: %w", err)
		}
		secretToStore = encrypted
	}

	// Update user with encrypted TOTP secret
	_, err = s.db.Collection("users").UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"totp_secret": secretToStore}},
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to save TOTP secret: %w", err)
	}

	// Generate provisioning URL (uses plaintext secret for QR code)
	otpauthURL := utils.GenerateTOTPURL(secret, user.Email, s.cfg.App.Name)

	return secret, otpauthURL, nil
}

// Disable2FA removes the TOTP secret for the user
func (s *AuthService) Disable2FA(ctx context.Context, userID primitive.ObjectID) error {
	_, err := s.db.Collection("users").UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$unset": bson.M{"totp_secret": ""}},
	)
	if err != nil {
		return fmt.Errorf("failed to disable 2FA: %w", err)
	}
	return nil
}

// BeginPasskeyRegistration starts the passkey registration process
func (s *AuthService) BeginPasskeyRegistration(ctx context.Context, userID primitive.ObjectID) (*protocol.CredentialCreation, error) {
	if s.webAuthn == nil {
		return nil, errors.New("WebAuthn not initialized")
	}

	// Get user
	var user models.User
	err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Wrap user for WebAuthn
	webAuthnUser := utils.WebAuthnUser{User: &user}

	// Begin registration
	options, session, err := s.webAuthn.BeginRegistration(webAuthnUser)
	if err != nil {
		return nil, fmt.Errorf("failed to begin registration: %w", err)
	}

	// Store session (in production, use Redis or similar)
	s.sessionsMu.Lock()
	s.sessions[userID.Hex()] = session
	s.sessionsMu.Unlock()

	return options, nil
}

// FinishPasskeyRegistration completes the passkey registration process
func (s *AuthService) FinishPasskeyRegistration(ctx context.Context, userID primitive.ObjectID, credentialName string, response []byte) error {
	if s.webAuthn == nil {
		return errors.New("WebAuthn not initialized")
	}

	// Get session
	s.sessionsMu.Lock()
	session, exists := s.sessions[userID.Hex()]
	if !exists {
		s.sessionsMu.Unlock()
		return errors.New("session not found")
	}
	delete(s.sessions, userID.Hex())
	s.sessionsMu.Unlock()

	// Get user
	var user models.User
	err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Wrap user for WebAuthn
	webAuthnUser := utils.WebAuthnUser{User: &user}

	// Parse credential creation response
	parsedResponse, err := utils.ParseCredentialCreationResponse(response)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Finish registration
	credential, err := s.webAuthn.CreateCredential(webAuthnUser, *session, parsedResponse)
	if err != nil {
		return fmt.Errorf("failed to create credential: %w", err)
	}

	// Convert to our model
	passkeyCredential := utils.ConvertWebAuthnCredential(credential, credentialName)

	// Add credential to user
	_, err = s.db.Collection("users").UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$push": bson.M{"passkey_credentials": passkeyCredential}},
	)
	if err != nil {
		return fmt.Errorf("failed to save credential: %w", err)
	}

	return nil
}

// BeginPasskeyLogin starts the passkey authentication process
func (s *AuthService) BeginPasskeyLogin(ctx context.Context, email string) (*protocol.CredentialAssertion, error) {
	if s.webAuthn == nil {
		return nil, errors.New("WebAuthn not initialized")
	}

	email = strings.TrimSpace(email)
	if email == "" {
		return nil, errors.New("user not found")
	}

	// Find user by email
	var user models.User
	findOpts := options.FindOne().SetCollation(caseInsensitiveEmailCollation)
	err := s.db.Collection("users").FindOne(ctx, bson.M{"email": email}, findOpts).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if !user.IsActive {
		return nil, errors.New("user account is disabled")
	}

	if len(user.PasskeyCredentials) == 0 {
		return nil, errors.New("no passkeys registered")
	}

	// Wrap user for WebAuthn
	webAuthnUser := utils.WebAuthnUser{User: &user}

	// Begin login
	options, session, err := s.webAuthn.BeginLogin(webAuthnUser)
	if err != nil {
		return nil, fmt.Errorf("failed to begin login: %w", err)
	}

	// Store session
	s.sessionsMu.Lock()
	s.sessions[user.ID.Hex()] = session
	s.sessionsMu.Unlock()

	return options, nil
}

// BeginPasskeyDiscoverableLogin starts discoverable credential authentication (no email required)
func (s *AuthService) BeginPasskeyDiscoverableLogin(ctx context.Context) (*protocol.CredentialAssertion, error) {
	if s.webAuthn == nil {
		return nil, errors.New("WebAuthn not initialized")
	}

	// Begin discoverable login (no user specified)
	options, session, err := s.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin discoverable login: %w", err)
	}

	// Store session with a temporary key (we'll update it when we know the user)
	sessionKey := fmt.Sprintf("discoverable_%d", time.Now().UnixNano())
	s.sessionsMu.Lock()
	s.sessions[sessionKey] = session
	s.sessionsMu.Unlock()

	// Store the session key in the response (we'll need it to retrieve the session)
	return options, nil
}

// FinishPasskeyLogin completes the passkey authentication process
func (s *AuthService) FinishPasskeyLogin(ctx context.Context, email string, response []byte, ipAddress, userAgent string) (*TokenResponse, error) {
	if s.webAuthn == nil {
		return nil, errors.New("WebAuthn not initialized")
	}

	email = strings.TrimSpace(email)
	if email == "" {
		return nil, errors.New("invalid credentials")
	}

	// Find user by email
	var user models.User
	findOpts := options.FindOne().SetCollation(caseInsensitiveEmailCollation)
	err := s.db.Collection("users").FindOne(ctx, bson.M{"email": email}, findOpts).Decode(&user)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Get session
	s.sessionsMu.Lock()
	session, exists := s.sessions[user.ID.Hex()]
	if !exists {
		s.sessionsMu.Unlock()
		return nil, errors.New("session not found")
	}
	delete(s.sessions, user.ID.Hex())
	s.sessionsMu.Unlock()

	// Wrap user for WebAuthn
	webAuthnUser := utils.WebAuthnUser{User: &user}

	// Parse credential assertion response
	parsedResponse, err := utils.ParseCredentialRequestResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Validate login
	credential, err := s.webAuthn.ValidateLogin(webAuthnUser, *session, parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Update credential sign count and last used time
	now := time.Now()
	_, err = s.db.Collection("users").UpdateOne(
		ctx,
		bson.M{
			"_id":                    user.ID,
			"passkey_credentials.id": credential.ID,
		},
		bson.M{"$set": bson.M{
			"passkey_credentials.$.sign_count":   credential.Authenticator.SignCount,
			"passkey_credentials.$.last_used_at": now,
		}},
	)
	if err != nil {
		// Log but don't fail - authentication was successful
		fmt.Printf("Warning: Failed to update credential: %v\n", err)
	}

	// Generate tokens
	accessToken, err := utils.GenerateAccessToken(
		user.ID,
		user.Email,
		user.Role,
		s.cfg.JWT.Secret,
		s.cfg.JWT.AccessTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := utils.GenerateRefreshToken(
		user.ID,
		s.cfg.JWT.RefreshSecret,
		s.cfg.JWT.RefreshTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session record (best effort - don't fail login if session creation fails)
	if s.sessionService != nil {
		expiresAt := time.Now().Add(s.cfg.JWT.RefreshTTL)
		_ = s.sessionService.CreateSession(ctx, user.ID, refreshToken, "Passkey Login", ipAddress, userAgent, expiresAt)
	}

	return &TokenResponse{
		Access:             accessToken,
		Refresh:            refreshToken,
		MustChangePassword: user.MustChangePassword,
	}, nil
}

// FinishPasskeyDiscoverableLogin completes discoverable credential authentication
func (s *AuthService) FinishPasskeyDiscoverableLogin(ctx context.Context, response []byte, ipAddress, userAgent string) (*TokenResponse, error) {
	if s.webAuthn == nil {
		return nil, errors.New("WebAuthn not initialized")
	}

	// Parse credential assertion response
	parsedResponse, err := utils.ParseCredentialRequestResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Find the matching session (try all discoverable sessions)
	var session *webauthn.SessionData
	var sessionKey string
	s.sessionsMu.Lock()
	for key, sess := range s.sessions {
		if len(key) > 13 && key[:13] == "discoverable_" {
			session = sess
			sessionKey = key
			break
		}
	}

	if session == nil {
		s.sessionsMu.Unlock()
		return nil, errors.New("session not found")
	}
	delete(s.sessions, sessionKey)
	s.sessionsMu.Unlock()

	// Create user handler for discoverable login
	userHandler := func(rawID, userHandle []byte) (webauthn.User, error) {
		// Convert user handle to ObjectID
		userIDHex := string(userHandle)
		userID, err := primitive.ObjectIDFromHex(userIDHex)
		if err != nil {
			return nil, errors.New("invalid user handle")
		}

		// Find user by ID
		var user models.User
		err = s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
		if err != nil {
			return nil, errors.New("user not found")
		}

		if !user.IsActive {
			return nil, errors.New("user account is disabled")
		}

		return utils.WebAuthnUser{User: &user}, nil
	}

	// Validate discoverable login
	credential, err := s.webAuthn.ValidateDiscoverableLogin(userHandler, *session, parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Get the user from the user handle for token generation
	userIDHex := string(parsedResponse.Response.UserHandle)
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return nil, fmt.Errorf("invalid user handle: %w", err)
	}
	var user models.User
	if err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Update credential sign count and last used time
	now := time.Now()
	_, err = s.db.Collection("users").UpdateOne(
		ctx,
		bson.M{
			"_id":                    user.ID,
			"passkey_credentials.id": credential.ID,
		},
		bson.M{"$set": bson.M{
			"passkey_credentials.$.sign_count":   credential.Authenticator.SignCount,
			"passkey_credentials.$.last_used_at": now,
		}},
	)
	if err != nil {
		// Log but don't fail - authentication was successful
		fmt.Printf("Warning: Failed to update credential: %v\n", err)
	}

	// Generate tokens
	accessToken, err := utils.GenerateAccessToken(
		user.ID,
		user.Email,
		user.Role,
		s.cfg.JWT.Secret,
		s.cfg.JWT.AccessTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := utils.GenerateRefreshToken(
		user.ID,
		s.cfg.JWT.RefreshSecret,
		s.cfg.JWT.RefreshTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session record (best effort - don't fail login if session creation fails)
	if s.sessionService != nil {
		expiresAt := time.Now().Add(s.cfg.JWT.RefreshTTL)
		_ = s.sessionService.CreateSession(ctx, user.ID, refreshToken, "Passkey Login (Discoverable)", ipAddress, userAgent, expiresAt)
	}

	return &TokenResponse{
		Access:             accessToken,
		Refresh:            refreshToken,
		MustChangePassword: user.MustChangePassword,
	}, nil
}

// ListPasskeys returns all passkeys for a user
func (s *AuthService) ListPasskeys(ctx context.Context, userID primitive.ObjectID) ([]models.PasskeyCredential, error) {
	var user models.User
	err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user.PasskeyCredentials, nil
}

// DeletePasskey removes a passkey from a user
func (s *AuthService) DeletePasskey(ctx context.Context, userID primitive.ObjectID, credentialID []byte) error {
	_, err := s.db.Collection("users").UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$pull": bson.M{"passkey_credentials": bson.M{"id": credentialID}}},
	)
	if err != nil {
		return fmt.Errorf("failed to delete passkey: %w", err)
	}

	return nil
}

// Logout revokes the session associated with the refresh token
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if s.sessionService != nil {
		return s.sessionService.RevokeSession(ctx, refreshToken)
	}
	return nil
}

// BootstrapAdmin creates the admin user if it doesn't exist
func (s *AuthService) BootstrapAdmin(ctx context.Context) error {
	// Check if admin already exists
	count, err := s.db.Collection("users").CountDocuments(ctx, bson.M{"role": "ADMIN"})
	if err != nil {
		return fmt.Errorf("failed to check for existing admin: %w", err)
	}

	if count > 0 {
		// Admin already exists
		return nil
	}

	// Hash password if it's plain text (for development)
	passwordHash := s.cfg.Admin.PasswordHash
	if passwordHash != "" && len(passwordHash) < 50 {
		// Looks like plain text password, hash it
		hashed, err := utils.HashPassword(passwordHash)
		if err != nil {
			return fmt.Errorf("failed to hash admin password: %w", err)
		}
		passwordHash = hashed
	}

	adminEmail := strings.ToLower(strings.TrimSpace(s.cfg.Admin.Email))

	// Create admin user from config
	admin := models.User{
		ID:           primitive.NewObjectID(),
		Email:        adminEmail,
		PasswordHash: passwordHash,
		Role:         "ADMIN",
		IsActive:     true,
		CreatedAt:    time.Now(),
	}

	_, err = s.db.Collection("users").InsertOne(ctx, admin)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	return nil
}
