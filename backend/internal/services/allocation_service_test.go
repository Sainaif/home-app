package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestAllocationBreakdown_RoundingPrecision(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   float64
	}{
		{
			name:   "Two decimal places",
			amount: 123.456,
			want:   123.46,
		},
		{
			name:   "Exact two decimals",
			amount: 100.50,
			want:   100.50,
		},
		{
			name:   "Round down",
			amount: 99.994,
			want:   99.99,
		},
		{
			name:   "Round up",
			amount: 99.995,
			want:   100.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rounded := utils.RoundToTwoDecimals(tt.amount)
			assert.Equal(t, tt.want, rounded)
		})
	}
}

func TestAllocationBreakdown_UnitsRounding(t *testing.T) {
	tests := []struct {
		name  string
		units float64
		want  float64
	}{
		{
			name:  "Three decimal places",
			units: 123.4567,
			want:  123.457,
		},
		{
			name:  "Exact three decimals",
			units: 100.500,
			want:  100.500,
		},
		{
			name:  "Round down",
			units: 99.9994,
			want:  99.999,
		},
		{
			name:  "Round up",
			units: 99.9995,
			want:  100.000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rounded := utils.RoundToThreeDecimals(tt.units)
			assert.Equal(t, tt.want, rounded)
		})
	}
}

// TestGroupAggregationLogic tests the core logic of group aggregation
// This is a unit test focusing on the calculation logic without database dependencies
func TestGroupAggregationLogic(t *testing.T) {
	t.Run("Users in same group should have costs aggregated", func(t *testing.T) {
		// Scenario: Group with 2 users (A, B) and 1 individual user (C)
		// Each should have 1/3 cost, but group should show 2/3 total

		totalAmount := 300.0
		expectedGroupShare := 200.0      // 2/3 of 300
		expectedIndividualShare := 100.0 // 1/3 of 300

		groupID := uuid.New().String()
		userA := models.User{
			ID:      uuid.New().String(),
			Name:    "User A",
			GroupID: &groupID,
		}
		userB := models.User{
			ID:      uuid.New().String(),
			Name:    "User B",
			GroupID: &groupID,
		}
		userC := models.User{
			ID:   uuid.New().String(),
			Name: "User C",
			// No GroupID - individual user
		}

		users := []models.User{userA, userB, userC}
		totalWeight := 3.0 // Each user has weight 1.0

		// Simulate the aggregation logic from CalculateSimpleAllocation
		groupAllocations := make(map[string]float64)
		var individualAllocations []struct {
			userID string
			amount float64
		}

		for _, u := range users {
			weight := 1.0
			amount := (weight / totalWeight) * totalAmount

			if u.GroupID != nil {
				// Aggregate to group
				groupAllocations[*u.GroupID] += amount
			} else {
				// Individual user
				individualAllocations = append(individualAllocations, struct {
					userID string
					amount float64
				}{u.ID, amount})
			}
		}

		// Assertions
		assert.Len(t, groupAllocations, 1, "Should have exactly 1 group")
		assert.Len(t, individualAllocations, 1, "Should have exactly 1 individual user")

		groupAmount := groupAllocations[groupID]
		assert.InDelta(t, expectedGroupShare, groupAmount, 0.01, "Group should have 2/3 of total")

		individualAmount := individualAllocations[0].amount
		assert.InDelta(t, expectedIndividualShare, individualAmount, 0.01, "Individual should have 1/3 of total")

		// Total should equal original amount
		total := groupAmount + individualAmount
		assert.InDelta(t, totalAmount, total, 0.01, "Total allocations should equal original amount")
	})

	t.Run("Multiple groups with different sizes", func(t *testing.T) {
		// Scenario: Group1 (2 users), Group2 (3 users), Individual (1 user)
		// Total 6 users, each gets 1/6

		totalAmount := 600.0

		group1ID := uuid.New().String()
		group2ID := uuid.New().String()

		users := []models.User{
			{ID: uuid.New().String(), Name: "G1-User1", GroupID: &group1ID},
			{ID: uuid.New().String(), Name: "G1-User2", GroupID: &group1ID},
			{ID: uuid.New().String(), Name: "G2-User1", GroupID: &group2ID},
			{ID: uuid.New().String(), Name: "G2-User2", GroupID: &group2ID},
			{ID: uuid.New().String(), Name: "G2-User3", GroupID: &group2ID},
			{ID: uuid.New().String(), Name: "Individual", GroupID: nil},
		}

		totalWeight := 6.0
		groupAllocations := make(map[string]float64)
		individualCount := 0

		for _, u := range users {
			weight := 1.0
			amount := (weight / totalWeight) * totalAmount

			if u.GroupID != nil {
				groupAllocations[*u.GroupID] += amount
			} else {
				individualCount++
			}
		}

		// Assertions
		assert.Len(t, groupAllocations, 2, "Should have exactly 2 groups")
		assert.Equal(t, 1, individualCount, "Should have 1 individual user")

		expectedGroup1 := 200.0 // 2/6 of 600
		expectedGroup2 := 300.0 // 3/6 of 600

		assert.InDelta(t, expectedGroup1, groupAllocations[group1ID], 0.01)
		assert.InDelta(t, expectedGroup2, groupAllocations[group2ID], 0.01)
	})

	t.Run("All users in groups", func(t *testing.T) {
		totalAmount := 300.0
		groupID := uuid.New().String()

		users := []models.User{
			{ID: uuid.New().String(), Name: "User A", GroupID: &groupID},
			{ID: uuid.New().String(), Name: "User B", GroupID: &groupID},
			{ID: uuid.New().String(), Name: "User C", GroupID: &groupID},
		}

		totalWeight := 3.0
		groupAllocations := make(map[string]float64)

		for _, u := range users {
			weight := 1.0
			amount := (weight / totalWeight) * totalAmount
			groupAllocations[*u.GroupID] += amount
		}

		assert.Len(t, groupAllocations, 1, "Should have exactly 1 group")
		assert.InDelta(t, totalAmount, groupAllocations[groupID], 0.01, "Group should have 100% of total")
	})

	t.Run("No users in groups", func(t *testing.T) {
		totalAmount := 300.0

		users := []models.User{
			{ID: uuid.New().String(), Name: "User A", GroupID: nil},
			{ID: uuid.New().String(), Name: "User B", GroupID: nil},
			{ID: uuid.New().String(), Name: "User C", GroupID: nil},
		}

		totalWeight := 3.0
		groupAllocations := make(map[string]float64)
		individualCount := 0

		for _, u := range users {
			weight := 1.0
			amount := (weight / totalWeight) * totalAmount

			if u.GroupID != nil {
				groupAllocations[*u.GroupID] += amount
			} else {
				individualCount++
			}
		}

		assert.Len(t, groupAllocations, 0, "Should have no groups")
		assert.Equal(t, 3, individualCount, "Should have 3 individual users")
	})
}
