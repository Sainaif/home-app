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

type SupplyService struct {
	supplySettings      repository.SupplySettingsRepository
	supplyItems         repository.SupplyItemRepository
	supplyContributions repository.SupplyContributionRepository
	users               repository.UserRepository
	notificationService *NotificationService
}

func NewSupplyService(
	supplySettings repository.SupplySettingsRepository,
	supplyItems repository.SupplyItemRepository,
	supplyContributions repository.SupplyContributionRepository,
	users repository.UserRepository,
	notificationService *NotificationService,
) *SupplyService {
	return &SupplyService{
		supplySettings:      supplySettings,
		supplyItems:         supplyItems,
		supplyContributions: supplyContributions,
		users:               users,
		notificationService: notificationService,
	}
}

// ========== Settings Methods ==========

// GetSettings retrieves the supply settings (creates default if not exists)
func (s *SupplyService) GetSettings(ctx context.Context) (*models.SupplySettings, error) {
	settings, err := s.supplySettings.Get(ctx)
	if err != nil {
		// Create default settings
		settings = &models.SupplySettings{
			ID:                    "singleton",
			WeeklyContributionPLN: utils.FloatToDecimalString(10.0), // 10 PLN per person per week
			ContributionDay:       "monday",
			CurrentBudgetPLN:      utils.FloatToDecimalString(0.0),
			LastContributionAt:    time.Now(),
			IsActive:              true,
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
		}

		if err := s.supplySettings.Upsert(ctx, settings); err != nil {
			return nil, fmt.Errorf("failed to create default settings: %w", err)
		}

		return settings, nil
	}

	return settings, nil
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

	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}

	settings.WeeklyContributionPLN = utils.FloatToDecimalString(weeklyContribution)
	settings.ContributionDay = contributionDay
	settings.UpdatedAt = time.Now()

	if err := s.supplySettings.Upsert(ctx, settings); err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	return nil
}

// AdjustBudget manually adjusts the budget (ADMIN only)
func (s *SupplyService) AdjustBudget(ctx context.Context, adjustment float64, notes string) error {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}

	currentBudget := utils.DecimalStringToFloat(settings.CurrentBudgetPLN)
	newBudget := currentBudget + adjustment
	settings.CurrentBudgetPLN = utils.FloatToDecimalString(newBudget)
	settings.UpdatedAt = time.Now()

	if err := s.supplySettings.Upsert(ctx, settings); err != nil {
		return fmt.Errorf("failed to adjust budget: %w", err)
	}

	return nil
}

// ========== Item Methods ==========

