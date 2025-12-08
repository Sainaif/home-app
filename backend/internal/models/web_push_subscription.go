package models

// WebPushSubscription represents a web push subscription for push notifications
type WebPushSubscription struct {
	ID             string  `db:"id" json:"id"`
	UserID         string  `db:"user_id" json:"userId"`
	Endpoint       string  `db:"endpoint" json:"endpoint"`
	ExpirationTime *string `db:"expiration_time" json:"expirationTime,omitempty"`
	P256dh         string  `db:"p256dh" json:"p256dh"`
	Auth           string  `db:"auth" json:"auth"`
}
