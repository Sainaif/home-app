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

type BillService struct {
	db *mongo.Database
}

func NewBillService(db *mongo.Database) *BillService {
	return &BillService{db: db}
}

type CreateBillRequest struct {
	Type           string    `json:"type"` // electricity, gas, internet, inne
	CustomType     *string   `json:"customType,omitempty"` // required when type is "inne"
	PeriodStart    time.Time `json:"periodStart"`
	PeriodEnd      time.Time `json:"periodEnd"`
	TotalAmountPLN float64   `json:"totalAmountPLN"`
	TotalUnits     *float64  `json:"totalUnits,omitempty"`
	Notes          *string   `json:"notes,omitempty"`
}

type AllocateRequest struct {
	Strategy string             `json:"strategy"` // equal, proportional, weights, override
	Weights  map[string]float64 `json:"weights,omitempty"`
}

// CreateBill creates a new bill (ADMIN only)
func (s *BillService) CreateBill(ctx context.Context, req CreateBillRequest) (*models.Bill, error) {
	// Validate bill type
	validTypes := map[string]bool{"electricity": true, "gas": true, "internet": true, "inne": true}
	if !validTypes[req.Type] {
		return nil, errors.New("invalid bill type")
	}

	// Validate customType is provided when type is "inne"
	if req.Type == "inne" && (req.CustomType == nil || *req.CustomType == "") {
		return nil, errors.New("customType is required when type is 'inne'")
	}

	// Validate customType is not provided for other types
	if req.Type != "inne" && req.CustomType != nil {
		return nil, errors.New("customType should only be provided when type is 'inne'")
	}

	if req.PeriodEnd.Before(req.PeriodStart) {
		return nil, errors.New("period end must be after period start")
	}

	amountDec, err := utils.DecimalFromFloat(req.TotalAmountPLN)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	bill := models.Bill{
		ID:             primitive.NewObjectID(),
		Type:           req.Type,
		CustomType:     req.CustomType,
		PeriodStart:    req.PeriodStart,
		PeriodEnd:      req.PeriodEnd,
		TotalAmountPLN: amountDec,
		Notes:          req.Notes,
		Status:         "draft",
		CreatedAt:      time.Now(),
	}

	if req.TotalUnits != nil {
		unitsDec, err := utils.DecimalFromFloat(*req.TotalUnits)
		if err != nil {
			return nil, fmt.Errorf("invalid units: %w", err)
		}
		bill.TotalUnits = unitsDec
	}

	_, err = s.db.Collection("bills").InsertOne(ctx, bill)
	if err != nil {
		return nil, fmt.Errorf("failed to create bill: %w", err)
	}

	return &bill, nil
}

// GetBills retrieves bills with optional filtering
func (s *BillService) GetBills(ctx context.Context, billType *string, from *time.Time, to *time.Time) ([]models.Bill, error) {
	filter := bson.M{}

	if billType != nil {
		filter["type"] = *billType
	}

	if from != nil || to != nil {
		dateFilter := bson.M{}
		if from != nil {
			dateFilter["$gte"] = *from
		}
		if to != nil {
			dateFilter["$lte"] = *to
		}
		filter["period_start"] = dateFilter
	}

	cursor, err := s.db.Collection("bills").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var bills []models.Bill
	if err := cursor.All(ctx, &bills); err != nil {
		return nil, fmt.Errorf("failed to decode bills: %w", err)
	}

	return bills, nil
}

