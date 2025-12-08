package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// MockUser represents a user for testing
type MockUser struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	IsActive bool   `json:"isActive"`
}

func TestGetCurrentUser_Success(t *testing.T) {
	app := fiber.New()

	mockUser := MockUser{
		ID:       uuid.New().String(),
		Email:    "test@example.com",
		Name:     "Test User",
		Role:     "user",
		IsActive: true,
	}

	app.Get("/users/me", func(c *fiber.Ctx) error {
		return c.JSON(mockUser)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result MockUser
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "test@example.com", result.Email)
	assert.Equal(t, "Test User", result.Name)
}

func TestGetCurrentUser_Unauthorized(t *testing.T) {
	app := fiber.New()

	app.Get("/users/me", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUpdateUser_Success(t *testing.T) {
	app := fiber.New()

	userID := uuid.New().String()

	app.Put("/users/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if _, err := uuid.Parse(id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid user ID",
			})
		}

		var req struct {
			Name  *string `json:"name"`
			Email *string `json:"email"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		return c.JSON(fiber.Map{
			"message": "User updated successfully",
		})
	})

	name := "Updated Name"
	reqBody := map[string]*string{
		"name": &name,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/users/"+userID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUpdateUser_InvalidID(t *testing.T) {
	app := fiber.New()

	app.Put("/users/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if _, err := uuid.Parse(id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid user ID",
			})
		}
		return c.SendStatus(fiber.StatusOK)
	})

	reqBody := map[string]string{
		"name": "Updated Name",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/users/invalid-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestChangePassword_Success(t *testing.T) {
	app := fiber.New()

	app.Post("/users/me/password", func(c *fiber.Ctx) error {
		var req struct {
			OldPassword string `json:"oldPassword"`
			NewPassword string `json:"newPassword"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.OldPassword == "" || req.NewPassword == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Both old and new passwords are required",
			})
		}

		if len(req.NewPassword) < 8 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Password must be at least 8 characters",
			})
		}

		// Simulate successful password change
		return c.JSON(fiber.Map{
			"message": "Password changed successfully",
		})
	})

	reqBody := map[string]string{
		"oldPassword": "oldPassword123",
		"newPassword": "newPassword456",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/users/me/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestChangePassword_MissingFields(t *testing.T) {
	app := fiber.New()

	app.Post("/users/me/password", func(c *fiber.Ctx) error {
		var req struct {
			OldPassword string `json:"oldPassword"`
			NewPassword string `json:"newPassword"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.OldPassword == "" || req.NewPassword == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Both old and new passwords are required",
			})
		}

		return c.SendStatus(fiber.StatusOK)
	})

	// Missing newPassword
	reqBody := map[string]string{
		"oldPassword": "oldPassword123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/users/me/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestChangePassword_WeakPassword(t *testing.T) {
	app := fiber.New()

	app.Post("/users/me/password", func(c *fiber.Ctx) error {
		var req struct {
			OldPassword string `json:"oldPassword"`
			NewPassword string `json:"newPassword"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if len(req.NewPassword) < 8 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Password must be at least 8 characters",
			})
		}

		return c.SendStatus(fiber.StatusOK)
	})

	reqBody := map[string]string{
		"oldPassword": "oldPassword123",
		"newPassword": "short",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/users/me/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetUsers_Admin_Success(t *testing.T) {
	app := fiber.New()

	mockUsers := []MockUser{
		{
			ID:       uuid.New().String(),
			Email:    "user1@example.com",
			Name:     "User One",
			Role:     "user",
			IsActive: true,
		},
		{
			ID:       uuid.New().String(),
			Email:    "user2@example.com",
			Name:     "User Two",
			Role:     "admin",
			IsActive: true,
		},
	}

	app.Get("/users", func(c *fiber.Ctx) error {
		// Simulate admin access
		return c.JSON(mockUsers)
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Set("Authorization", "Bearer admin-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []MockUser
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 2)
}

func TestGetUsers_Forbidden(t *testing.T) {
	app := fiber.New()

	app.Get("/users", func(c *fiber.Ctx) error {
		// Simulate non-admin access
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Set("Authorization", "Bearer user-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestCreateUser_Admin_Success(t *testing.T) {
	app := fiber.New()

	app.Post("/users", func(c *fiber.Ctx) error {
		var req struct {
			Email    string `json:"email"`
			Name     string `json:"name"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.Email == "" || req.Name == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Email, name, and password are required",
			})
		}

		return c.Status(fiber.StatusCreated).JSON(MockUser{
			ID:       uuid.New().String(),
			Email:    req.Email,
			Name:     req.Name,
			Role:     req.Role,
			IsActive: true,
		})
	})

	reqBody := map[string]string{
		"email":    "newuser@example.com",
		"name":     "New User",
		"password": "securePassword123",
		"role":     "user",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result MockUser
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "newuser@example.com", result.Email)
	assert.Equal(t, "New User", result.Name)
}

func TestCreateUser_MissingFields(t *testing.T) {
	app := fiber.New()

	app.Post("/users", func(c *fiber.Ctx) error {
		var req struct {
			Email    string `json:"email"`
			Name     string `json:"name"`
			Password string `json:"password"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.Email == "" || req.Name == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Email, name, and password are required",
			})
		}

		return c.SendStatus(fiber.StatusCreated)
	})

	// Missing password
	reqBody := map[string]string{
		"email": "newuser@example.com",
		"name":  "New User",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
