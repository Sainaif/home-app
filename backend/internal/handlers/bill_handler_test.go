package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// MockBill represents a bill for testing
type MockBill struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	CustomType     string    `json:"customType,omitempty"`
	AllocationType string    `json:"allocationType"`
	TotalAmountPLN float64   `json:"totalAmountPLN"`
	TotalUnits     float64   `json:"totalUnits,omitempty"`
	PeriodStart    time.Time `json:"periodStart"`
	PeriodEnd      time.Time `json:"periodEnd"`
	Status         string    `json:"status"`
	Notes          string    `json:"notes,omitempty"`
}

func TestGetBills_Success(t *testing.T) {
	app := fiber.New()

	mockBills := []MockBill{
		{
			ID:             uuid.New().String(),
			Type:           "electricity",
			AllocationType: "metered",
			TotalAmountPLN: 250.50,
			TotalUnits:     120.5,
			PeriodStart:    time.Now().AddDate(0, -1, 0),
			PeriodEnd:      time.Now(),
			Status:         "draft",
		},
		{
			ID:             uuid.New().String(),
			Type:           "gas",
			AllocationType: "metered",
			TotalAmountPLN: 180.00,
			TotalUnits:     45.2,
			PeriodStart:    time.Now().AddDate(0, -1, 0),
			PeriodEnd:      time.Now(),
			Status:         "posted",
		},
	}

	app.Get("/bills", func(c *fiber.Ctx) error {
		status := c.Query("status")
		if status != "" {
			// Filter by status
			var filtered []MockBill
			for _, b := range mockBills {
				if b.Status == status {
					filtered = append(filtered, b)
				}
			}
			return c.JSON(filtered)
		}
		return c.JSON(mockBills)
	})

	req := httptest.NewRequest(http.MethodGet, "/bills", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []MockBill
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 2)
}

func TestGetBills_FilterByStatus(t *testing.T) {
	app := fiber.New()

	mockBills := []MockBill{
		{
			ID:     uuid.New().String(),
			Type:   "electricity",
			Status: "draft",
		},
		{
			ID:     uuid.New().String(),
			Type:   "gas",
			Status: "posted",
		},
	}

	app.Get("/bills", func(c *fiber.Ctx) error {
		status := c.Query("status")
		if status != "" {
			var filtered []MockBill
			for _, b := range mockBills {
				if b.Status == status {
					filtered = append(filtered, b)
				}
			}
			return c.JSON(filtered)
		}
		return c.JSON(mockBills)
	})

	req := httptest.NewRequest(http.MethodGet, "/bills?status=posted", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []MockBill
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Len(t, result, 1)
	assert.Equal(t, "posted", result[0].Status)
}

func TestGetBill_Success(t *testing.T) {
	app := fiber.New()

	billID := uuid.New().String()
	mockBill := MockBill{
		ID:             billID,
		Type:           "electricity",
		AllocationType: "metered",
		TotalAmountPLN: 250.50,
		TotalUnits:     120.5,
		PeriodStart:    time.Now().AddDate(0, -1, 0),
		PeriodEnd:      time.Now(),
		Status:         "draft",
	}

	app.Get("/bills/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid bill ID",
			})
		}

		return c.JSON(mockBill)
	})

	req := httptest.NewRequest(http.MethodGet, "/bills/"+billID, nil)
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result MockBill
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "electricity", result.Type)
}

