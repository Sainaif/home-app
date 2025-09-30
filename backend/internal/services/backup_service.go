package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/sainaif/holy-home/internal/models"
)

type BackupService struct {
	db *mongo.Database
}

func NewBackupService(db *mongo.Database) *BackupService {
	return &BackupService{db: db}
}

// BackupData represents a complete system backup
type BackupData struct {
	Version          string                   `json:"version"`
	ExportedAt       time.Time                `json:"exportedAt"`
	Users            []models.User            `json:"users"`
	Groups           []models.Group           `json:"groups"`
	Bills            []models.Bill            `json:"bills"`
	Consumptions     []models.Consumption     `json:"consumptions"`
	Payments         []models.Payment         `json:"payments"`
	Loans            []models.Loan            `json:"loans"`
	LoanPayments     []models.LoanPayment     `json:"loanPayments"`
	Chores           []models.Chore           `json:"chores"`
	ChoreAssignments []models.ChoreAssignment `json:"choreAssignments"`
	ChoreSettings    *models.ChoreSettings    `json:"choreSettings,omitempty"`
	Notifications    []models.Notification    `json:"notifications"`
	SupplySettings   *models.SupplySettings   `json:"supplySettings,omitempty"`
	SupplyItems      []models.SupplyItem      `json:"supplyItems"`
	SupplyContributions []models.SupplyContribution `json:"supplyContributions"`
}

// ExportAll exports all data from all collections
func (s *BackupService) ExportAll(ctx context.Context) (*BackupData, error) {
	backup := &BackupData{
		Version:    "1.0",
		ExportedAt: time.Now(),
	}

	// Export users
	var users []models.User
	cursor, err := s.db.Collection("users").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}
	backup.Users = users
	cursor.Close(ctx)

	// Export groups
	var groups []models.Group
	cursor, err = s.db.Collection("groups").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch groups: %w", err)
	}
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, fmt.Errorf("failed to decode groups: %w", err)
	}
	backup.Groups = groups
	cursor.Close(ctx)

	// Export bills
	var bills []models.Bill
	cursor, err = s.db.Collection("bills").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bills: %w", err)
	}
	if err := cursor.All(ctx, &bills); err != nil {
		return nil, fmt.Errorf("failed to decode bills: %w", err)
	}
	backup.Bills = bills
	cursor.Close(ctx)

	// Export consumptions
	var consumptions []models.Consumption
	cursor, err = s.db.Collection("consumptions").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch consumptions: %w", err)
	}
	if err := cursor.All(ctx, &consumptions); err != nil {
		return nil, fmt.Errorf("failed to decode consumptions: %w", err)
	}
	backup.Consumptions = consumptions
	cursor.Close(ctx)

	// Export payments
	var payments []models.Payment
	cursor, err = s.db.Collection("payments").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payments: %w", err)
	}
	if err := cursor.All(ctx, &payments); err != nil {
		return nil, fmt.Errorf("failed to decode payments: %w", err)
	}
	backup.Payments = payments
	cursor.Close(ctx)

	// Export loans
	var loans []models.Loan
	cursor, err = s.db.Collection("loans").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch loans: %w", err)
	}
	if err := cursor.All(ctx, &loans); err != nil {
		return nil, fmt.Errorf("failed to decode loans: %w", err)
	}
	backup.Loans = loans
	cursor.Close(ctx)

	// Export loan payments
	var loanPayments []models.LoanPayment
	cursor, err = s.db.Collection("loan_payments").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch loan payments: %w", err)
	}
	if err := cursor.All(ctx, &loanPayments); err != nil {
		return nil, fmt.Errorf("failed to decode loan payments: %w", err)
	}
	backup.LoanPayments = loanPayments
	cursor.Close(ctx)

	// Export chores
	var chores []models.Chore
	cursor, err = s.db.Collection("chores").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chores: %w", err)
	}
	if err := cursor.All(ctx, &chores); err != nil {
		return nil, fmt.Errorf("failed to decode chores: %w", err)
	}
	backup.Chores = chores
	cursor.Close(ctx)

	// Export chore assignments
	var choreAssignments []models.ChoreAssignment
	cursor, err = s.db.Collection("chore_assignments").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chore assignments: %w", err)
	}
	if err := cursor.All(ctx, &choreAssignments); err != nil {
		return nil, fmt.Errorf("failed to decode chore assignments: %w", err)
	}
	backup.ChoreAssignments = choreAssignments
	cursor.Close(ctx)

	// Export chore settings (singleton)
	var choreSettings models.ChoreSettings
	err = s.db.Collection("chore_settings").FindOne(ctx, bson.M{}).Decode(&choreSettings)
	if err == nil {
		backup.ChoreSettings = &choreSettings
	}

	// Export notifications
	var notifications []models.Notification
	cursor, err = s.db.Collection("notifications").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch notifications: %w", err)
	}
	if err := cursor.All(ctx, &notifications); err != nil {
		return nil, fmt.Errorf("failed to decode notifications: %w", err)
	}
	backup.Notifications = notifications
	cursor.Close(ctx)

	// Export supply settings (singleton)
	var supplySettings models.SupplySettings
	err = s.db.Collection("supply_settings").FindOne(ctx, bson.M{}).Decode(&supplySettings)
	if err == nil {
		backup.SupplySettings = &supplySettings
	}

	// Export supply items
	var supplyItems []models.SupplyItem
	cursor, err = s.db.Collection("supply_items").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch supply items: %w", err)
	}
	if err := cursor.All(ctx, &supplyItems); err != nil {
		return nil, fmt.Errorf("failed to decode supply items: %w", err)
	}
	backup.SupplyItems = supplyItems
	cursor.Close(ctx)

	// Export supply contributions
	var supplyContributions []models.SupplyContribution
	cursor, err = s.db.Collection("supply_contributions").Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch supply contributions: %w", err)
	}
	if err := cursor.All(ctx, &supplyContributions); err != nil {
		return nil, fmt.Errorf("failed to decode supply contributions: %w", err)
	}
	backup.SupplyContributions = supplyContributions
	cursor.Close(ctx)

	return backup, nil
}

