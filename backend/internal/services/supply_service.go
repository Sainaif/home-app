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

type SupplyService struct {
	db *mongo.Database
}

func NewSupplyService(db *mongo.Database) *SupplyService {
	return &SupplyService{db: db}
}

// ========== Settings Methods ==========

// GetSettings retrieves the supply settings (creates default if not exists)
func (s *SupplyService) GetSettings(ctx context.Context) (*models.SupplySettings, error) {
	var settings models.SupplySettings
	err := s.db.Collection("supply_settings").FindOne(ctx, bson.M{}).Decode(&settings)

	if err == mongo.ErrNoDocuments {
		// Create default settings
		defaultContribution, _ := utils.DecimalFromFloat(10.0) // 10 PLN per person per week
		zeroBudget, _ := utils.DecimalFromFloat(0.0)

		settings = models.SupplySettings{
			ID:                    primitive.NewObjectID(),
			WeeklyContributionPLN: defaultContribution,
			ContributionDay:       "monday",
			CurrentBudgetPLN:      zeroBudget,
			LastContributionAt:    time.Now(),
			IsActive:              true,
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
		}

		_, err = s.db.Collection("supply_settings").InsertOne(ctx, settings)
		if err != nil {
			return nil, fmt.Errorf("failed to create default settings: %w", err)
		}

		return &settings, nil
	}

	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &settings, nil
}

// UpdateSettings updates supply settings (ADMIN only)
func (s *SupplyService) UpdateSettings(ctx context.Context, weeklyContribution float64, contributionDay string) error {
	if weeklyContribution <= 0 {
		return errors.New("weekly contribution must be positive")
	}

	validDays := map[string]bool{
		"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
		"friday": true, "saturday": true, "sunday": true,
	}
	if !validDays[contributionDay] {
		return errors.New("invalid contribution day")
	}

	contributionDec, err := utils.DecimalFromFloat(weeklyContribution)
	if err != nil {
		return fmt.Errorf("invalid contribution amount: %w", err)
	}

	update := bson.M{
		"$set": bson.M{
			"weekly_contribution_pln": contributionDec,
			"contribution_day":        contributionDay,
			"updated_at":              time.Now(),
		},
	}

	result, err := s.db.Collection("supply_settings").UpdateOne(ctx, bson.M{}, update)
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("settings not found")
	}

	return nil
}

// AdjustBudget manually adjusts the budget (ADMIN only)
func (s *SupplyService) AdjustBudget(ctx context.Context, adjustment float64, notes string) error {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}

	currentBudget, _ := utils.DecimalToFloat(settings.CurrentBudgetPLN)
	newBudget, _ := utils.DecimalFromFloat(currentBudget + adjustment)

	update := bson.M{
		"$set": bson.M{
			"current_budget_pln": newBudget,
			"updated_at":         time.Now(),
		},
	}

	_, err = s.db.Collection("supply_settings").UpdateOne(ctx, bson.M{}, update)
	if err != nil {
		return fmt.Errorf("failed to adjust budget: %w", err)
	}

	return nil
}

// ========== Item Methods ==========

// GetItems retrieves supply items with optional filters and sorting
func (s *SupplyService) GetItems(ctx context.Context, filter, sort *string) ([]models.SupplyItem, error) {
	queryFilter := bson.M{}

	// Apply filters
	if filter != nil && *filter != "" {
		switch *filter {
		case "low_stock":
			queryFilter["$expr"] = bson.M{"$lte": []interface{}{"$current_quantity", "$min_quantity"}}
		case "needs_refund":
			queryFilter["needs_refund"] = true
		}
	}

	// Default sort by priority desc, then name asc
	sortOrder := bson.D{{Key: "priority", Value: -1}, {Key: "name", Value: 1}}

	if sort != nil && *sort != "" {
		switch *sort {
		case "quantity_asc":
			sortOrder = bson.D{{Key: "current_quantity", Value: 1}, {Key: "name", Value: 1}}
		case "quantity_desc":
			sortOrder = bson.D{{Key: "current_quantity", Value: -1}, {Key: "name", Value: 1}}
		case "name":
			sortOrder = bson.D{{Key: "name", Value: 1}}
		case "recently_restocked":
			sortOrder = bson.D{{Key: "last_restocked_at", Value: -1}}
		}
	}

	opts := options.Find().SetSort(sortOrder)
	cursor, err := s.db.Collection("supply_items").Find(ctx, queryFilter, opts)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var items []models.SupplyItem
	if err := cursor.All(ctx, &items); err != nil {
		return nil, fmt.Errorf("failed to decode items: %w", err)
	}

	return items, nil
}

