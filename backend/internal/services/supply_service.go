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

// GetItems retrieves supply items with optional status filter
func (s *SupplyService) GetItems(ctx context.Context, status *string) ([]models.SupplyItem, error) {
	filter := bson.M{}
	if status != nil && *status != "" {
		filter["status"] = *status
	}

	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: -1}, {Key: "added_at", Value: -1}})
	cursor, err := s.db.Collection("supply_items").Find(ctx, filter, opts)
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

// CreateItem adds a new item to the shopping list
func (s *SupplyService) CreateItem(ctx context.Context, userID primitive.ObjectID, name, category string, quantity *string, priority int) (*models.SupplyItem, error) {
	validCategories := map[string]bool{
		"groceries": true, "cleaning": true, "toiletries": true, "other": true,
	}
	if !validCategories[category] {
		return nil, errors.New("invalid category")
	}

	if priority < 1 || priority > 5 {
		return nil, errors.New("priority must be between 1 and 5")
	}

	if name == "" {
		return nil, errors.New("item name is required")
	}

	item := models.SupplyItem{
		ID:            primitive.NewObjectID(),
		Name:          name,
		Category:      category,
		Status:        "needed",
		Quantity:      quantity,
		Priority:      priority,
		AddedByUserID: userID,
		AddedAt:       time.Now(),
	}

	_, err := s.db.Collection("supply_items").InsertOne(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	return &item, nil
}

// UpdateItem updates item details
func (s *SupplyService) UpdateItem(ctx context.Context, itemID primitive.ObjectID, name *string, category *string, quantity *string, priority *int) error {
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

	if quantity != nil {
		update["$set"].(bson.M)["quantity"] = *quantity
	}

	if priority != nil {
		if *priority < 1 || *priority > 5 {
			return errors.New("priority must be between 1 and 5")
		}
		update["$set"].(bson.M)["priority"] = *priority
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

// MarkAsBought marks an item as bought and deducts from budget
func (s *SupplyService) MarkAsBought(ctx context.Context, itemID, userID primitive.ObjectID, amountPLN float64, notes *string) error {
	if amountPLN <= 0 {
		return errors.New("amount must be positive")
	}

	amountDec, err := utils.DecimalFromFloat(amountPLN)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	// Get current settings
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}

	// Deduct from budget
	currentBudget, _ := utils.DecimalToFloat(settings.CurrentBudgetPLN)
	newBudget, _ := utils.DecimalFromFloat(currentBudget - amountPLN)

	now := time.Now()

	// Update item
	itemUpdate := bson.M{
		"$set": bson.M{
			"status":           "bought",
			"bought_by_user_id": userID,
			"bought_at":        now,
			"amount_pln":       amountDec,
		},
	}
	if notes != nil {
		itemUpdate["$set"].(bson.M)["notes"] = *notes
	}

	result, err := s.db.Collection("supply_items").UpdateOne(
		ctx,
		bson.M{"_id": itemID, "status": "needed"},
		itemUpdate,
	)
	if err != nil {
		return fmt.Errorf("failed to mark item as bought: %w", err)
	}

	if result.MatchedCount == 0 {
		return errors.New("item not found or already bought")
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
	// Total spent all time
	pipeline := []bson.M{
		{"$match": bson.M{"status": "bought"}},
		{"$group": bson.M{
			"_id": nil,
			"totalSpent": bson.M{"$sum": bson.M{"$toDouble": "$amount_pln"}},
			"itemCount":  bson.M{"$sum": 1},
		}},
	}

	cursor, err := s.db.Collection("supply_items").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate stats: %w", err)
	}
	defer cursor.Close(ctx)

	var totalResult []struct {
		TotalSpent float64 `bson:"totalSpent"`
		ItemCount  int     `bson:"itemCount"`
	}
	if err := cursor.All(ctx, &totalResult); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	totalSpent := 0.0
	itemCount := 0
	if len(totalResult) > 0 {
		totalSpent = totalResult[0].TotalSpent
		itemCount = totalResult[0].ItemCount
	}

	// Spending by category
	categoryPipeline := []bson.M{
		{"$match": bson.M{"status": "bought"}},
		{"$group": bson.M{
			"_id":   "$category",
			"total": bson.M{"$sum": bson.M{"$toDouble": "$amount_pln"}},
			"count": bson.M{"$sum": 1},
		}},
		{"$sort": bson.M{"total": -1}},
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

	// Spending by user
	userPipeline := []bson.M{
		{"$match": bson.M{"status": "bought"}},
		{"$group": bson.M{
			"_id":   "$bought_by_user_id",
			"total": bson.M{"$sum": bson.M{"$toDouble": "$amount_pln"}},
			"count": bson.M{"$sum": 1},
		}},
		{"$sort": bson.M{"total": -1}},
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

	return map[string]interface{}{
		"totalSpent":    totalSpent,
		"itemCount":     itemCount,
		"byCategory":    categoryStats,
		"byUser":        userStats,
	}, nil
}
