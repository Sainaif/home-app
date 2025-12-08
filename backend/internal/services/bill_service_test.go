package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBillTypeValidation tests bill type validation logic
func TestBillTypeValidation(t *testing.T) {
	validTypes := []string{"electricity", "gas", "internet", "inne"}

	tests := []struct {
		name        string
		billType    string
		customType  *string
		expectValid bool
	}{
		{
			name:        "Valid electricity type",
			billType:    "electricity",
			customType:  nil,
			expectValid: true,
		},
		{
			name:        "Valid gas type",
			billType:    "gas",
			customType:  nil,
			expectValid: true,
		},
		{
			name:        "Valid internet type",
			billType:    "internet",
			customType:  nil,
			expectValid: true,
		},
		{
			name:        "Valid inne type with custom",
			billType:    "inne",
			customType:  stringPtr("Water Bill"),
			expectValid: true,
		},
		{
			name:        "Invalid type",
			billType:    "invalid",
			customType:  nil,
			expectValid: false,
		},
		{
			name:        "Empty type",
			billType:    "",
			customType:  nil,
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, validType := range validTypes {
				if tt.billType == validType {
					isValid = true
					break
				}
			}

			assert.Equal(t, tt.expectValid, isValid, "Type validation mismatch for: %s", tt.billType)
		})
	}
}

// TestBillPeriodValidation tests that period end must be after period start
func TestBillPeriodValidation(t *testing.T) {
	tests := []struct {
		name        string
		periodStart string
		periodEnd   string
		expectValid bool
	}{
		{
			name:        "Valid period - end after start",
			periodStart: "2024-01-01",
			periodEnd:   "2024-01-31",
			expectValid: true,
		},
		{
			name:        "Invalid period - end before start",
			periodStart: "2024-01-31",
			periodEnd:   "2024-01-01",
			expectValid: false,
		},
		{
			name:        "Invalid period - same day",
			periodStart: "2024-01-15",
			periodEnd:   "2024-01-15",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple comparison as strings work for ISO dates
			isValid := tt.periodEnd > tt.periodStart
			assert.Equal(t, tt.expectValid, isValid, "Period validation mismatch")
		})
	}
}

// TestAllocationTypeValidation tests allocation type validation
func TestAllocationTypeValidation(t *testing.T) {
	validAllocTypes := []string{"simple", "metered"}

	tests := []struct {
		name           string
		allocationType string
		expectValid    bool
	}{
		{
			name:           "Valid simple allocation",
			allocationType: "simple",
			expectValid:    true,
		},
		{
			name:           "Valid metered allocation",
			allocationType: "metered",
			expectValid:    true,
		},
		{
			name:           "Invalid allocation type",
			allocationType: "custom",
			expectValid:    false,
		},
		{
			name:           "Empty allocation type",
			allocationType: "",
			expectValid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, validType := range validAllocTypes {
				if tt.allocationType == validType {
					isValid = true
					break
				}
			}
			assert.Equal(t, tt.expectValid, isValid, "Allocation type validation mismatch for: %s", tt.allocationType)
		})
	}
}

// TestBillStatusValidation tests bill status validation
func TestBillStatusValidation(t *testing.T) {
	validStatuses := []string{"draft", "active", "closed", "reopened"}

	tests := []struct {
		name        string
		status      string
		expectValid bool
	}{
		{
			name:        "Valid draft status",
			status:      "draft",
			expectValid: true,
		},
		{
			name:        "Valid active status",
			status:      "active",
			expectValid: true,
		},
		{
			name:        "Valid closed status",
			status:      "closed",
			expectValid: true,
		},
		{
			name:        "Valid reopened status",
			status:      "reopened",
			expectValid: true,
		},
		{
			name:        "Invalid status",
			status:      "pending",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, validStatus := range validStatuses {
				if tt.status == validStatus {
					isValid = true
					break
				}
			}
			assert.Equal(t, tt.expectValid, isValid, "Status validation mismatch for: %s", tt.status)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