// CreateItem adds a new supply item with initial inventory
func (s *SupplyService) CreateItem(ctx context.Context, userID primitive.ObjectID, name, category string, currentQuantity, minQuantity int, unit string, priority int, notes *string) (*models.SupplyItem, error) {
	validCategories := map[string]bool{
		"groceries": true, "cleaning": true, "toiletries": true, "other": true,
	}
	if !validCategories[category] {
		return nil, errors.New("invalid category")
	}

	validUnits := map[string]bool{
		"pcs": true, "kg": true, "L": true, "bottles": true, "boxes": true,
		"rolls": true, "bags": true, "jars": true, "cans": true,
	}
	if !validUnits[unit] {
		return nil, errors.New("invalid unit")
	}

	if priority < 1 || priority > 5 {
		return nil, errors.New("priority must be between 1 and 5")
	}

	if name == "" {
		return nil, errors.New("item name is required")
	}

	if currentQuantity < 0 {
		return nil, errors.New("current quantity cannot be negative")
	}

	if minQuantity < 0 {
		return nil, errors.New("min quantity cannot be negative")
	}

	item := models.SupplyItem{
		ID:              primitive.NewObjectID(),
		Name:            name,
		Category:        category,
		CurrentQuantity: currentQuantity,
		MinQuantity:     minQuantity,
		Unit:            unit,
		Priority:        priority,
		AddedByUserID:   userID,
		AddedAt:         time.Now(),
		NeedsRefund:     false,
		Notes:           notes,
	}

	_, err := s.db.Collection("supply_items").InsertOne(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	return &item, nil
}

// UpdateItem updates item details
func (s *SupplyService) UpdateItem(ctx context.Context, itemID primitive.ObjectID, name *string, category *string, minQuantity *int, unit *string, priority *int, notes *string) error {
	update := bson.M{"$set": bson.M{}}

	if name != nil && *name != "" {
		update["$set"].(bson.M)["name"] = *name
	}

	if category != nil {
		validCategories := map[string]bool{
			"groceries": true, "cleaning": true, "toiletries": true, "other": true,
		}
		if !validCategories[*category] {
			return errors.New("invalid category")
		}
		update["$set"].(bson.M)["category"] = *category
	}

	if unit != nil {
		validUnits := map[string]bool{
			"pcs": true, "kg": true, "L": true, "bottles": true, "boxes": true,
			"rolls": true, "bags": true, "jars": true, "cans": true,
		}
		if !validUnits[*unit] {
			return errors.New("invalid unit")
		}
		update["$set"].(bson.M)["unit"] = *unit
	}

	if minQuantity != nil {
		if *minQuantity < 0 {
			return errors.New("min quantity cannot be negative")
		}
		update["$set"].(bson.M)["min_quantity"] = *minQuantity
	}

	if priority != nil {
		if *priority < 1 || *priority > 5 {
			return errors.New("priority must be between 1 and 5")
		}
		update["$set"].(bson.M)["priority"] = *priority
	}

	if notes != nil {
		update["$set"].(bson.M)["notes"] = *notes
	}

	if len(update["$set"].(bson.M)) == 0 {
		return errors.New("no fields to update")
	}

	result, err := s.db.Collection("supply_items").UpdateOne(ctx, bson.M{"_id": itemID}, update)
	if err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("item not found")
	}

	return nil
}

