package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/config"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
	"github.com/sainaif/holy-home/internal/utils"
)

// webAuthnSession wraps session data with expiry time for TTL-based cleanup
type webAuthnSession struct {
	data      *webauthn.SessionData
	expiresAt time.Time
}

const webAuthnSessionTTL = 5 * time.Minute // Sessions expire after 5 minutes

// emailRegex validates email format (RFC 5322 simplified)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// isValidEmail checks if the email format is valid
func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

type AuthService struct {
	users          repository.UserRepository
	passkeys       repository.PasskeyCredentialRepository
	roles          repository.RoleRepository
	cfg            *config.Config
	webAuthn       *webauthn.WebAuthn
	sessions       map[string]*webAuthnSession // In-memory session storage with TTL
	sessionsMu     sync.RWMutex                // Mutex to protect concurrent access to sessions map
	sessionService *SessionService
	stopCleanup    chan struct{} // Signal to stop cleanup goroutine
}

func NewAuthService(
	users repository.UserRepository,
	passkeys repository.PasskeyCredentialRepository,
	roles repository.RoleRepository,
	cfg *config.Config,
	sessionService *SessionService,
) *AuthService {
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

	service := &AuthService{
		users:          users,
		passkeys:       passkeys,
		roles:          roles,
		cfg:            cfg,
		webAuthn:       wa,
		sessions:       make(map[string]*webAuthnSession),
		sessionService: sessionService,
		stopCleanup:    make(chan struct{}),
	}

	// Start session cleanup goroutine
	go service.cleanupExpiredSessions()

	return service
}

// cleanupExpiredSessions periodically removes expired WebAuthn sessions
func (s *AuthService) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.sessionsMu.Lock()
			now := time.Now()
			for key, session := range s.sessions {
				if now.After(session.expiresAt) {
					delete(s.sessions, key)
				}
			}
			s.sessionsMu.Unlock()
		case <-s.stopCleanup:
			return
		}
	}
}

// Close stops the cleanup goroutine (call on shutdown)
func (s *AuthService) Close() {
	close(s.stopCleanup)
}

type LoginRequest struct {
	Email      string `json:"email"`              // Email for login (if email login is enabled)
	Username   string `json:"username,omitempty"` // Username for login (if username login is enabled)
	Identifier string `json:"identifier"`         // Generic identifier - can be email or username
	Password   string `json:"password"`
	TOTPCode   string `json:"totpCode,omitempty"` // Required if 2FA is enabled for the user
}

type TokenResponse struct {
	Access             string `json:"access"`
	Refresh            string `json:"refresh"`
	MustChangePassword bool   `json:"mustChangePassword"`
	Requires2FA        bool   `json:"requires2FA,omitempty"` // True if 2FA is required but no code provided
}

