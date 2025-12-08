package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
	"github.com/sainaif/holy-home/internal/utils"
)

var ErrNoPreviousReading = errors.New("no previous meter reading found")

type ConsumptionService struct {
	consumptions repository.ConsumptionRepository
	bills        repository.BillRepository
	users        repository.UserRepository
}

func NewConsumptionService(
	consumptions repository.ConsumptionRepository,
	bills repository.BillRepository,
	users repository.UserRepository,
) *ConsumptionService {
	return &ConsumptionService{
		consumptions: consumptions,
		bills:        bills,
		users:        users,
	}
}

type CreateConsumptionRequest struct {
	BillID     string    `json:"billId"`
	UserID     string    `json:"userId"`
	Units      float64   `json:"units"`
	MeterValue *float64  `json:"meterValue,omitempty"`
	RecordedAt time.Time `json:"recordedAt"`
}

// CreateConsumption records a consumption reading
func (s *ConsumptionService) CreateConsumption(ctx context.Context, req CreateConsumptionRequest, source string) (*models.Consumption, error) {
	// Verify bill exists
	bill, err := s.bills.GetByID(ctx, req.BillID)
	if err != nil {
		return nil, errors.New("bill not found")
	}

	// Check if bill is closed
	if bill.Status == "closed" {
		return nil, errors.New("cannot add consumption to closed bill")
	}

	// Verify user exists and get their group
	user, err := s.users.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, errors.New("user not found")
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

	unitsDec := utils.FloatToDecimalString(unitsValue)

	consumption := &models.Consumption{
		ID:          uuid.New().String(),
		BillID:      req.BillID,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Units:       unitsDec,
		RecordedAt:  req.RecordedAt,
		Source:      source,
	}

	if req.MeterValue != nil {
		meterDec := utils.FloatToDecimalString(*req.MeterValue)
		consumption.MeterValue = &meterDec
	}

	if err := s.consumptions.Create(ctx, consumption); err != nil {
		return nil, fmt.Errorf("failed to create consumption: %w", err)
	}

	return consumption, nil
}

// GetConsumptions retrieves consumptions for a bill, or all consumptions if billID is nil
func (s *ConsumptionService) GetConsumptions(ctx context.Context, billID *string) ([]models.Consumption, error) {
	if billID != nil {
		return s.consumptions.ListByBillID(ctx, *billID)
	}
	// For listing all consumptions, we'd need a List method - for now return empty
	return []models.Consumption{}, nil
}

// GetUserConsumptions retrieves consumptions for a user or their group
func (s *ConsumptionService) GetUserConsumptions(ctx context.Context, userID string, from *time.Time, to *time.Time) ([]models.Consumption, error) {
	// Get user to check if they're in a group
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Determine subject type and ID based on group membership
	subjectType := "user"
	subjectID := userID
	if user.GroupID != nil {
		subjectType = "group"
		subjectID = *user.GroupID
	}

	// Get consumptions for this subject
	consumptions, err := s.consumptions.ListBySubject(ctx, subjectType, subjectID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Filter by date range if specified
	if from != nil || to != nil {
		filtered := make([]models.Consumption, 0)
		for _, c := range consumptions {
			if from != nil && c.RecordedAt.Before(*from) {
				continue
			}
			if to != nil && c.RecordedAt.After(*to) {
				continue
			}
			filtered = append(filtered, c)
		}
		return filtered, nil
	}

	return consumptions, nil
}

// DeleteConsumption deletes a consumption/reading
func (s *ConsumptionService) DeleteConsumption(ctx context.Context, consumptionID string) error {
	consumption, err := s.consumptions.GetByID(ctx, consumptionID)
	if err != nil {
		return errors.New("consumption not found")
	}
	if consumption == nil {
		return errors.New("consumption not found")
	}

	return s.consumptions.Delete(ctx, consumptionID)
}

// MarkConsumptionInvalid marks a consumption as invalid (user or their group must own it)
func (s *ConsumptionService) MarkConsumptionInvalid(ctx context.Context, consumptionID, userID string) error {
	// Find the consumption
	consumption, err := s.consumptions.GetByID(ctx, consumptionID)
	if err != nil || consumption == nil {
		return errors.New("consumption not found")
	}

	// Get user to check their group
	user, err := s.users.GetByID(ctx, userID)
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
	consumption.Source = "invalid"
	return s.consumptions.Update(ctx, consumption)
}

// calculateUnitsFromMeter derives consumption units based on the previous meter reading
func (s *ConsumptionService) calculateUnitsFromMeter(ctx context.Context, subjectID string, subjectType string, currentMeter float64, recordedAt time.Time) (float64, error) {
	// Get all consumptions for this subject and find the most recent one before recordedAt
	consumptions, err := s.consumptions.ListBySubject(ctx, subjectType, subjectID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch consumptions: %w", err)
	}

	var previous *models.Consumption
	for i := range consumptions {
		c := &consumptions[i]
		if c.MeterValue == nil {
			continue
		}
		if !c.RecordedAt.Before(recordedAt) {
			continue
		}
		if previous == nil || c.RecordedAt.After(previous.RecordedAt) {
			previous = c
		}
	}

	if previous == nil || previous.MeterValue == nil {
		return 0, ErrNoPreviousReading
	}

	prevValue := utils.DecimalStringToFloat(*previous.MeterValue)

	units := currentMeter - prevValue
	if units < 0 {
		return 0, errors.New("meter reading cannot be lower than previous reading")
	}

	return units, nil
}