// RestockItem increases quantity and optionally records amount spent for refund
func (s *SupplyService) RestockItem(ctx context.Context, itemID, userID primitive.ObjectID, quantityToAdd int, amountPLN *float64, needsRefund bool) error {
	if quantityToAdd <= 0 {
		return errors.New("quantity to add must be positive")
	}

	now := time.Now()
	update := bson.M{
		"$inc": bson.M{"current_quantity": quantityToAdd},
		"$set": bson.M{
			"last_restocked_at":        now,
			"last_restocked_by_user_id": userID,
			"needs_refund":             needsRefund,
		},
	}

	if amountPLN != nil {
		if *amountPLN < 0 {
			return errors.New("amount cannot be negative")
		}
		amountDec, err := utils.DecimalFromFloat(*amountPLN)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
		update["$set"].(bson.M)["last_restock_amount_pln"] = amountDec
	}

	result, err := s.db.Collection("supply_items").UpdateOne(ctx, bson.M{"_id": itemID}, update)
	if err != nil {
		return fmt.Errorf("failed to restock item: %w", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("item not found")
	}

	return nil
}

// ConsumeItem reduces quantity (for use/consumption)
func (s *SupplyService) ConsumeItem(ctx context.Context, itemID primitive.ObjectID, quantityToSubtract int) error {
	if quantityToSubtract <= 0 {
		return errors.New("quantity to subtract must be positive")
	}

	// Check current quantity first
	var item models.SupplyItem
	err := s.db.Collection("supply_items").FindOne(ctx, bson.M{"_id": itemID}).Decode(&item)
	if err == mongo.ErrNoDocuments {
		return errors.New("item not found")
	}
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	if item.CurrentQuantity < quantityToSubtract {
		return errors.New("insufficient quantity")
	}

	update := bson.M{
		"$inc": bson.M{"current_quantity": -quantityToSubtract},
	}

	_, err = s.db.Collection("supply_items").UpdateOne(ctx, bson.M{"_id": itemID}, update)
	if err != nil {
		return fmt.Errorf("failed to consume item: %w", err)
	}

	return nil
}

// SetQuantity directly sets the quantity (for corrections)
func (s *SupplyService) SetQuantity(ctx context.Context, itemID primitive.ObjectID, newQuantity int) error {
	if newQuantity < 0 {
		return errors.New("quantity cannot be negative")
	}

	update := bson.M{
		"$set": bson.M{"current_quantity": newQuantity},
	}

	result, err := s.db.Collection("supply_items").UpdateOne(ctx, bson.M{"_id": itemID}, update)
	if err != nil {
		return fmt.Errorf("failed to set quantity: %w", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("item not found")
	}

	return nil
}

// MarkAsRefunded marks an item as refunded and deducts from shared budget
func (s *SupplyService) MarkAsRefunded(ctx context.Context, itemID primitive.ObjectID) error {
	// Get item to check refund details
	var item models.SupplyItem
	err := s.db.Collection("supply_items").FindOne(ctx, bson.M{"_id": itemID}).Decode(&item)
	if err == mongo.ErrNoDocuments {
		return errors.New("item not found")
	}
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	if !item.NeedsRefund {
		return errors.New("item does not need refund")
	}

	if item.LastRestockAmountPLN == nil {
		return errors.New("no refund amount recorded")
	}

	// Deduct from budget
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}

	amountToRefund, _ := utils.DecimalToFloat(*item.LastRestockAmountPLN)
	currentBudget, _ := utils.DecimalToFloat(settings.CurrentBudgetPLN)

	if currentBudget < amountToRefund {
		return fmt.Errorf("insufficient budget: have %.2f PLN, need %.2f PLN", currentBudget, amountToRefund)
	}

	newBudget, _ := utils.DecimalFromFloat(currentBudget - amountToRefund)

	// Update item
	itemUpdate := bson.M{
		"$set": bson.M{"needs_refund": false},
	}

	_, err = s.db.Collection("supply_items").UpdateOne(ctx, bson.M{"_id": itemID}, itemUpdate)
	if err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	// Update budget
	_, err = s.db.Collection("supply_settings").UpdateOne(
		ctx,
		bson.M{},
		bson.M{"$set": bson.M{"current_budget_pln": newBudget, "updated_at": time.Now()}},
	)
	if err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
	}

	return nil
}

// DeleteItem deletes an item (ADMIN or creator only - enforced at handler level)
func (s *SupplyService) DeleteItem(ctx context.Context, itemID primitive.ObjectID) error {
	result, err := s.db.Collection("supply_items").DeleteOne(ctx, bson.M{"_id": itemID})
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	if result.DeletedCount == 0 {
		return errors.New("item not found")
	}

	return nil
}

// ========== Contribution Methods ==========

// GetContributions retrieves contributions with optional filters
func (s *SupplyService) GetContributions(ctx context.Context, userID *primitive.ObjectID, fromDate *time.Time) ([]models.SupplyContribution, error) {
	filter := bson.M{}

	if userID != nil {
		filter["user_id"] = *userID
	}

	if fromDate != nil {
		filter["period_start"] = bson.M{"$gte": *fromDate}
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := s.db.Collection("supply_contributions").Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer cursor.Close(ctx)

	var contributions []models.SupplyContribution
	if err := cursor.All(ctx, &contributions); err != nil {
		return nil, fmt.Errorf("failed to decode contributions: %w", err)
	}

	return contributions, nil
}

// CreateManualContribution adds a manual contribution
func (s *SupplyService) CreateManualContribution(ctx context.Context, userID primitive.ObjectID, amountPLN float64, notes *string) error {
	if amountPLN <= 0 {
		return errors.New("amount must be positive")
	}

	amountDec, err := utils.DecimalFromFloat(amountPLN)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	now := time.Now()
	contribution := models.SupplyContribution{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		AmountPLN:   amountDec,
		PeriodStart: now,
		PeriodEnd:   now,
		Type:        "manual",
		Notes:       notes,
		CreatedAt:   now,
	}

	_, err = s.db.Collection("supply_contributions").InsertOne(ctx, contribution)
	if err != nil {
		return fmt.Errorf("failed to create contribution: %w", err)
	}

	// Add to budget
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}

	currentBudget, _ := utils.DecimalToFloat(settings.CurrentBudgetPLN)
	newBudget, _ := utils.DecimalFromFloat(currentBudget + amountPLN)

	_, err = s.db.Collection("supply_settings").UpdateOne(
		ctx,
		bson.M{},
		bson.M{"$set": bson.M{"current_budget_pln": newBudget, "updated_at": time.Now()}},
	)
	if err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
	}

	return nil
}