// GetItems retrieves supply items with optional filters and sorting
func (s *SupplyService) GetItems(ctx context.Context, filter, sort *string) ([]models.SupplyItem, error) {
	var items []models.SupplyItem
	var err error

	// Apply filters
	if filter != nil && *filter != "" {
		switch *filter {
		case "low_stock":
			items, err = s.supplyItems.ListLowStock(ctx)
		case "needs_refund":
			// Get all items and filter
			allItems, err := s.supplyItems.List(ctx)
			if err != nil {
				return nil, fmt.Errorf("database error: %w", err)
			}
			for _, item := range allItems {
				if item.NeedsRefund {
					items = append(items, item)
				}
			}
			return items, nil
		default:
			items, err = s.supplyItems.List(ctx)
		}
	} else {
		items, err = s.supplyItems.List(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Note: Sorting is handled at repository level or can be done in-memory if needed
	return items, nil
}

// CreateItem adds a new supply item with initial inventory
func (s *SupplyService) CreateItem(ctx context.Context, userID string, name, category string, currentQuantity, minQuantity int, unit string, priority int, notes *string) (*models.SupplyItem, error) {
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
		ID:              uuid.New().String(),
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

	if err := s.supplyItems.Create(ctx, &item); err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	// Send notifications to all active users about the new supply item
	if s.notificationService != nil {
		users, err := s.users.ListActive(ctx)
		if err == nil {
			now := time.Now()
			for _, user := range users {
				if user.ID != userID { // Don't notify the creator
					notification := &models.Notification{
						ID:           uuid.New().String(),
						UserID:       &user.ID,
						Channel:      "app",
						TemplateID:   "supply",
						ScheduledFor: now,
						SentAt:       &now,
						Status:       "sent",
						Title:        "Nowy artykul zaopatrzeniowy",
						Body:         fmt.Sprintf("Dodano: %s do listy zaopatrzenia", name),
					}
					s.notificationService.CreateNotification(ctx, notification)
				}
			}
		}
	}

	return &item, nil
}

// UpdateItem updates item details
func (s *SupplyService) UpdateItem(ctx context.Context, itemID string, name *string, category *string, minQuantity *int, unit *string, priority *int, notes *string) error {
	item, err := s.supplyItems.GetByID(ctx, itemID)
	if err != nil {
		return errors.New("item not found")
	}

	if name != nil && *name != "" {
		item.Name = *name
	}

	if category != nil {
		validCategories := map[string]bool{
			"groceries": true, "cleaning": true, "toiletries": true, "other": true,
		}
		if !validCategories[*category] {
			return errors.New("invalid category")
		}
		item.Category = *category
	}

	if unit != nil {
		validUnits := map[string]bool{
			"pcs": true, "kg": true, "L": true, "bottles": true, "boxes": true,
			"rolls": true, "bags": true, "jars": true, "cans": true,
		}
		if !validUnits[*unit] {
			return errors.New("invalid unit")
		}
		item.Unit = *unit
	}

	if minQuantity != nil {
		if *minQuantity < 0 {
			return errors.New("min quantity cannot be negative")
		}
		item.MinQuantity = *minQuantity
	}

	if priority != nil {
		if *priority < 1 || *priority > 5 {
			return errors.New("priority must be between 1 and 5")
		}
		item.Priority = *priority
	}

	if notes != nil {
		item.Notes = notes
	}

	if err := s.supplyItems.Update(ctx, item); err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	return nil
}

// RestockItem increases quantity and optionally records amount spent for refund
func (s *SupplyService) RestockItem(ctx context.Context, itemID, userID string, quantityToAdd int, amountPLN *float64, needsRefund bool) error {
	if quantityToAdd <= 0 {
		return errors.New("quantity to add must be positive")
	}

	item, err := s.supplyItems.GetByID(ctx, itemID)
	if err != nil {
		return errors.New("item not found")
	}

	now := time.Now()
	item.CurrentQuantity += quantityToAdd
	item.LastRestockedAt = &now
	item.LastRestockedByUserID = &userID
	item.NeedsRefund = needsRefund

	if amountPLN != nil {
		if *amountPLN < 0 {
			return errors.New("amount cannot be negative")
		}
		amountStr := utils.FloatToDecimalString(*amountPLN)
		item.LastRestockAmountPLN = &amountStr
	}

	if err := s.supplyItems.Update(ctx, item); err != nil {
		return fmt.Errorf("failed to restock item: %w", err)
	}

	return nil
}

// ConsumeItem reduces quantity (for use/consumption)
func (s *SupplyService) ConsumeItem(ctx context.Context, itemID string, quantityToSubtract int) error {
	if quantityToSubtract <= 0 {
		return errors.New("quantity to subtract must be positive")
	}

	item, err := s.supplyItems.GetByID(ctx, itemID)
	if err != nil {
		return errors.New("item not found")
	}

	if item.CurrentQuantity < quantityToSubtract {
		return errors.New("insufficient quantity")
	}

	// Track if we're crossing the low stock threshold
	wasAboveMin := item.CurrentQuantity >= item.MinQuantity
	item.CurrentQuantity -= quantityToSubtract
	isNowBelowMin := item.CurrentQuantity < item.MinQuantity

	if err := s.supplyItems.Update(ctx, item); err != nil {
		return fmt.Errorf("failed to consume item: %w", err)
	}

	// Send low stock notifications if threshold was just crossed
	if s.notificationService != nil && wasAboveMin && isNowBelowMin {
		users, err := s.users.ListActive(ctx)
		if err == nil {
			now := time.Now()
			for _, user := range users {
				notification := &models.Notification{
					ID:           uuid.New().String(),
					UserID:       &user.ID,
					Channel:      "app",
					TemplateID:   "supply",
					ScheduledFor: now,
					SentAt:       &now,
					Status:       "sent",
					Title:        "Niski stan zapasow",
					Body:         fmt.Sprintf("%s: %d/%d %s (ponizej minimum)", item.Name, item.CurrentQuantity, item.MinQuantity, item.Unit),
				}
				s.notificationService.CreateNotification(ctx, notification)
			}
		}
	}

	return nil
}

// SetQuantity directly sets the quantity (for corrections)
func (s *SupplyService) SetQuantity(ctx context.Context, itemID string, newQuantity int) error {
	if newQuantity < 0 {
		return errors.New("quantity cannot be negative")
	}

	item, err := s.supplyItems.GetByID(ctx, itemID)
	if err != nil {
		return errors.New("item not found")
	}

	item.CurrentQuantity = newQuantity

	if err := s.supplyItems.Update(ctx, item); err != nil {
		return fmt.Errorf("failed to set quantity: %w", err)
	}

	return nil
}

// MarkAsRefunded marks an item as refunded and deducts from shared budget
func (s *SupplyService) MarkAsRefunded(ctx context.Context, itemID string) error {
	// Get item to check refund details
	item, err := s.supplyItems.GetByID(ctx, itemID)
	if err != nil {
		return errors.New("item not found")
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

	amountToRefund := utils.DecimalStringToFloat(*item.LastRestockAmountPLN)
	currentBudget := utils.DecimalStringToFloat(settings.CurrentBudgetPLN)

	if currentBudget < amountToRefund {
		return fmt.Errorf("insufficient budget: have %.2f PLN, need %.2f PLN", currentBudget, amountToRefund)
	}

	// Update item
	item.NeedsRefund = false
	if err := s.supplyItems.Update(ctx, item); err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	// Update budget
	settings.CurrentBudgetPLN = utils.FloatToDecimalString(currentBudget - amountToRefund)
	settings.UpdatedAt = time.Now()
	if err := s.supplySettings.Upsert(ctx, settings); err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
	}

	return nil
}

// DeleteItem deletes an item (ADMIN or creator only - enforced at handler level)
func (s *SupplyService) DeleteItem(ctx context.Context, itemID string) error {
	if err := s.supplyItems.Delete(ctx, itemID); err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}
	return nil
}

// ========== Contribution Methods ==========

// GetContributions retrieves contributions with optional filters
func (s *SupplyService) GetContributions(ctx context.Context, userID *string, fromDate *time.Time) ([]models.SupplyContribution, error) {
	var contributions []models.SupplyContribution
	var err error

	if userID != nil {
		contributions, err = s.supplyContributions.ListByUserID(ctx, *userID)
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}

		// Filter by date if provided
		if fromDate != nil {
			filtered := []models.SupplyContribution{}
			for _, c := range contributions {
				if c.PeriodStart.Equal(*fromDate) || c.PeriodStart.After(*fromDate) {
					filtered = append(filtered, c)
				}
			}
			contributions = filtered
		}
	} else if fromDate != nil {
		// Get all contributions in period - need to implement differently
		// For now, get all and filter
		allContribs, err := s.getAllContributions(ctx)
		if err != nil {
			return nil, err
		}
		for _, c := range allContribs {
			if c.PeriodStart.Equal(*fromDate) || c.PeriodStart.After(*fromDate) {
				contributions = append(contributions, c)
			}
		}
	} else {
		contributions, err = s.getAllContributions(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return contributions, nil
}

// getAllContributions retrieves all contributions
func (s *SupplyService) getAllContributions(ctx context.Context) ([]models.SupplyContribution, error) {
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, err
	}

	var allContributions []models.SupplyContribution
	for _, user := range users {
		contribs, err := s.supplyContributions.ListByUserID(ctx, user.ID)
		if err != nil {
			continue
		}
		allContributions = append(allContributions, contribs...)
	}

	return allContributions, nil
}

// CreateManualContribution adds a manual contribution
func (s *SupplyService) CreateManualContribution(ctx context.Context, userID string, amountPLN float64, notes *string) error {
	if amountPLN <= 0 {
		return errors.New("amount must be positive")
	}

	now := time.Now()
	contribution := models.SupplyContribution{
		ID:          uuid.New().String(),
		UserID:      userID,
		AmountPLN:   utils.FloatToDecimalString(amountPLN),
		PeriodStart: now,
		PeriodEnd:   now,
		Type:        "manual",
		Notes:       notes,
		CreatedAt:   now,
	}

	if err := s.supplyContributions.Create(ctx, &contribution); err != nil {
		return fmt.Errorf("failed to create contribution: %w", err)
	}

	// Add to budget
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}

	currentBudget := utils.DecimalStringToFloat(settings.CurrentBudgetPLN)
	settings.CurrentBudgetPLN = utils.FloatToDecimalString(currentBudget + amountPLN)
	settings.UpdatedAt = time.Now()

	if err := s.supplySettings.Upsert(ctx, settings); err != nil {
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

	now := time.Now()
	if now.Weekday().String() != settings.ContributionDay {
		return nil // Not the right day
	}

	// Get all active users
	users, err := s.users.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	weekEnd := weekStart.AddDate(0, 0, 6)

	totalContributed := 0.0
	weeklyContribution := utils.DecimalStringToFloat(settings.WeeklyContributionPLN)

	// Create contribution for each active user
	for _, user := range users {
		contribution := models.SupplyContribution{
			ID:          uuid.New().String(),
			UserID:      user.ID,
			AmountPLN:   settings.WeeklyContributionPLN,
			PeriodStart: weekStart,
			PeriodEnd:   weekEnd,
			Type:        "weekly_auto",
			CreatedAt:   now,
		}

		if err := s.supplyContributions.Create(ctx, &contribution); err != nil {
			return fmt.Errorf("failed to create contribution for user %s: %w", user.Email, err)
		}

		totalContributed += weeklyContribution
	}

	// Update budget
	currentBudget := utils.DecimalStringToFloat(settings.CurrentBudgetPLN)
	settings.CurrentBudgetPLN = utils.FloatToDecimalString(currentBudget + totalContributed)
	settings.LastContributionAt = now
	settings.UpdatedAt = now

	if err := s.supplySettings.Upsert(ctx, settings); err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
	}

	return nil
}