// GetBill retrieves a single bill
func (s *BillService) GetBill(ctx context.Context, billID primitive.ObjectID) (*models.Bill, error) {
	var bill models.Bill
	err := s.db.Collection("bills").FindOne(ctx, bson.M{"_id": billID}).Decode(&bill)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("bill not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &bill, nil
}

// AllocateBill performs cost allocation based on strategy (ADMIN only)
func (s *BillService) AllocateBill(ctx context.Context, billID primitive.ObjectID, req AllocateRequest) error {
	// Get bill
	bill, err := s.GetBill(ctx, billID)
	if err != nil {
		return err
	}

	if bill.Status == "closed" {
		return errors.New("cannot allocate closed bill")
	}

	// Delete existing allocations
	_, err = s.db.Collection("allocations").DeleteMany(ctx, bson.M{"bill_id": billID})
	if err != nil {
		return fmt.Errorf("failed to clear allocations: %w", err)
	}

	// Perform allocation based on type and strategy
	switch bill.Type {
	case "electricity":
		return s.allocateElectricity(ctx, bill, req)
	case "gas":
		return s.allocateGas(ctx, bill, req)
	case "internet":
		return s.allocateInternet(ctx, bill)
	case "inne":
		return s.allocateInne(ctx, bill, req)
	default:
		return errors.New("unsupported bill type")
	}
}

// allocateElectricity implements the complex electricity allocation logic
func (s *BillService) allocateElectricity(ctx context.Context, bill *models.Bill, req AllocateRequest) error {
	// Get all consumptions for this bill
	cursor, err := s.db.Collection("consumptions").Find(ctx, bson.M{"bill_id": bill.ID})
	if err != nil {
		return fmt.Errorf("failed to get consumptions: %w", err)
	}
	defer cursor.Close(ctx)

	var consumptions []models.Consumption
	if err := cursor.All(ctx, &consumptions); err != nil {
		return fmt.Errorf("failed to decode consumptions: %w", err)
	}

	// Calculate total individual units
	totalIndividualUnits := 0.0
	userUnits := make(map[primitive.ObjectID]float64)
	for _, c := range consumptions {
		units, _ := utils.DecimalToFloat(c.Units)
		userUnits[c.UserID] = units
		totalIndividualUnits += units
	}

	// Get total units and amount
	totalUnits, _ := utils.DecimalToFloat(bill.TotalUnits)
	totalAmount, _ := utils.DecimalToFloat(bill.TotalAmountPLN)

	// Calculate common area units
	commonUnits := totalUnits - totalIndividualUnits
	if commonUnits < 0 {
		return errors.New("individual consumption exceeds total units")
	}

	// Calculate cost per unit
	costPerUnit := totalAmount / totalUnits

	// Cost for individual usage
	individualPoolCost := totalIndividualUnits * costPerUnit

	// Cost for common area
	commonPoolCost := commonUnits * costPerUnit

	// Get all users and groups for weight calculation
	users, err := s.getAllActiveUsers(ctx)
	if err != nil {
		return err
	}

	groups, err := s.getAllGroups(ctx)
	if err != nil {
		return err
	}

	// Calculate weights
	weights := s.calculateWeights(users, groups, req.Weights)
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}

	// Create allocations for each user
	allocations := []interface{}{}
	for userID, units := range userUnits {
		// Personal usage cost
		personalCost := (units / totalIndividualUnits) * individualPoolCost

		// Common area share
		weight := weights[userID]
		commonShare := (weight / totalWeight) * commonPoolCost

		totalCost := utils.RoundPLN(personalCost + commonShare)
		roundedUnits := utils.RoundUnits(units)

		costDec, _ := utils.DecimalFromFloat(totalCost)
		unitsDec, _ := utils.DecimalFromFloat(roundedUnits)

		allocation := models.Allocation{
			ID:          primitive.NewObjectID(),
			BillID:      bill.ID,
			SubjectType: "user",
			SubjectID:   userID,
			AmountPLN:   costDec,
			Units:       unitsDec,
			Method:      req.Strategy,
		}
		allocations = append(allocations, allocation)
	}

	if len(allocations) > 0 {
		_, err = s.db.Collection("allocations").InsertMany(ctx, allocations)
		if err != nil {
			return fmt.Errorf("failed to create allocations: %w", err)
		}
	}

	return nil
}

// allocateGas implements gas allocation (equal split by default)
func (s *BillService) allocateGas(ctx context.Context, bill *models.Bill, req AllocateRequest) error {
	users, err := s.getAllActiveUsers(ctx)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		return errors.New("no active users to allocate to")
	}

	totalAmount, _ := utils.DecimalToFloat(bill.TotalAmountPLN)
	perUser := utils.RoundPLN(totalAmount / float64(len(users)))

	allocations := []interface{}{}
	for _, user := range users {
		costDec, _ := utils.DecimalFromFloat(perUser)
		unitsDec, _ := utils.DecimalFromFloat(0)

		allocation := models.Allocation{
			ID:          primitive.NewObjectID(),
			BillID:      bill.ID,
			SubjectType: "user",
			SubjectID:   user.ID,
			AmountPLN:   costDec,
			Units:       unitsDec,
			Method:      "equal",
		}
		allocations = append(allocations, allocation)
	}

	_, err = s.db.Collection("allocations").InsertMany(ctx, allocations)
	if err != nil {
		return fmt.Errorf("failed to create allocations: %w", err)
	}

	return nil
}

