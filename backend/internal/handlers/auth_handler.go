package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/sainaif/holy-home/internal/middleware"
	"github.com/sainaif/holy-home/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
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

	tokens, err := h.authService.Login(c.Context(), req)
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
		RefreshToken string `json:"refreshToken"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	tokens, err := h.authService.RefreshTokens(c.Context(), req.RefreshToken)
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
		tokens, err := h.authService.FinishPasskeyDiscoverableLogin(c.Context(), credBytes)
		if err != nil {
			fmt.Printf("[Passkey] Discoverable login failed: %v\n", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(tokens)
	}

	tokens, err := h.authService.FinishPasskeyLogin(c.Context(), req.Email, credBytes)
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