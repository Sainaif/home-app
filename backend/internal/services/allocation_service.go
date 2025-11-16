package services

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

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

	// Group users by group or show individually
	groupAllocations := make(map[primitive.ObjectID]struct {
		groupID   primitive.ObjectID
		groupName string
		weight    float64
		amount    float64
	})
	individualAllocations := []AllocationBreakdown{}

	for _, u := range users {
		weight := userWeights[u.ID]
		amount := (weight / totalWeight) * totalAmount

		if u.GroupID != nil {
			// User is in a group - aggregate to group
			if existing, ok := groupAllocations[*u.GroupID]; ok {
				existing.amount += amount
				groupAllocations[*u.GroupID] = existing
			} else {
				// Find group name
				groupName := ""
				for _, g := range groups {
					if g.ID == *u.GroupID {
						groupName = g.Name
						break
					}
				}
				groupAllocations[*u.GroupID] = struct {
					groupID   primitive.ObjectID
					groupName string
					weight    float64
					amount    float64
				}{
					groupID:   *u.GroupID,
					groupName: groupName,
					weight:    weight,
					amount:    amount,
				}
			}
		} else {
			// User is not in a group - show individually with Name
			individualAllocations = append(individualAllocations, AllocationBreakdown{
				SubjectID:   u.ID,
				SubjectType: "user",
				SubjectName: u.Name,
				Weight:      weight,
				Amount:      utils.RoundToTwoDecimals(amount),
			})
		}
	}

	// Build final breakdown
	breakdown := make([]AllocationBreakdown, 0, len(groupAllocations)+len(individualAllocations))

	// Add group allocations
	for _, ga := range groupAllocations {
		breakdown = append(breakdown, AllocationBreakdown{
			SubjectID:   ga.groupID,
			SubjectType: "group",
			SubjectName: ga.groupName,
			Weight:      ga.weight,
			Amount:      utils.RoundToTwoDecimals(ga.amount),
		})
	}

	// Add individual allocations
	breakdown = append(breakdown, individualAllocations...)

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

	// Calculate total consumed units from readings (aggregated by subject)
	subjectUnits := make(map[primitive.ObjectID]float64)
	totalConsumedUnits := 0.0

	for _, c := range consumptions {
		units, err := utils.DecimalToFloat(c.Units)
		if err != nil {
			continue
		}

		if units <= 0 && c.MeterValue != nil {
			derivedUnits, derr := s.deriveUnitsFromMeter(ctx, c)
			switch {
			case derr == nil:
				units = derivedUnits
			case errors.Is(derr, ErrNoPreviousReading):
				fallback, ferr := utils.DecimalToFloat(*c.MeterValue)
				if ferr == nil {
					units = fallback
				}
			default:
				return nil, derr
			}
		}

		if units <= 0 {
			continue
		}

		subjectUnits[c.SubjectID] += units // Aggregate units per subject (group or user)
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

	// Group users by group or show individually
	groupAllocations := make(map[primitive.ObjectID]struct {
		groupID        primitive.ObjectID
		groupName      string
		weight         float64
		personalAmount float64
		sharedAmount   float64
		units          float64
	})
	individualAllocations := []AllocationBreakdown{}

	for _, u := range users {
		weight := userWeights[u.ID]

		// Determine subject ID for consumption lookup
		var subjectID primitive.ObjectID
		if u.GroupID != nil {
			subjectID = *u.GroupID
		} else {
			subjectID = u.ID
		}

		// Personal amount based on consumption
		units := subjectUnits[subjectID]
		personalAmount := units * ratePerUnit

		// Shared amount based on weight
		sharedAmount := (weight / totalWeight) * sharedPool

		if u.GroupID != nil {
			// User is in a group - aggregate to group
			if existing, ok := groupAllocations[*u.GroupID]; ok {
				existing.sharedAmount += sharedAmount
				// Personal amount is already aggregated at the group level from consumption
				groupAllocations[*u.GroupID] = existing
			} else {
				// Find group name
				groupName := ""
				for _, g := range groups {
					if g.ID == *u.GroupID {
						groupName = g.Name
						break
					}
				}
				groupAllocations[*u.GroupID] = struct {
					groupID        primitive.ObjectID
					groupName      string
					weight         float64
					personalAmount float64
					sharedAmount   float64
					units          float64
				}{
					groupID:        *u.GroupID,
					groupName:      groupName,
					weight:         weight,
					personalAmount: personalAmount, // From group's total consumption
					sharedAmount:   sharedAmount,
					units:          units,
				}
			}
		} else {
			// User is not in a group - show individually
			totalUserAmount := personalAmount + sharedAmount

			individualAllocations = append(individualAllocations, AllocationBreakdown{
				SubjectID:      u.ID,
				SubjectType:    "user",
				SubjectName:    u.Name,
				Weight:         weight,
				Amount:         utils.RoundToTwoDecimals(totalUserAmount),
				PersonalAmount: floatPtr(utils.RoundToTwoDecimals(personalAmount)),
				SharedAmount:   floatPtr(utils.RoundToTwoDecimals(sharedAmount)),
				Units:          floatPtr(utils.RoundToThreeDecimals(units)),
			})
		}
	}

	// Build final breakdown
	breakdown := make([]AllocationBreakdown, 0, len(groupAllocations)+len(individualAllocations))

	// Add group allocations
	for _, ga := range groupAllocations {
		totalAmount := ga.personalAmount + ga.sharedAmount
		breakdown = append(breakdown, AllocationBreakdown{
			SubjectID:      ga.groupID,
			SubjectType:    "group",
			SubjectName:    ga.groupName,
			Weight:         ga.weight,
			Amount:         utils.RoundToTwoDecimals(totalAmount),
			PersonalAmount: floatPtr(utils.RoundToTwoDecimals(ga.personalAmount)),
			SharedAmount:   floatPtr(utils.RoundToTwoDecimals(ga.sharedAmount)),
			Units:          floatPtr(utils.RoundToThreeDecimals(ga.units)),
		})
	}

	// Add individual allocations
	breakdown = append(breakdown, individualAllocations...)

	return breakdown, nil
}