// GetStats returns spending statistics
func (s *SupplyService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	// Get all items for stats
	items, err := s.supplyItems.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}

	// Total spent by category
	categoryStats := make(map[string]map[string]interface{})
	for _, item := range items {
		if item.LastRestockAmountPLN != nil {
			if _, exists := categoryStats[item.Category]; !exists {
				categoryStats[item.Category] = map[string]interface{}{
					"_id":        item.Category,
					"totalSpent": 0.0,
					"count":      0,
				}
			}
			amount := utils.DecimalStringToFloat(*item.LastRestockAmountPLN)
			categoryStats[item.Category]["totalSpent"] = categoryStats[item.Category]["totalSpent"].(float64) + amount
			categoryStats[item.Category]["count"] = categoryStats[item.Category]["count"].(int) + 1
		}
	}

	categoryStatsSlice := make([]map[string]interface{}, 0, len(categoryStats))
	for _, stat := range categoryStats {
		categoryStatsSlice = append(categoryStatsSlice, stat)
	}

	// Spending by user
	userStats := make(map[string]map[string]interface{})
	for _, item := range items {
		if item.LastRestockedByUserID != nil && item.LastRestockAmountPLN != nil {
			userID := *item.LastRestockedByUserID
			if _, exists := userStats[userID]; !exists {
				userStats[userID] = map[string]interface{}{
					"_id":        userID,
					"totalSpent": 0.0,
					"count":      0,
				}
			}
			amount := utils.DecimalStringToFloat(*item.LastRestockAmountPLN)
			userStats[userID]["totalSpent"] = userStats[userID]["totalSpent"].(float64) + amount
			userStats[userID]["count"] = userStats[userID]["count"].(int) + 1
		}
	}

	userStatsSlice := make([]map[string]interface{}, 0, len(userStats))
	for _, stat := range userStats {
		userStatsSlice = append(userStatsSlice, stat)
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
		"byCategory":          categoryStatsSlice,
		"byUser":              userStatsSlice,
		"recentContributions": recentContributions,
	}, nil
}
