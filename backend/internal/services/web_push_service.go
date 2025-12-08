package services

import (
	"context"

	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
)

type WebPushService struct {
	webPushSubscriptions repository.WebPushSubscriptionRepository
}

func NewWebPushService(webPushSubscriptions repository.WebPushSubscriptionRepository) *WebPushService {
	return &WebPushService{webPushSubscriptions: webPushSubscriptions}
}

func (s *WebPushService) CreateSubscription(ctx context.Context, subscription *models.WebPushSubscription) error {
	return s.webPushSubscriptions.Create(ctx, subscription)
}

func (s *WebPushService) GetSubscriptionsByUserID(ctx context.Context, userID string) ([]models.WebPushSubscription, error) {
	return s.webPushSubscriptions.ListByUserID(ctx, userID)
}

func (s *WebPushService) DeleteSubscription(ctx context.Context, endpoint string) error {
	return s.webPushSubscriptions.Delete(ctx, endpoint)
}