// GetAllocationBreakdown returns allocation breakdown for a bill
func (s *AllocationService) GetAllocationBreakdown(ctx context.Context, billID primitive.ObjectID) ([]AllocationBreakdown, error) {
	// First, check if allocations already exist in the database
	cursor, err := s.db.Collection("allocations").Find(ctx, bson.M{"bill_id": billID})
	if err != nil {
		return nil, fmt.Errorf("failed to query allocations: %w", err)
	}
	defer cursor.Close(ctx)

	var storedAllocations []struct {
		SubjectID    primitive.ObjectID   `bson:"subject_id"`
		SubjectType  string               `bson:"subject_type"`
		AllocatedPLN primitive.Decimal128 `bson:"allocated_pln"`
	}
	if err := cursor.All(ctx, &storedAllocations); err != nil {
		return nil, fmt.Errorf("failed to decode allocations: %w", err)
	}

	// If allocations exist, return them
	if len(storedAllocations) > 0 {
		breakdown := make([]AllocationBreakdown, 0, len(storedAllocations))

		for _, alloc := range storedAllocations {
			amount, _ := utils.DecimalToFloat(alloc.AllocatedPLN)

			// Get subject name
			var subjectName string
			if alloc.SubjectType == "user" {
				var user models.User
				if err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": alloc.SubjectID}).Decode(&user); err == nil {
					subjectName = user.Name
				}
			} else if alloc.SubjectType == "group" {
				var group models.Group
				if err := s.db.Collection("groups").FindOne(ctx, bson.M{"_id": alloc.SubjectID}).Decode(&group); err == nil {
					subjectName = group.Name
				}
			}

			breakdown = append(breakdown, AllocationBreakdown{
				SubjectID:   alloc.SubjectID,
				SubjectType: alloc.SubjectType,
				SubjectName: subjectName,
				Weight:      1.0, // Not applicable for stored allocations
				Amount:      utils.RoundToTwoDecimals(amount),
			})
		}

		return breakdown, nil
	}

	// If no allocations exist, calculate them on-the-fly (for draft bills)
	// Get the bill
	var bill models.Bill
	err = s.db.Collection("bills").FindOne(ctx, bson.M{"_id": billID}).Decode(&bill)
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

// deriveUnitsFromMeter calculates usage delta when only meter readings were stored (legacy data)
func (s *AllocationService) deriveUnitsFromMeter(ctx context.Context, consumption models.Consumption) (float64, error) {
	if consumption.MeterValue == nil {
		return 0, nil
	}

	currentValue, err := utils.DecimalToFloat(*consumption.MeterValue)
	if err != nil {
		return 0, fmt.Errorf("invalid meter value: %w", err)
	}

	filter := bson.M{
		"subject_id":   consumption.SubjectID,
		"subject_type": consumption.SubjectType,
		"meter_value":  bson.M{"$ne": nil},
		"recorded_at":  bson.M{"$lt": consumption.RecordedAt},
	}
	opts := options.FindOne().SetSort(bson.D{{Key: "recorded_at", Value: -1}})

	var previous models.Consumption
	err = s.db.Collection("consumptions").FindOne(ctx, filter, opts).Decode(&previous)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to fetch previous reading: %w", err)
	}

	if previous.MeterValue == nil {
		return 0, nil
	}

	prevValue, err := utils.DecimalToFloat(*previous.MeterValue)
	if err != nil {
		return 0, fmt.Errorf("invalid previous meter value: %w", err)
	}

	units := currentValue - prevValue
	if units < 0 {
		return 0, errors.New("meter reading cannot be lower than previous reading")
	}

	return units, nil
}
