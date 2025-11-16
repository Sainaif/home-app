package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/utils"
)

var ErrNoPreviousReading = errors.New("no previous meter reading found")

type ConsumptionService struct {
	db *mongo.Database
}

func NewConsumptionService(db *mongo.Database) *ConsumptionService {
	return &ConsumptionService{db: db}
}

type CreateConsumptionRequest struct {
	BillID     primitive.ObjectID `json:"billId"`
	UserID     primitive.ObjectID `json:"userId"`
	Units      float64            `json:"units"`
	MeterValue *float64           `json:"meterValue,omitempty"`
	RecordedAt time.Time          `json:"recordedAt"`
}

// CreateConsumption records a consumption reading
func (s *ConsumptionService) CreateConsumption(ctx context.Context, req CreateConsumptionRequest, source string) (*models.Consumption, error) {
	// Verify bill exists
	var bill models.Bill
	err := s.db.Collection("bills").FindOne(ctx, bson.M{"_id": req.BillID}).Decode(&bill)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("bill not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Check if bill is closed
	if bill.Status == "closed" {
		return nil, errors.New("cannot add consumption to closed bill")
	}

	// Verify user exists and get their group
	var user models.User
	err = s.db.Collection("users").FindOne(ctx, bson.M{"_id": req.UserID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Determine subject type and ID based on user's group membership
	subjectType := "user"
	subjectID := req.UserID
	if user.GroupID != nil {
		subjectType = "group"
		subjectID = *user.GroupID
	}

	unitsValue := req.Units

	if unitsValue <= 0 {
		if req.MeterValue == nil {
			return nil, errors.New("units must be greater than zero when no meter reading is provided")
		}

		computedUnits, err := s.calculateUnitsFromMeter(ctx, subjectID, subjectType, *req.MeterValue, req.RecordedAt)
		switch {
		case err == nil:
			unitsValue = computedUnits
		case errors.Is(err, ErrNoPreviousReading):
			unitsValue = *req.MeterValue
		default:
			return nil, err
		}
	}

	unitsDec, err := utils.DecimalFromFloat(unitsValue)
	if err != nil {
		return nil, fmt.Errorf("invalid units: %w", err)
	}

	consumption := models.Consumption{
		ID:          primitive.NewObjectID(),
		BillID:      req.BillID,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Units:       unitsDec,
		RecordedAt:  req.RecordedAt,
		Source:      source,
	}

	if req.MeterValue != nil {
		meterDec, err := utils.DecimalFromFloat(*req.MeterValue)
		if err != nil {
			return nil, fmt.Errorf("invalid meter value: %w", err)
		}
		consumption.MeterValue = &meterDec
	}

	_, err = s.db.Collection("consumptions").InsertOne(ctx, consumption)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumption: %w", err)
	}

	return &consumption, nil
}

// GetConsumptions retrieves consumptions for a bill, or all consumptions if billID is nil
func (s *ConsumptionService) GetConsumptions(ctx context.Context, billID *primitive.ObjectID) ([]models.Consumption, error) {
	filter := bson.M{}
	if billID != nil {
		filter["bill_id"] = *billID
	}

	cursor, err := s.db.Collection("consumptions").Find(ctx, filter, nil)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var consumptions []models.Consumption
	if err := cursor.All(ctx, &consumptions); err != nil {
		return nil, fmt.Errorf("failed to decode consumptions: %w", err)
	}

	return consumptions, nil
}

// GetUserConsumptions retrieves consumptions for a user or their group
func (s *ConsumptionService) GetUserConsumptions(ctx context.Context, userID primitive.ObjectID, from *time.Time, to *time.Time) ([]models.Consumption, error) {
	// Get user to check if they're in a group
	var user models.User
	err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Build filter based on whether user is in a group
	filter := bson.M{}
	if user.GroupID != nil {
		// User is in a group, find group consumptions
		filter["subject_type"] = "group"
		filter["subject_id"] = *user.GroupID
	} else {
		// User is not in a group, find individual consumptions
		filter["subject_type"] = "user"
		filter["subject_id"] = userID
	}

	if from != nil || to != nil {
		dateFilter := bson.M{}
		if from != nil {
			dateFilter["$gte"] = *from
		}
		if to != nil {
			dateFilter["$lte"] = *to
		}
		filter["recorded_at"] = dateFilter
	}

	cursor, err := s.db.Collection("consumptions").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var consumptions []models.Consumption
	if err := cursor.All(ctx, &consumptions); err != nil {
		return nil, fmt.Errorf("failed to decode consumptions: %w", err)
	}

	return consumptions, nil
}

// DeleteConsumption deletes a consumption/reading
func (s *ConsumptionService) DeleteConsumption(ctx context.Context, consumptionID primitive.ObjectID) error {
	result, err := s.db.Collection("consumptions").DeleteOne(ctx, bson.M{"_id": consumptionID})
	if err != nil {
		return fmt.Errorf("failed to delete consumption: %w", err)
	}

	if result.DeletedCount == 0 {
		return errors.New("consumption not found")
	}

	return nil
}

// MarkConsumptionInvalid marks a consumption as invalid (user or their group must own it)
func (s *ConsumptionService) MarkConsumptionInvalid(ctx context.Context, consumptionID, userID primitive.ObjectID) error {
	// Find the consumption
	var consumption models.Consumption
	err := s.db.Collection("consumptions").FindOne(ctx, bson.M{"_id": consumptionID}).Decode(&consumption)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("consumption not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	// Get user to check their group
	var user models.User
	err = s.db.Collection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify ownership - either direct user ownership or group ownership
	isOwner := false
	if consumption.SubjectType == "user" && consumption.SubjectID == userID {
		isOwner = true
	} else if consumption.SubjectType == "group" && user.GroupID != nil && consumption.SubjectID == *user.GroupID {
		isOwner = true
	}

	if !isOwner {
		return errors.New("you can only mark your own readings as invalid")
	}

	// Update source to indicate it's invalid
	_, err = s.db.Collection("consumptions").UpdateOne(
		ctx,
		bson.M{"_id": consumptionID},
		bson.M{"$set": bson.M{"source": "invalid"}},
	)
	if err != nil {
		return fmt.Errorf("failed to mark consumption as invalid: %w", err)
	}

	return nil
}

// calculateUnitsFromMeter derives consumption units based on the previous meter reading
func (s *ConsumptionService) calculateUnitsFromMeter(ctx context.Context, subjectID primitive.ObjectID, subjectType string, currentMeter float64, recordedAt time.Time) (float64, error) {
	filter := bson.M{
		"subject_id":   subjectID,
		"subject_type": subjectType,
		"meter_value":  bson.M{"$ne": nil},
		"recorded_at":  bson.M{"$lt": recordedAt},
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "recorded_at", Value: -1}})

	var previous models.Consumption
	err := s.db.Collection("consumptions").FindOne(ctx, filter, opts).Decode(&previous)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, ErrNoPreviousReading
		}
		return 0, fmt.Errorf("failed to fetch previous reading: %w", err)
	}

	if previous.MeterValue == nil {
		return 0, ErrNoPreviousReading
	}

	prevValue, err := utils.DecimalToFloat(*previous.MeterValue)
	if err != nil {
		return 0, fmt.Errorf("invalid previous meter value: %w", err)
	}

	units := currentMeter - prevValue
	if units < 0 {
		return 0, errors.New("meter reading cannot be lower than previous reading")
	}

	return units, nil
}
