package services

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/utils"
)

type AllocationService struct {
	db *mongo.Database
}

func NewAllocationService(db *mongo.Database) *AllocationService {
	return &AllocationService{db: db}
}

// AllocationBreakdown represents cost breakdown per user/group
type AllocationBreakdown struct {
	SubjectID   primitive.ObjectID `json:"subjectId"`
	SubjectType string             `json:"subjectType"` // "user" or "group"
	SubjectName string             `json:"subjectName"`
	Weight      float64            `json:"weight"`
	Amount      float64            `json:"amount"`
	// For metered allocation (electricity)
	PersonalAmount *float64 `json:"personalAmount,omitempty"`
	SharedAmount   *float64 `json:"sharedAmount,omitempty"`
	Units          *float64 `json:"units,omitempty"`
}

// CalculateSimpleAllocation divides total cost by weights
func (s *AllocationService) CalculateSimpleAllocation(ctx context.Context, billID primitive.ObjectID, totalAmount float64) ([]AllocationBreakdown, error) {
	// Get all active users
	users, err := s.getAllActiveUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	if len(users) == 0 {
		return nil, errors.New("no active users found")
	}

	// Get all groups
	groups, err := s.getAllGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	// Build group weight map
	groupWeights := make(map[primitive.ObjectID]float64)
	for _, g := range groups {
		groupWeights[g.ID] = g.Weight
	}

	// Calculate weights for each user
	userWeights := make(map[primitive.ObjectID]float64)
	totalWeight := 0.0

	for _, u := range users {
		weight := 1.0 // default weight
		if u.GroupID != nil {
			if gw, ok := groupWeights[*u.GroupID]; ok {
				weight = gw
			}
		}
		userWeights[u.ID] = weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return nil, errors.New("total weight is zero")
	}

	// Calculate allocation per user
	breakdown := make([]AllocationBreakdown, 0, len(users))
	for _, u := range users {
		weight := userWeights[u.ID]
		amount := (weight / totalWeight) * totalAmount

		breakdown = append(breakdown, AllocationBreakdown{
			SubjectID:   u.ID,
			SubjectType: "user",
			SubjectName: u.Email,
			Weight:      weight,
			Amount:      utils.RoundToTwoDecimals(amount),
		})
	}

	return breakdown, nil
}

// CalculateMeteredAllocation calculates based on meter readings + shared common area
func (s *AllocationService) CalculateMeteredAllocation(ctx context.Context, billID primitive.ObjectID, totalAmount float64, totalUnits *float64) ([]AllocationBreakdown, error) {
	if totalUnits == nil || *totalUnits == 0 {
		return nil, errors.New("totalUnits is required for metered allocation")
	}

	// Get all consumptions for this bill
	consumptions, err := s.getConsumptionsForBill(ctx, billID)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumptions: %w", err)
	}

	// Get all active users
	users, err := s.getAllActiveUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	if len(users) == 0 {
		return nil, errors.New("no active users found")
	}

	// Get all groups
	groups, err := s.getAllGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	// Build group weight map
	groupWeights := make(map[primitive.ObjectID]float64)
	for _, g := range groups {
		groupWeights[g.ID] = g.Weight
	}

	// Calculate user weights
	userWeights := make(map[primitive.ObjectID]float64)
	totalWeight := 0.0

	for _, u := range users {
		weight := 1.0 // default weight
		if u.GroupID != nil {
			if gw, ok := groupWeights[*u.GroupID]; ok {
				weight = gw
			}
		}
		userWeights[u.ID] = weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return nil, errors.New("total weight is zero")
	}

	// Calculate total consumed units from readings
	userUnits := make(map[primitive.ObjectID]float64)
	totalConsumedUnits := 0.0

	for _, c := range consumptions {
		units, err := utils.DecimalToFloat(c.Units)
		if err != nil {
			continue
		}
		userUnits[c.UserID] = units
		totalConsumedUnits += units
	}

	// Calculate personal and shared pools
	personalPoolRatio := totalConsumedUnits / *totalUnits
	if personalPoolRatio > 1.0 {
		personalPoolRatio = 1.0 // cap at 100%
	}
	sharedPoolRatio := 1.0 - personalPoolRatio

	personalPool := totalAmount * personalPoolRatio
	sharedPool := totalAmount * sharedPoolRatio

	// Calculate rate per unit for personal usage
	ratePerUnit := 0.0
	if totalConsumedUnits > 0 {
		ratePerUnit = personalPool / totalConsumedUnits
	}

	// Calculate allocation per user
	breakdown := make([]AllocationBreakdown, 0, len(users))
	for _, u := range users {
		weight := userWeights[u.ID]

		// Personal amount based on consumption
		units := userUnits[u.ID]
		personalAmount := units * ratePerUnit

		// Shared amount based on weight
		sharedAmount := (weight / totalWeight) * sharedPool

		// Total amount
		totalUserAmount := personalAmount + sharedAmount

		breakdown = append(breakdown, AllocationBreakdown{
			SubjectID:      u.ID,
			SubjectType:    "user",
			SubjectName:    u.Email,
			Weight:         weight,
			Amount:         utils.RoundToTwoDecimals(totalUserAmount),
			PersonalAmount: floatPtr(utils.RoundToTwoDecimals(personalAmount)),
			SharedAmount:   floatPtr(utils.RoundToTwoDecimals(sharedAmount)),
			Units:          floatPtr(utils.RoundToThreeDecimals(units)),
		})
	}

	return breakdown, nil
}

// GetAllocationBreakdown returns allocation breakdown for a bill
func (s *AllocationService) GetAllocationBreakdown(ctx context.Context, billID primitive.ObjectID) ([]AllocationBreakdown, error) {
	// Get the bill
	var bill models.Bill
	err := s.db.Collection("bills").FindOne(ctx, bson.M{"_id": billID}).Decode(&bill)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("bill not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Get total amount
	totalAmount, err := utils.DecimalToFloat(bill.TotalAmountPLN)
	if err != nil {
		return nil, fmt.Errorf("invalid total amount: %w", err)
	}

	// Determine allocation type
	allocationType := "simple" // default
	if bill.AllocationType != nil {
		allocationType = *bill.AllocationType
	}

	// Calculate allocation based on type
	if allocationType == "metered" {
		var totalUnits *float64
		if bill.TotalUnits != (primitive.Decimal128{}) {
			units, err := utils.DecimalToFloat(bill.TotalUnits)
			if err == nil {
				totalUnits = &units
			}
		}
		return s.CalculateMeteredAllocation(ctx, billID, totalAmount, totalUnits)
	}

	return s.CalculateSimpleAllocation(ctx, billID, totalAmount)
}

// Helper functions
func (s *AllocationService) getAllActiveUsers(ctx context.Context) ([]models.User, error) {
	cursor, err := s.db.Collection("users").Find(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (s *AllocationService) getAllGroups(ctx context.Context) ([]models.Group, error) {
	cursor, err := s.db.Collection("groups").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var groups []models.Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *AllocationService) getConsumptionsForBill(ctx context.Context, billID primitive.ObjectID) ([]models.Consumption, error) {
	cursor, err := s.db.Collection("consumptions").Find(ctx, bson.M{"bill_id": billID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var consumptions []models.Consumption
	if err := cursor.All(ctx, &consumptions); err != nil {
		return nil, err
	}
	return consumptions, nil
}

func floatPtr(f float64) *float64 {
	return &f
}