// Login authenticates a user and returns JWT tokens
func (s *AuthService) Login(ctx context.Context, req LoginRequest, ipAddress, userAgent string) (*TokenResponse, error) {
	// Determine the identifier to use for login
	// Priority: identifier > email > username
	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		identifier = strings.TrimSpace(req.Email)
	}
	if identifier == "" {
		identifier = strings.TrimSpace(req.Username)
	}
	if identifier == "" {
		return nil, errors.New("invalid credentials")
	}

	// Find user by email or username based on config
	var user *models.User
	var err error

	// Check if identifier looks like an email
	isEmail := isValidEmail(identifier)

	if isEmail && s.cfg.Auth.AllowEmailLogin {
		// Try to find by email
		user, err = s.users.GetByEmail(ctx, identifier)
	} else if !isEmail && s.cfg.Auth.AllowUsernameLogin {
		// Try to find by username (case-insensitive)
		user, err = s.users.GetByUsername(ctx, identifier)
	} else if isEmail && !s.cfg.Auth.AllowEmailLogin {
		return nil, errors.New("email login is disabled")
	} else if !isEmail && !s.cfg.Auth.AllowUsernameLogin {
		return nil, errors.New("username login is disabled")
	} else {
		return nil, errors.New("invalid credentials")
	}

	if err != nil {
		return nil, errors.New("invalid credentials")
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
				// Decryption failed - this could be a legacy unencrypted secret
				fmt.Printf("Warning: TOTP decryption failed for user %s (may be legacy unencrypted secret): %v\n", user.Email, err)
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
	var sessionExists bool
	if s.sessionService != nil {
		_, err := s.sessionService.ValidateSession(ctx, refreshToken)
		sessionExists = (err == nil)
	}

	// Find user
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
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
func (s *AuthService) Enable2FA(ctx context.Context, userID string) (string, string, error) {
	// Get user
	user, err := s.users.GetByID(ctx, userID)
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
	if err := s.users.UpdateTOTPSecret(ctx, userID, secretToStore); err != nil {
		return "", "", fmt.Errorf("failed to save TOTP secret: %w", err)
	}

	// Generate provisioning URL (uses plaintext secret for QR code)
	otpauthURL := utils.GenerateTOTPURL(secret, user.Email, s.cfg.App.Name)

	return secret, otpauthURL, nil
}

// Disable2FA removes the TOTP secret for the user
func (s *AuthService) Disable2FA(ctx context.Context, userID string) error {
	if err := s.users.UpdateTOTPSecret(ctx, userID, ""); err != nil {
		return fmt.Errorf("failed to disable 2FA: %w", err)
	}
	return nil
}

// BeginPasskeyRegistration starts the passkey registration process
func (s *AuthService) BeginPasskeyRegistration(ctx context.Context, userID string) (*protocol.CredentialCreation, error) {
	if s.webAuthn == nil {
		return nil, errors.New("WebAuthn not initialized")
	}

	// Get user
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Load passkey credentials
	creds, err := s.passkeys.GetByUserID(ctx, userID)
	if err != nil {
		creds = []models.PasskeyCredential{}
	}
	user.PasskeyCredentials = creds

	// Wrap user for WebAuthn
	webAuthnUser := utils.WebAuthnUser{User: user}

	// Begin registration
	options, session, err := s.webAuthn.BeginRegistration(webAuthnUser)
	if err != nil {
		return nil, fmt.Errorf("failed to begin registration: %w", err)
	}

	// Store session with TTL
	s.sessionsMu.Lock()
	s.sessions[userID] = &webAuthnSession{
		data:      session,
		expiresAt: time.Now().Add(webAuthnSessionTTL),
	}
	s.sessionsMu.Unlock()

	return options, nil
}

// FinishPasskeyRegistration completes the passkey registration process
func (s *AuthService) FinishPasskeyRegistration(ctx context.Context, userID string, credentialName string, response []byte) error {
	if s.webAuthn == nil {
		return errors.New("WebAuthn not initialized")
	}

	// Get session
	s.sessionsMu.Lock()
	sessionWrapper, exists := s.sessions[userID]
	if !exists || time.Now().After(sessionWrapper.expiresAt) {
		if exists {
			delete(s.sessions, userID)
		}
		s.sessionsMu.Unlock()
		return errors.New("session not found or expired")
	}
	session := sessionWrapper.data
	delete(s.sessions, userID)
	s.sessionsMu.Unlock()

	// Get user
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Load passkey credentials
	creds, err := s.passkeys.GetByUserID(ctx, userID)
	if err != nil {
		creds = []models.PasskeyCredential{}
	}
	user.PasskeyCredentials = creds

	// Wrap user for WebAuthn
	webAuthnUser := utils.WebAuthnUser{User: user}

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

	// Add credential to passkeys table
	if err := s.passkeys.Create(ctx, userID, &passkeyCredential); err != nil {
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
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if !user.IsActive {
		return nil, errors.New("user account is disabled")
	}

	// Load passkey credentials
	creds, err := s.passkeys.GetByUserID(ctx, user.ID)
	if err != nil || len(creds) == 0 {
		return nil, errors.New("invalid credentials")
	}
	user.PasskeyCredentials = creds

	// Wrap user for WebAuthn
	webAuthnUser := utils.WebAuthnUser{User: user}

	// Begin login
	options, session, err := s.webAuthn.BeginLogin(webAuthnUser)
	if err != nil {
		return nil, fmt.Errorf("failed to begin login: %w", err)
	}

	// Store session with TTL
	s.sessionsMu.Lock()
	s.sessions[user.ID] = &webAuthnSession{
		data:      session,
		expiresAt: time.Now().Add(webAuthnSessionTTL),
	}
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

	// Store session with a temporary key and TTL
	sessionKey := fmt.Sprintf("discoverable_%d", time.Now().UnixNano())
	s.sessionsMu.Lock()
	s.sessions[sessionKey] = &webAuthnSession{
		data:      session,
		expiresAt: time.Now().Add(webAuthnSessionTTL),
	}
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
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Get session
	s.sessionsMu.Lock()
	sessionWrapper, exists := s.sessions[user.ID]
	if !exists || time.Now().After(sessionWrapper.expiresAt) {
		if exists {
			delete(s.sessions, user.ID)
		}
		s.sessionsMu.Unlock()
		return nil, errors.New("session not found or expired")
	}
	session := sessionWrapper.data
	delete(s.sessions, user.ID)
	s.sessionsMu.Unlock()

	// Load passkey credentials
	creds, err := s.passkeys.GetByUserID(ctx, user.ID)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	user.PasskeyCredentials = creds

	// Wrap user for WebAuthn
	webAuthnUser := utils.WebAuthnUser{User: user}

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
	if err := s.passkeys.UpdateSignCount(ctx, credential.ID, credential.Authenticator.SignCount); err != nil {
		fmt.Printf("Warning: Failed to update sign count: %v\n", err)
	}
	if err := s.passkeys.UpdateLastUsed(ctx, credential.ID, now); err != nil {
		fmt.Printf("Warning: Failed to update last used: %v\n", err)
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
			session = sess.data
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
		// User handle is the user ID
		userID := string(userHandle)

		// Find user by ID
		user, err := s.users.GetByID(ctx, userID)
		if err != nil {
			return nil, errors.New("user not found")
		}

		if !user.IsActive {
			return nil, errors.New("user account is disabled")
		}

		// Load passkey credentials
		creds, err := s.passkeys.GetByUserID(ctx, userID)
		if err != nil {
			return nil, errors.New("failed to load credentials")
		}
		user.PasskeyCredentials = creds

		return utils.WebAuthnUser{User: user}, nil
	}

	// Validate discoverable login
	credential, err := s.webAuthn.ValidateDiscoverableLogin(userHandler, *session, parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Get the user from the user handle for token generation
	userID := string(parsedResponse.Response.UserHandle)
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Update credential sign count and last used time
	now := time.Now()
	if err := s.passkeys.UpdateSignCount(ctx, credential.ID, credential.Authenticator.SignCount); err != nil {
		fmt.Printf("Warning: Failed to update sign count: %v\n", err)
	}
	if err := s.passkeys.UpdateLastUsed(ctx, credential.ID, now); err != nil {
		fmt.Printf("Warning: Failed to update last used: %v\n", err)
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
func (s *AuthService) ListPasskeys(ctx context.Context, userID string) ([]models.PasskeyCredential, error) {
	return s.passkeys.GetByUserID(ctx, userID)
}

// DeletePasskey removes a passkey from a user
func (s *AuthService) DeletePasskey(ctx context.Context, userID string, credentialID []byte) error {
	return s.passkeys.Delete(ctx, userID, credentialID)
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
	// Check if any admin already exists
	users, err := s.users.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for existing admin: %w", err)
	}

	for _, u := range users {
		if u.Role == "ADMIN" {
			// Admin already exists
			return nil
		}
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
		ID:           uuid.New().String(),
		Email:        adminEmail,
		PasswordHash: passwordHash,
		Role:         "ADMIN",
		IsActive:     true,
		CreatedAt:    time.Now(),
	}

	if err := s.users.Create(ctx, &admin); err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	return nil
}
