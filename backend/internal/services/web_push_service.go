package services

import (
	"context"

	"github.com/sainaif/holy-home/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type WebPushService struct {
	db *mongo.Database
}

func NewWebPushService(db *mongo.Database) *WebPushService {
	return &WebPushService{db: db}
}

func (s *WebPushService) CreateSubscription(ctx context.Context, subscription *models.WebPushSubscription) error {
	_, err := s.db.Collection("web_push_subscriptions").InsertOne(ctx, subscription)
	return err
}

func (s *WebPushService) GetSubscriptionsByUserID(ctx context.Context, userID primitive.ObjectID) ([]*models.WebPushSubscription, error) {
	cursor, err := s.db.Collection("web_push_subscriptions").Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var subscriptions []*models.WebPushSubscription
	if err := cursor.All(ctx, &subscriptions); err != nil {
		return nil, err
	}

	return subscriptions, nil
}

func (s *WebPushService) DeleteSubscription(ctx context.Context, endpoint string) error {
	_, err := s.db.Collection("web_push_subscriptions").DeleteOne(ctx, bson.M{"endpoint": endpoint})
	return err
}
