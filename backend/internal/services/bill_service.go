package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/utils"
)

type BillService struct {
	db                  *mongo.Database
	notificationService *NotificationService
}

func NewBillService(db *mongo.Database, notificationService *NotificationService) *BillService {
	return &BillService{db: db, notificationService: notificationService}
}

type CreateBillRequest struct {
	Type            string     `json:"type"`                     // electricity, gas, internet, inne
	CustomType      *string    `json:"customType,omitempty"`     // required when type is "inne"
	AllocationType  *string    `json:"allocationType,omitempty"` // "simple" or "metered", required when type is "inne"
	PeriodStart     time.Time  `json:"periodStart"`
	PeriodEnd       time.Time  `json:"periodEnd"`
	PaymentDeadline *time.Time `json:"paymentDeadline,omitempty"` // optional payment deadline
	TotalAmountPLN  float64    `json:"totalAmountPLN"`
	TotalUnits      *float64   `json:"totalUnits,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
}

// CreateBill creates a new bill in the database
func (s *BillService) CreateBill(ctx context.Context, req CreateBillRequest, creatorID primitive.ObjectID) (*models.Bill, error) {
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

	// Validate allocationType for "inne" type
	if req.Type == "inne" && (req.AllocationType == nil || (*req.AllocationType != "simple" && *req.AllocationType != "metered")) {
		return nil, errors.New("allocationType must be 'simple' or 'metered' when type is 'inne'")
	}

	// Set default allocation types for standard bill types
	allocationType := req.AllocationType
	if req.Type == "gas" || req.Type == "internet" {
		simpleType := "simple"
		allocationType = &simpleType
	} else if req.Type == "electricity" {
		meteredType := "metered"
		allocationType = &meteredType
	}

	if req.PeriodEnd.Before(req.PeriodStart) {
		return nil, errors.New("period end must be after period start")
	}

	amountDec, err := utils.DecimalFromFloat(req.TotalAmountPLN)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	bill := models.Bill{
		ID:              primitive.NewObjectID(),
		Type:            req.Type,
		CustomType:      req.CustomType,
		AllocationType:  allocationType,
		PeriodStart:     req.PeriodStart,
		PeriodEnd:       req.PeriodEnd,
		PaymentDeadline: req.PaymentDeadline,
		TotalAmountPLN:  amountDec,
		Notes:           req.Notes,
		Status:          "draft",
		CreatedAt:       time.Now(),
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

	// Create a notification for all users except the creator
	users, err := s.getAllActiveUsers(ctx)
	if err != nil {
		log.Printf("failed to get all active users: %v", err)
	} else {
		for _, user := range users {
			if user.ID != creatorID {
				now := time.Now()
				notification := &models.Notification{
					UserID:       &user.ID,
					Channel:      "app",
					TemplateID:   "bill",
					ScheduledFor: now,
					SentAt:       &now,
					Status:       "sent",
				}
				s.notificationService.CreateNotification(ctx, notification)
			}
		}
	}

	return &bill, nil
}

// GetBills gets all bills, can filter by type and dates
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

// GetBill gets a single bill by ID
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

// PostBill marks bill as posted (freezes allocations)
func (s *BillService) PostBill(ctx context.Context, billID primitive.ObjectID) error {
	return s.updateBillStatus(ctx, billID, "draft", "posted")
}

// CloseBill marks bill as closed (no more changes)
func (s *BillService) CloseBill(ctx context.Context, billID primitive.ObjectID) error {
	return s.updateBillStatus(ctx, billID, "posted", "closed")
}

// ReopenBill reverts a bill back to draft or posted status
func (s *BillService) ReopenBill(ctx context.Context, billID primitive.ObjectID, userID primitive.ObjectID, targetStatus, reason string) error {
	// Validate target status
	if targetStatus != "draft" && targetStatus != "posted" {
		return errors.New("target status must be 'draft' or 'posted'")
	}

	if reason == "" {
		return errors.New("reopen reason is required")
	}

	// Get current bill
	bill, err := s.GetBill(ctx, billID)
	if err != nil {
		return err
	}

	// Validate state transition
	if bill.Status == "draft" {
		return errors.New("bill is already in draft status")
	}

	if bill.Status == "posted" && targetStatus == "posted" {
		return errors.New("bill is already in posted status")
	}

	now := time.Now()

	// Update bill status and reopen metadata
	update := bson.M{
		"$set": bson.M{
			"status":        targetStatus,
			"reopened_at":   now,
			"reopen_reason": reason,
			"reopened_by":   userID,
		},
	}

	result, err := s.db.Collection("bills").UpdateOne(
		ctx,
		bson.M{"_id": billID},
		update,
	)
	if err != nil {
		return fmt.Errorf("failed to reopen bill: %w", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("bill not found")
	}

	return nil
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

	// Delete all payments
	_, err = s.db.Collection("payments").DeleteMany(ctx, bson.M{"bill_id": billID})
	if err != nil {
		return fmt.Errorf("failed to delete payments: %w", err)
	}

	// Delete all allocations
	_, err = s.db.Collection("allocations").DeleteMany(ctx, bson.M{"bill_id": billID})
	if err != nil {
		return fmt.Errorf("failed to delete allocations: %w", err)
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

// PaymentStatusEntry represents payment status for a user/group
type PaymentStatusEntry struct {
	SubjectID    primitive.ObjectID   `json:"subjectId"`
	SubjectType  string               `json:"subjectType"` // "user" or "group"
	SubjectName  string               `json:"subjectName"`
	AllocatedPLN primitive.Decimal128 `json:"allocatedPLN"`
	PaidPLN      primitive.Decimal128 `json:"paidPLN"`
	RemainingPLN primitive.Decimal128 `json:"remainingPLN"`
	IsPaid       bool                 `json:"isPaid"`
}

// GetBillPaymentStatus returns detailed payment status showing who paid and who hasn't
func (s *BillService) GetBillPaymentStatus(ctx context.Context, billID primitive.ObjectID) ([]PaymentStatusEntry, error) {
	// Get all allocations for this bill
	allocCursor, err := s.db.Collection("allocations").Find(ctx, bson.M{"bill_id": billID})
	if err != nil {
		return nil, err
	}
	defer allocCursor.Close(ctx)

	var allocations []struct {
		SubjectID    primitive.ObjectID   `bson:"subject_id"`
		SubjectType  string               `bson:"subject_type"`
		AllocatedPLN primitive.Decimal128 `bson:"allocated_pln"`
	}
	if err := allocCursor.All(ctx, &allocations); err != nil {
		return nil, err
	}

	// Get all payments for this bill
	paymentCursor, err := s.db.Collection("payments").Find(ctx, bson.M{"bill_id": billID})
	if err != nil {
		return nil, err
	}
	defer paymentCursor.Close(ctx)

	var payments []struct {
		PayerUserID primitive.ObjectID   `bson:"payer_user_id"`
		AmountPLN   primitive.Decimal128 `bson:"amount_pln"`
	}
	if err := paymentCursor.All(ctx, &payments); err != nil {
		return nil, err
	}

	// Build payment map
	paymentMap := make(map[primitive.ObjectID]primitive.Decimal128)
	for _, payment := range payments {
		existing := paymentMap[payment.PayerUserID]
		existingFloat, _ := utils.DecimalToFloat(existing)
		paymentFloat, _ := utils.DecimalToFloat(payment.AmountPLN)
		totalFloat := existingFloat + paymentFloat
		totalDecimal, _ := utils.DecimalFromFloat(totalFloat)
		paymentMap[payment.PayerUserID] = totalDecimal
	}

	// Build status entries
	var statusEntries []PaymentStatusEntry
	for _, alloc := range allocations {
		var subjectName string

		// Get subject name
		if alloc.SubjectType == "user" {
			var user models.User
			err := s.db.Collection("users").FindOne(ctx, bson.M{"_id": alloc.SubjectID}).Decode(&user)
			if err == nil {
				subjectName = user.Name
			} else {
				subjectName = "Unknown User"
			}

			// Get paid amount for this user
			paidPLN := paymentMap[alloc.SubjectID]
			paidFloat, _ := utils.DecimalToFloat(paidPLN)
			allocFloat, _ := utils.DecimalToFloat(alloc.AllocatedPLN)
			remainingFloat := allocFloat - paidFloat
			remainingPLN, _ := utils.DecimalFromFloat(remainingFloat)

			statusEntries = append(statusEntries, PaymentStatusEntry{
				SubjectID:    alloc.SubjectID,
				SubjectType:  alloc.SubjectType,
				SubjectName:  subjectName,
				AllocatedPLN: alloc.AllocatedPLN,
				PaidPLN:      paidPLN,
				RemainingPLN: remainingPLN,
				IsPaid:       paidFloat >= allocFloat,
			})
		} else if alloc.SubjectType == "group" {
			var group models.Group
			err := s.db.Collection("groups").FindOne(ctx, bson.M{"_id": alloc.SubjectID}).Decode(&group)
			if err == nil {
				subjectName = group.Name
			} else {
				subjectName = "Unknown Group"
			}

			// Get all users in this group
			userCursor, err := s.db.Collection("users").Find(ctx, bson.M{"group_id": alloc.SubjectID})
			if err != nil {
				continue
			}

			var groupUsers []models.User
			if err := userCursor.All(ctx, &groupUsers); err != nil {
				userCursor.Close(ctx)
				continue
			}
			userCursor.Close(ctx)

			// Calculate total paid by all group members
			totalPaidFloat := 0.0
			for _, user := range groupUsers {
				paidPLN := paymentMap[user.ID]
				paidFloat, _ := utils.DecimalToFloat(paidPLN)
				totalPaidFloat += paidFloat
			}

			totalPaidDecimal, _ := utils.DecimalFromFloat(totalPaidFloat)
			allocFloat, _ := utils.DecimalToFloat(alloc.AllocatedPLN)
			remainingFloat := allocFloat - totalPaidFloat
			remainingPLN, _ := utils.DecimalFromFloat(remainingFloat)

			statusEntries = append(statusEntries, PaymentStatusEntry{
				SubjectID:    alloc.SubjectID,
				SubjectType:  alloc.SubjectType,
				SubjectName:  subjectName,
				AllocatedPLN: alloc.AllocatedPLN,
				PaidPLN:      totalPaidDecimal,
				RemainingPLN: remainingPLN,
				IsPaid:       totalPaidFloat >= allocFloat,
			})
		}
	}

	return statusEntries, nil
}
