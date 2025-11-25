
package services

import (
	"context"
	"time"

	"github.com/sainaif/holy-home/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NotificationPreferenceService struct {
	db *mongo.Database
}

func NewNotificationPreferenceService(db *mongo.Database) *NotificationPreferenceService {
	return &NotificationPreferenceService{db: db}
}

func (s *NotificationPreferenceService) GetPreferences(ctx context.Context, userID primitive.ObjectID) (*models.NotificationPreference, error) {
	var preferences models.NotificationPreference
	err := s.db.Collection("notification_preferences").FindOne(ctx, bson.M{"user_id": userID}).Decode(&preferences)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return s.createDefaultPreferences(ctx, userID)
		}
		return nil, err
	}
	return &preferences, nil
}

func (s *NotificationPreferenceService) UpdatePreferences(ctx context.Context, userID primitive.ObjectID, preferences map[string]bool, allEnabled bool) (*models.NotificationPreference, error) {
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var updatedPreferences models.NotificationPreference
	err := s.db.Collection("notification_preferences").FindOneAndUpdate(
		ctx,
		bson.M{"user_id": userID},
		bson.M{
			"$set": bson.M{
				"preferences":  preferences,
				"all_enabled":  allEnabled,
				"updated_at":   time.Now(),
			},
		},
		opts,
	).Decode(&updatedPreferences)
	if err != nil {
		return nil, err
	}
	return &updatedPreferences, nil
}

func (s *NotificationPreferenceService) createDefaultPreferences(ctx context.Context, userID primitive.ObjectID) (*models.NotificationPreference, error) {
	defaultPreferences := &models.NotificationPreference{
		UserID: userID,
		Preferences: map[string]bool{
			"bill":   true,
			"chore":  true,
			"supply": true,
			"loan":   true,
		},
		AllEnabled: true,
		UpdatedAt:  time.Now(),
	}
	_, err := s.db.Collection("notification_preferences").InsertOne(ctx, defaultPreferences)
	if err != nil {
		return nil, err
	}
	return defaultPreferences, nil
}
