package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/config"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
)

type AuthHandler struct {
	authService  *services.AuthService
	userService  *services.UserService
	auditService *services.AuditService
	cfg          *config.Config
}

func NewAuthHandler(authService *services.AuthService, userService *services.UserService, auditService *services.AuditService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		userService:  userService,
		auditService: auditService,
		cfg:          cfg,
	}
}

// AuthConfigResponse contains public auth configuration for the frontend
type AuthConfigResponse struct {
	AllowEmailLogin    bool `json:"allowEmailLogin"`
	AllowUsernameLogin bool `json:"allowUsernameLogin"`
	RequireUsername    bool `json:"requireUsername"`
	TwoFAEnabled       bool `json:"twoFAEnabled"`
}

// GetAuthConfig returns public auth configuration
// @Summary Get auth configuration
// @Description Get public authentication configuration for the frontend
// @Tags auth
// @Produce json
// @Success 200 {object} AuthConfigResponse
// @Router /auth/config [get]
func (h *AuthHandler) GetAuthConfig(c *fiber.Ctx) error {
	return c.JSON(AuthConfigResponse{
		AllowEmailLogin:    h.cfg.Auth.AllowEmailLogin,
		AllowUsernameLogin: h.cfg.Auth.AllowUsernameLogin,
		RequireUsername:    h.cfg.Auth.RequireUsername,
		TwoFAEnabled:       h.cfg.Auth.TwoFAEnabled,
	})
}

// Login godoc
// @Summary Login
// @Description Authenticate user and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body services.LoginRequest true "Login credentials"
// @Success 200 {object} services.TokenResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req services.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	tokens, err := h.authService.Login(c.Context(), req, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(tokens)
}

// Refresh godoc
// @Summary Refresh tokens
// @Description Generate new tokens from refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body map[string]string true "Refresh token"
// @Success 200 {object} services.TokenResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req struct {
		RefreshToken       string `json:"refreshToken"`
		LegacyRefreshToken string `json:"refresh_token"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		refreshToken = req.LegacyRefreshToken
	}

	if refreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Refresh token is required",
		})
	}

	tokens, err := h.authService.RefreshTokens(c.Context(), refreshToken, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(tokens)
}

// Enable2FA godoc
// @Summary Enable 2FA
// @Description Generate TOTP secret for 2FA
// @Tags auth
// @Produce json
// @Security Bearer
// @Success 200 {object} map[string]string
// @Router /auth/enable-2fa [post]
func (h *AuthHandler) Enable2FA(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	secret, otpauthURL, err := h.authService.Enable2FA(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"secret":      secret,
		"otpauth_url": otpauthURL,
	})
}

// Disable2FA godoc
// @Summary Disable 2FA
// @Description Remove TOTP secret
// @Tags auth
// @Produce json
// @Security Bearer
// @Success 200 {object} map[string]string
// @Router /auth/disable-2fa [post]
func (h *AuthHandler) Disable2FA(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if err := h.authService.Disable2FA(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "2FA disabled successfully",
	})
}

// BeginPasskeyRegistration godoc
// @Summary Begin passkey registration
// @Description Start the passkey registration flow for authenticated user
// @Tags auth
// @Produce json
// @Security Bearer
// @Success 200 {object} map[string]interface{}
// @Router /auth/passkey/register/begin [post]
func (h *AuthHandler) BeginPasskeyRegistration(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	options, err := h.authService.BeginPasskeyRegistration(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(options)
}

// FinishPasskeyRegistration godoc
// @Summary Finish passkey registration
// @Description Complete the passkey registration flow
// @Tags auth
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body map[string]interface{} true "Credential and name"
// @Success 200 {object} map[string]string
// @Router /auth/passkey/register/finish [post]
func (h *AuthHandler) FinishPasskeyRegistration(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		Name       string                 `json:"name"`
		Credential map[string]interface{} `json:"credential"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Re-marshal credential to bytes for processing
	credBytes, err := json.Marshal(req.Credential)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid credential format",
		})
	}

	if err := h.authService.FinishPasskeyRegistration(c.Context(), userID, req.Name, credBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Passkey registered successfully",
	})
}

