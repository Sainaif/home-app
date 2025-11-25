package services

import (
	"context"
	"encoding/json"
	"log"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/sainaif/holy-home/internal/config"
	"github.com/sainaif/holy-home/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type NotificationService struct {
	db                            *mongo.Database
	eventService                  *EventService
	webPushService                *WebPushService
	notificationPreferenceService *NotificationPreferenceService
	cfg                           *config.Config
}

func NewNotificationService(db *mongo.Database, eventService *EventService, webPushService *WebPushService, notificationPreferenceService *NotificationPreferenceService, cfg *config.Config) *NotificationService {
	return &NotificationService{db: db, eventService: eventService, webPushService: webPushService, notificationPreferenceService: notificationPreferenceService, cfg: cfg}
}

func (s *NotificationService) CreateNotification(ctx context.Context, notification *models.Notification) error {
	preferences, err := s.notificationPreferenceService.GetPreferences(ctx, *notification.UserID)
	if err != nil {
		return err
	}
	if !preferences.AllEnabled || !preferences.Preferences[notification.TemplateID] {
		return nil
	}

	_, err = s.db.Collection("notifications").InsertOne(ctx, notification)
	if err != nil {
		return err
	}

	s.eventService.BroadcastToUser(*notification.UserID, EventNotificationCreated, map[string]interface{}{
		"notification": notification,
	})

	subscriptions, err := s.webPushService.GetSubscriptionsByUserID(ctx, *notification.UserID)
	if err == nil {
		for _, sub := range subscriptions {
			s.sendPushNotification(sub, notification)
		}
	}

	return nil
}

func (s *NotificationService) sendPushNotification(sub *models.WebPushSubscription, notification *models.Notification) {
	vapidPrivateKey := s.cfg.VAPID.PrivateKey
	if vapidPrivateKey == "" {
		return
	}

	subJSON, _ := json.Marshal(sub)
	var subscription webpush.Subscription
	json.Unmarshal(subJSON, &subscription)

	notificationJSON, _ := json.Marshal(notification)

	resp, err := webpush.SendNotification(notificationJSON, &subscription, &webpush.Options{
		VAPIDPrivateKey: vapidPrivateKey,
	})
	if err != nil {
		log.Printf("Error sending push notification: %v", err)
	}
	if resp.StatusCode == 410 {
		s.webPushService.DeleteSubscription(context.Background(), sub.Endpoint)
	}
	defer resp.Body.Close()
}

func (s *NotificationService) GetNotificationsForUser(ctx context.Context, userID primitive.ObjectID) ([]*models.Notification, error) {
	cursor, err := s.db.Collection("notifications").Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []*models.Notification
	if err := cursor.All(ctx, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

func (s *NotificationService) MarkNotificationAsRead(ctx context.Context, notificationID, userID primitive.ObjectID) error {
	_, err := s.db.Collection("notifications").UpdateOne(
		ctx,
		bson.M{"_id": notificationID, "user_id": userID},
		bson.M{"$set": bson.M{"read": true}},
	)
	return err
}

func (s *NotificationService) MarkAllNotificationsAsRead(ctx context.Context, userID primitive.ObjectID) error {
	_, err := s.db.Collection("notifications").UpdateMany(
		ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": bson.M{"read": true}},
	)
	return err
}