// ImportAll imports data and replaces all existing data (DANGEROUS - ADMIN ONLY)
func (s *BackupService) ImportAll(ctx context.Context, backup *BackupData) error {
	// WARNING: This will delete ALL existing data and replace it with the backup

	// Clear all collections first
	collections := []string{
		"users", "groups", "bills", "consumptions",
		"payments", "loans", "loan_payments", "chores", "chore_assignments",
		"chore_settings", "notifications", "supply_settings", "supply_items",
		"supply_contributions",
	}

	for _, collName := range collections {
		if _, err := s.db.Collection(collName).DeleteMany(ctx, bson.M{}); err != nil {
			return fmt.Errorf("failed to clear collection %s: %w", collName, err)
		}
	}

	// Import users
	if len(backup.Users) > 0 {
		docs := make([]interface{}, len(backup.Users))
		for i, u := range backup.Users {
			docs[i] = u
		}
		if _, err := s.db.Collection("users").InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("failed to import users: %w", err)
		}
	}

	// Import groups
	if len(backup.Groups) > 0 {
		docs := make([]interface{}, len(backup.Groups))
		for i, g := range backup.Groups {
			docs[i] = g
		}
		if _, err := s.db.Collection("groups").InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("failed to import groups: %w", err)
		}
	}

	// Import bills
	if len(backup.Bills) > 0 {
		docs := make([]interface{}, len(backup.Bills))
		for i, b := range backup.Bills {
			docs[i] = b
		}
		if _, err := s.db.Collection("bills").InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("failed to import bills: %w", err)
		}
	}

	// Import consumptions
	if len(backup.Consumptions) > 0 {
		docs := make([]interface{}, len(backup.Consumptions))
		for i, c := range backup.Consumptions {
			docs[i] = c
		}
		if _, err := s.db.Collection("consumptions").InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("failed to import consumptions: %w", err)
		}
	}

	// Import payments
	if len(backup.Payments) > 0 {
		docs := make([]interface{}, len(backup.Payments))
		for i, p := range backup.Payments {
			docs[i] = p
		}
		if _, err := s.db.Collection("payments").InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("failed to import payments: %w", err)
		}
	}

	// Import loans
	if len(backup.Loans) > 0 {
		docs := make([]interface{}, len(backup.Loans))
		for i, l := range backup.Loans {
			docs[i] = l
		}
		if _, err := s.db.Collection("loans").InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("failed to import loans: %w", err)
		}
	}

	// Import loan payments
	if len(backup.LoanPayments) > 0 {
		docs := make([]interface{}, len(backup.LoanPayments))
		for i, lp := range backup.LoanPayments {
			docs[i] = lp
		}
		if _, err := s.db.Collection("loan_payments").InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("failed to import loan payments: %w", err)
		}
	}

	// Import chores
	if len(backup.Chores) > 0 {
		docs := make([]interface{}, len(backup.Chores))
		for i, c := range backup.Chores {
			docs[i] = c
		}
		if _, err := s.db.Collection("chores").InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("failed to import chores: %w", err)
		}
	}

	// Import chore assignments
	if len(backup.ChoreAssignments) > 0 {
		docs := make([]interface{}, len(backup.ChoreAssignments))
		for i, ca := range backup.ChoreAssignments {
			docs[i] = ca
		}
		if _, err := s.db.Collection("chore_assignments").InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("failed to import chore assignments: %w", err)
		}
	}

	// Import chore settings
	if backup.ChoreSettings != nil {
		if _, err := s.db.Collection("chore_settings").InsertOne(ctx, backup.ChoreSettings); err != nil {
			return fmt.Errorf("failed to import chore settings: %w", err)
		}
	}

	// Import notifications
	if len(backup.Notifications) > 0 {
		docs := make([]interface{}, len(backup.Notifications))
		for i, n := range backup.Notifications {
			docs[i] = n
		}
		if _, err := s.db.Collection("notifications").InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("failed to import notifications: %w", err)
		}
	}

	// Import supply settings
	if backup.SupplySettings != nil {
		if _, err := s.db.Collection("supply_settings").InsertOne(ctx, backup.SupplySettings); err != nil {
			return fmt.Errorf("failed to import supply settings: %w", err)
		}
	}

	// Import supply items
	if len(backup.SupplyItems) > 0 {
		docs := make([]interface{}, len(backup.SupplyItems))
		for i, si := range backup.SupplyItems {
			docs[i] = si
		}
		if _, err := s.db.Collection("supply_items").InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("failed to import supply items: %w", err)
		}
	}

	// Import supply contributions
	if len(backup.SupplyContributions) > 0 {
		docs := make([]interface{}, len(backup.SupplyContributions))
		for i, sc := range backup.SupplyContributions {
			docs[i] = sc
		}
		if _, err := s.db.Collection("supply_contributions").InsertMany(ctx, docs); err != nil {
			return fmt.Errorf("failed to import supply contributions: %w", err)
		}
	}

	return nil
}

// ExportJSON exports backup data as JSON string
func (s *BackupService) ExportJSON(ctx context.Context) ([]byte, error) {
	backup, err := s.ExportAll(ctx)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup to JSON: %w", err)
	}

	return jsonData, nil
}

// ImportJSON imports backup data from JSON string
func (s *BackupService) ImportJSON(ctx context.Context, jsonData []byte) error {
	var backup BackupData
	if err := json.Unmarshal(jsonData, &backup); err != nil {
		return fmt.Errorf("failed to unmarshal JSON backup: %w", err)
	}

	return s.ImportAll(ctx, &backup)
}