// BeginPasskeyLogin godoc
// @Summary Begin passkey login
// @Description Start the passkey authentication flow
// @Tags auth
// @Accept json
// @Produce json
// @Param request body map[string]string true "Email (optional for discoverable)"
// @Success 200 {object} map[string]interface{}
// @Router /auth/passkey/login/begin [post]
func (h *AuthHandler) BeginPasskeyLogin(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// If email is empty, use discoverable credentials
	if req.Email == "" {
		options, err := h.authService.BeginPasskeyDiscoverableLogin(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(options)
	}

	options, err := h.authService.BeginPasskeyLogin(c.Context(), req.Email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(options)
}

// FinishPasskeyLogin godoc
// @Summary Finish passkey login
// @Description Complete the passkey authentication flow and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "Email (optional) and credential"
// @Success 200 {object} services.TokenResponse
// @Router /auth/passkey/login/finish [post]
func (h *AuthHandler) FinishPasskeyLogin(c *fiber.Ctx) error {
	var req struct {
		Email      string                 `json:"email"`
		Credential map[string]interface{} `json:"credential"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Re-marshal credential to bytes for processing
	credBytes, err := json.Marshal(req.Credential)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid credential format",
		})
	}

	// If email is empty, use discoverable credentials
	if req.Email == "" {
		tokens, err := h.authService.FinishPasskeyDiscoverableLogin(c.Context(), credBytes, c.IP(), c.Get("User-Agent"))
		if err != nil {
			fmt.Printf("[Passkey] Discoverable login failed: %v\n", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(tokens)
	}

	tokens, err := h.authService.FinishPasskeyLogin(c.Context(), req.Email, credBytes, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(tokens)
}

// ListPasskeys godoc
// @Summary List user passkeys
// @Description Get all registered passkeys for authenticated user
// @Tags auth
// @Produce json
// @Security Bearer
// @Success 200 {array} models.PasskeyCredential
// @Router /auth/passkeys [get]
func (h *AuthHandler) ListPasskeys(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	passkeys, err := h.authService.ListPasskeys(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Return sanitized passkeys (without sensitive data)
	type passkeyResponse struct {
		CredentialID string `json:"credentialId"` // Base64URL encoded for deletion
		Name         string `json:"name"`
		CreatedAt    string `json:"createdAt"`
		LastUsedAt   string `json:"lastUsedAt"`
	}

	response := make([]passkeyResponse, len(passkeys))
	for i, pk := range passkeys {
		response[i] = passkeyResponse{
			CredentialID: base64.RawURLEncoding.EncodeToString(pk.ID),
			Name:         pk.Name,
			CreatedAt:    pk.CreatedAt.Format("2006-01-02 15:04:05"),
			LastUsedAt:   pk.LastUsedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return c.JSON(response)
}

// DeletePasskey godoc
// @Summary Delete a passkey
// @Description Remove a passkey from authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body map[string]string true "Credential ID"
// @Success 200 {object} map[string]string
// @Router /auth/passkeys [delete]
func (h *AuthHandler) DeletePasskey(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req struct {
		CredentialID string `json:"credentialId"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Decode base64url credential ID
	credID, err := base64.RawURLEncoding.DecodeString(req.CredentialID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid credential ID format",
		})
	}

	if err := h.authService.DeletePasskey(c.Context(), userID, credID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Passkey deleted successfully",
	})
}

// ValidateResetToken godoc
// @Summary Validate password reset token
// @Description Check if a password reset token is valid and not expired
// @Tags auth
// @Produce json
// @Param token query string true "Reset token"
// @Success 200 {object} map[string]interface{}
// @Router /auth/validate-reset-token [get]
func (h *AuthHandler) ValidateResetToken(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token is required",
		})
	}

	// Validate the token
	resetToken, err := h.userService.ValidateResetToken(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"valid": false,
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"valid":     true,
		"expiresAt": resetToken.ExpiresAt,
		"userId":    resetToken.UserID,
	})
}

// ResetPasswordWithToken godoc
// @Summary Reset password with token
// @Description Reset user password using a valid reset token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body map[string]string true "Token and new password"
// @Success 200 {object} services.TokenResponse
// @Router /auth/reset-password [post]
func (h *AuthHandler) ResetPasswordWithToken(c *fiber.Ctx) error {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate inputs
	if req.Token == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token and new password are required",
		})
	}

	// Validate password strength (minimum 8 characters)
	if len(req.NewPassword) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 8 characters long",
		})
	}

	// Validate the token first to get user ID for audit logging
	resetToken, err := h.userService.ValidateResetToken(c.Context(), req.Token)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Get user info for audit logging
	user, err := h.userService.GetUser(c.Context(), resetToken.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user information",
		})
	}

	// Reset the password and get JWT tokens
	tokens, err := h.userService.ResetPasswordWithToken(c.Context(), req.Token, req.NewPassword)
	if err != nil {
		// Log failed attempt
		h.auditService.LogAction(
			c.Context(),
			resetToken.UserID,
			user.Email,
			user.Name,
			"password_reset.complete",
			"user",
			&resetToken.UserID,
			map[string]interface{}{"error": err.Error()},
			c.IP(),
			c.Get("User-Agent"),
			"failure",
		)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Log successful password reset
	h.auditService.LogAction(
		c.Context(),
		resetToken.UserID,
		user.Email,
		user.Name,
		"password_reset.complete",
		"user",
		&resetToken.UserID,
		map[string]interface{}{"method": "reset_token"},
		c.IP(),
		c.Get("User-Agent"),
		"success",
	)

	return c.JSON(fiber.Map{
		"accessToken":  tokens["accessToken"],
		"refreshToken": tokens["refreshToken"],
		"message":      "Password reset successfully",
	})
}

// Logout godoc
// @Summary Logout
// @Description Revoke the current session's refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body map[string]string true "Refresh token"
// @Success 200 {object} map[string]string
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Refresh token is required",
		})
	}

	// Revoke the session (best effort - don't fail if session doesn't exist)
	if err := h.authService.Logout(c.Context(), req.RefreshToken); err != nil {
		// Log but don't fail - session may already be expired/deleted
		fmt.Printf("Warning: Failed to revoke session during logout: %v\n", err)
	}

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}