// ProcessWeeklyContributions creates automatic weekly contributions for all active users
func (s *SupplyService) ProcessWeeklyContributions(ctx context.Context) error {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}

	if !settings.IsActive {
		return errors.New("supply system is not active")
	}

	// Get all active users
	cursor, err := s.db.Collection("users").Find(ctx, bson.M{"is_active": true})
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return fmt.Errorf("failed to decode users: %w", err)
	}

	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	weekEnd := weekStart.AddDate(0, 0, 6)

	totalContributed := 0.0

	// Create contribution for each active user
	for _, user := range users {
		contribution := models.SupplyContribution{
			ID:          primitive.NewObjectID(),
			UserID:      user.ID,
			AmountPLN:   settings.WeeklyContributionPLN,
			PeriodStart: weekStart,
			PeriodEnd:   weekEnd,
			Type:        "weekly_auto",
			CreatedAt:   now,
		}

		_, err := s.db.Collection("supply_contributions").InsertOne(ctx, contribution)
		if err != nil {
			return fmt.Errorf("failed to create contribution for user %s: %w", user.Email, err)
		}

		amount, _ := utils.DecimalToFloat(settings.WeeklyContributionPLN)
		totalContributed += amount
	}

	// Update budget
	currentBudget, _ := utils.DecimalToFloat(settings.CurrentBudgetPLN)
	newBudget, _ := utils.DecimalFromFloat(currentBudget + totalContributed)

	_, err = s.db.Collection("supply_settings").UpdateOne(
		ctx,
		bson.M{},
		bson.M{"$set": bson.M{
			"current_budget_pln":   newBudget,
			"last_contribution_at": now,
			"updated_at":           now,
		}},
	)
	if err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
	}

	return nil
}

// GetStats returns spending statistics
func (s *SupplyService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	// Total spent by category (from restocks that needed refund and were refunded)
	categoryPipeline := []bson.M{
		{"$match": bson.M{
			"last_restock_amount_pln": bson.M{"$exists": true, "$ne": nil},
		}},
		{"$group": bson.M{
			"_id":        "$category",
			"totalSpent": bson.M{"$sum": bson.M{"$toDouble": "$last_restock_amount_pln"}},
			"count":      bson.M{"$sum": 1},
		}},
		{"$sort": bson.M{"totalSpent": -1}},
	}

	categoryCursor, err := s.db.Collection("supply_items").Aggregate(ctx, categoryPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate category stats: %w", err)
	}
	defer categoryCursor.Close(ctx)

	var categoryStats []map[string]interface{}
	if err := categoryCursor.All(ctx, &categoryStats); err != nil {
		return nil, fmt.Errorf("failed to decode category stats: %w", err)
	}

	// Spending by user (from restocks)
	userPipeline := []bson.M{
		{"$match": bson.M{
			"last_restocked_by_user_id": bson.M{"$exists": true, "$ne": nil},
			"last_restock_amount_pln":   bson.M{"$exists": true, "$ne": nil},
		}},
		{"$group": bson.M{
			"_id":        "$last_restocked_by_user_id",
			"totalSpent": bson.M{"$sum": bson.M{"$toDouble": "$last_restock_amount_pln"}},
			"count":      bson.M{"$sum": 1},
		}},
		{"$sort": bson.M{"totalSpent": -1}},
	}

	userCursor, err := s.db.Collection("supply_items").Aggregate(ctx, userPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate user stats: %w", err)
	}
	defer userCursor.Close(ctx)

	var userStats []map[string]interface{}
	if err := userCursor.All(ctx, &userStats); err != nil {
		return nil, fmt.Errorf("failed to decode user stats: %w", err)
	}

	// Recent contributions (last 10)
	contributions, err := s.GetContributions(ctx, nil, nil)
	if err != nil {
		return nil, err
	}

	recentContributions := contributions
	if len(recentContributions) > 10 {
		recentContributions = recentContributions[:10]
	}

	return map[string]interface{}{
		"byCategory":           categoryStats,
		"byUser":               userStats,
		"recentContributions":  recentContributions,
	}, nil
}
