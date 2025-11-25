package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type WebPushSubscription struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID         primitive.ObjectID `bson:"userId" json:"userId"`
	Endpoint       string             `bson:"endpoint" json:"endpoint"`
	ExpirationTime *string            `bson:"expirationTime" json:"expirationTime"`
	Keys           struct {
		P256dh string `bson:"p256dh" json:"p256dh"`
		Auth   string `bson:"auth" json:"auth"`
	} `bson:"keys" json:"keys"`
}
