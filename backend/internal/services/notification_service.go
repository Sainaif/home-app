package services

import (
	"context"
	"encoding/json"
	"log"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/sainaif/holy-home/internal/config"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
)

type NotificationService struct {
	notifications                 repository.NotificationRepository
	eventService                  *EventService
	webPushService                *WebPushService
	notificationPreferenceService *NotificationPreferenceService
	cfg                           *config.Config
}

func NewNotificationService(
	notifications repository.NotificationRepository,
	eventService *EventService,
	webPushService *WebPushService,
	notificationPreferenceService *NotificationPreferenceService,
	cfg *config.Config,
) *NotificationService {
	return &NotificationService{
		notifications:                 notifications,
		eventService:                  eventService,
		webPushService:                webPushService,
		notificationPreferenceService: notificationPreferenceService,
		cfg:                           cfg,
	}
}

func (s *NotificationService) CreateNotification(ctx context.Context, notification *models.Notification) error {
	if notification.UserID == nil {
		return nil
	}

	preferences, err := s.notificationPreferenceService.GetPreferences(ctx, *notification.UserID)
	if err != nil {
		return err
	}
	if !preferences.AllEnabled || !preferences.Preferences[notification.TemplateID] {
		return nil
	}

	if err := s.notifications.Create(ctx, notification); err != nil {
		return err
	}

	s.eventService.BroadcastToUser(*notification.UserID, EventNotificationCreated, map[string]interface{}{
		"notification": notification,
	})

	subscriptions, err := s.webPushService.GetSubscriptionsByUserID(ctx, *notification.UserID)
	if err == nil {
		for _, sub := range subscriptions {
			s.sendPushNotification(&sub, notification)
		}
	}

	return nil
}

func (s *NotificationService) sendPushNotification(sub *models.WebPushSubscription, notification *models.Notification) {
	vapidPrivateKey := s.cfg.VAPID.PrivateKey
	if vapidPrivateKey == "" {
		return
	}

	// Build webpush subscription format
	subscription := webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}

	notificationJSON, _ := json.Marshal(notification)

	resp, err := webpush.SendNotification(notificationJSON, &subscription, &webpush.Options{
		Subscriber:      "mailto:" + s.cfg.Admin.Email,
		VAPIDPublicKey:  s.cfg.VAPID.PublicKey,
		VAPIDPrivateKey: vapidPrivateKey,
	})
	if err != nil {
		log.Printf("Error sending push notification: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 410 {
		s.webPushService.DeleteSubscription(context.Background(), sub.Endpoint)
	}
}

func (s *NotificationService) GetNotificationsForUser(ctx context.Context, userID string) ([]models.Notification, error) {
	return s.notifications.ListByUserID(ctx, userID, 100)
}

func (s *NotificationService) MarkNotificationAsRead(ctx context.Context, notificationID, userID string) error {
	// First verify the notification belongs to this user
	notification, err := s.notifications.GetByID(ctx, notificationID)
	if err != nil {
		return err
	}
	if notification.UserID == nil || *notification.UserID != userID {
		return nil // Silently ignore if not the owner
	}
	return s.notifications.MarkAsRead(ctx, notificationID)
}

func (s *NotificationService) MarkAllNotificationsAsRead(ctx context.Context, userID string) error {
	return s.notifications.MarkAllAsReadForUser(ctx, userID)
}