// allocateInternet implements internet allocation (equal split)
func (s *BillService) allocateInternet(ctx context.Context, bill *models.Bill) error {
	return s.allocateGas(ctx, bill, AllocateRequest{Strategy: "equal"})
}

// allocateInne implements "inne" (other) allocation (equal split by default)
func (s *BillService) allocateInne(ctx context.Context, bill *models.Bill, req AllocateRequest) error {
	// For "inne" type, use equal split if no strategy specified
	if req.Strategy == "" {
		req.Strategy = "equal"
	}
	return s.allocateGas(ctx, bill, req)
}

// PostBill changes bill status to posted (freezes allocations)
func (s *BillService) PostBill(ctx context.Context, billID primitive.ObjectID) error {
	return s.updateBillStatus(ctx, billID, "draft", "posted")
}

// CloseBill changes bill status to closed (immutable)
func (s *BillService) CloseBill(ctx context.Context, billID primitive.ObjectID) error {
	return s.updateBillStatus(ctx, billID, "posted", "closed")
}

func (s *BillService) updateBillStatus(ctx context.Context, billID primitive.ObjectID, fromStatus, toStatus string) error {
	result, err := s.db.Collection("bills").UpdateOne(
		ctx,
		bson.M{"_id": billID, "status": fromStatus},
		bson.M{"$set": bson.M{"status": toStatus}},
	)
	if err != nil {
		return fmt.Errorf("failed to update bill status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("bill not found or not in %s status", fromStatus)
	}

	return nil
}

// Helper functions
func (s *BillService) getAllActiveUsers(ctx context.Context) ([]models.User, error) {
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

func (s *BillService) getAllGroups(ctx context.Context) ([]models.Group, error) {
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

func (s *BillService) calculateWeights(users []models.User, groups []models.Group, customWeights map[string]float64) map[primitive.ObjectID]float64 {
	groupWeights := make(map[primitive.ObjectID]float64)
	for _, g := range groups {
		groupWeights[g.ID] = g.Weight
	}

	weights := make(map[primitive.ObjectID]float64)
	for _, u := range users {
		if customWeights != nil {
			if w, ok := customWeights[u.ID.Hex()]; ok {
				weights[u.ID] = w
				continue
			}
		}

		// Use group weight if user belongs to a group
		if u.GroupID != nil {
			if gw, ok := groupWeights[*u.GroupID]; ok {
				weights[u.ID] = gw
				continue
			}
		}

		// Default weight
		weights[u.ID] = 1.0
	}

	return weights
}

// DeleteBill deletes a bill and all associated data
func (s *BillService) DeleteBill(ctx context.Context, billID primitive.ObjectID) error {
	// Delete all consumptions
	_, err := s.db.Collection("consumptions").DeleteMany(ctx, bson.M{"bill_id": billID})
	if err != nil {
		return fmt.Errorf("failed to delete consumptions: %w", err)
	}

	// Delete all allocations
	_, err = s.db.Collection("allocations").DeleteMany(ctx, bson.M{"bill_id": billID})
	if err != nil {
		return fmt.Errorf("failed to delete allocations: %w", err)
	}

	// Delete all payments
	_, err = s.db.Collection("payments").DeleteMany(ctx, bson.M{"bill_id": billID})
	if err != nil {
		return fmt.Errorf("failed to delete payments: %w", err)
	}

	// Delete the bill
	result, err := s.db.Collection("bills").DeleteOne(ctx, bson.M{"_id": billID})
	if err != nil {
		return fmt.Errorf("failed to delete bill: %w", err)
	}

	if result.DeletedCount == 0 {
		return errors.New("bill not found")
	}

	return nil
}