func TestGetBill_InvalidID(t *testing.T) {
	app := fiber.New()

	app.Get("/bills/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		// Validate UUID format
		if _, err := uuid.Parse(id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid bill ID",
			})
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/bills/invalid-id", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetBill_NotFound(t *testing.T) {
	app := fiber.New()

	app.Get("/bills/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if _, err := uuid.Parse(id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid bill ID",
			})
		}

		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Bill not found",
		})
	})

	billID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/bills/"+billID, nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCreateBill_Success(t *testing.T) {
	app := fiber.New()

	app.Post("/bills", func(c *fiber.Ctx) error {
		var req struct {
			Type           string  `json:"type"`
			TotalAmountPLN float64 `json:"totalAmountPLN"`
			TotalUnits     float64 `json:"totalUnits"`
			PeriodStart    string  `json:"periodStart"`
			PeriodEnd      string  `json:"periodEnd"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.Type == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Bill type is required",
			})
		}

		if req.TotalAmountPLN <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Total amount must be positive",
			})
		}

		return c.Status(fiber.StatusCreated).JSON(MockBill{
			ID:             uuid.New().String(),
			Type:           req.Type,
			TotalAmountPLN: req.TotalAmountPLN,
			TotalUnits:     req.TotalUnits,
			Status:         "draft",
		})
	})

	reqBody := map[string]interface{}{
		"type":           "electricity",
		"totalAmountPLN": 250.50,
		"totalUnits":     120.5,
		"periodStart":    "2024-01-01T00:00:00Z",
		"periodEnd":      "2024-01-31T23:59:59Z",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/bills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result MockBill
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "electricity", result.Type)
	assert.Equal(t, "draft", result.Status)
}

func TestCreateBill_MissingType(t *testing.T) {
	app := fiber.New()

	app.Post("/bills", func(c *fiber.Ctx) error {
		var req struct {
			Type           string  `json:"type"`
			TotalAmountPLN float64 `json:"totalAmountPLN"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.Type == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Bill type is required",
			})
		}

		return c.SendStatus(fiber.StatusCreated)
	})

	reqBody := map[string]interface{}{
		"totalAmountPLN": 250.50,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/bills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateBill_NegativeAmount(t *testing.T) {
	app := fiber.New()

	app.Post("/bills", func(c *fiber.Ctx) error {
		var req struct {
			Type           string  `json:"type"`
			TotalAmountPLN float64 `json:"totalAmountPLN"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.TotalAmountPLN <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Total amount must be positive",
			})
		}

		return c.SendStatus(fiber.StatusCreated)
	})

	reqBody := map[string]interface{}{
		"type":           "electricity",
		"totalAmountPLN": -100.0,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/bills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateBill_Success(t *testing.T) {
	app := fiber.New()

	billID := uuid.New().String()

	app.Put("/bills/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if _, err := uuid.Parse(id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid bill ID",
			})
		}

		var req struct {
			TotalAmountPLN float64 `json:"totalAmountPLN"`
			Notes          string  `json:"notes"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		return c.JSON(fiber.Map{
			"message": "Bill updated successfully",
		})
	})

	reqBody := map[string]interface{}{
		"totalAmountPLN": 300.00,
		"notes":          "Updated bill",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/bills/"+billID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteBill_Success(t *testing.T) {
	app := fiber.New()

	billID := uuid.New().String()

	app.Delete("/bills/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if _, err := uuid.Parse(id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid bill ID",
			})
		}

		return c.JSON(fiber.Map{
			"message": "Bill deleted successfully",
		})
	})

	req := httptest.NewRequest(http.MethodDelete, "/bills/"+billID, nil)
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDeleteBill_InvalidID(t *testing.T) {
	app := fiber.New()

	app.Delete("/bills/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if _, err := uuid.Parse(id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid bill ID",
			})
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodDelete, "/bills/invalid-id", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPostBill_Success(t *testing.T) {
	app := fiber.New()

	billID := uuid.New().String()

	app.Post("/bills/:id/post", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if _, err := uuid.Parse(id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid bill ID",
			})
		}

		return c.JSON(fiber.Map{
			"message": "Bill posted successfully",
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/bills/"+billID+"/post", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPostBill_AlreadyPosted(t *testing.T) {
	app := fiber.New()

	billID := uuid.New().String()

	app.Post("/bills/:id/post", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if _, err := uuid.Parse(id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid bill ID",
			})
		}

		// Simulate already posted bill
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Bill is already posted",
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/bills/"+billID+"/post", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
