package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/utils"
)

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

	// Verify user exists
	var user models.User
	err = s.db.Collection("users").FindOne(ctx, bson.M{"_id": req.UserID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	unitsDec, err := utils.DecimalFromFloat(req.Units)
	if err != nil {
		return nil, fmt.Errorf("invalid units: %w", err)
	}

	consumption := models.Consumption{
		ID:         primitive.NewObjectID(),
		BillID:     req.BillID,
		UserID:     req.UserID,
		Units:      unitsDec,
		RecordedAt: req.RecordedAt,
		Source:     source,
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

// GetUserConsumptions retrieves consumptions for a user
func (s *ConsumptionService) GetUserConsumptions(ctx context.Context, userID primitive.ObjectID, from *time.Time, to *time.Time) ([]models.Consumption, error) {
	filter := bson.M{"user_id": userID}

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

// MarkConsumptionInvalid marks a consumption as invalid (user must own it)
func (s *ConsumptionService) MarkConsumptionInvalid(ctx context.Context, consumptionID, userID primitive.ObjectID) error {
	// Find the consumption and verify ownership
	var consumption models.Consumption
	err := s.db.Collection("consumptions").FindOne(ctx, bson.M{"_id": consumptionID}).Decode(&consumption)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("consumption not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	// Verify user owns this consumption
	if consumption.UserID != userID {
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
