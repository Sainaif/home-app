package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/sainaif/holy-home/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SessionService struct {
	db *mongo.Database
}

func NewSessionService(db *mongo.Database) *SessionService {
	return &SessionService{db: db}
}

// CreateSession creates a new session with a refresh token
func (s *SessionService) CreateSession(ctx context.Context, userID primitive.ObjectID, refreshToken, name, ipAddress, userAgent string, expiresAt time.Time) error {
	// Hash the refresh token before storing
	hashedToken := hashToken(refreshToken)

	session := models.Session{
		ID:           primitive.NewObjectID(),
		UserID:       userID,
		RefreshToken: hashedToken,
		Name:         name,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    time.Now(),
		LastUsedAt:   time.Now(),
		ExpiresAt:    expiresAt,
	}

	_, err := s.db.Collection("sessions").InsertOne(ctx, session)
	return err
}

// GetUserSessions retrieves all sessions for a user
func (s *SessionService) GetUserSessions(ctx context.Context, userID primitive.ObjectID) ([]models.Session, error) {
	// Clean up expired sessions first
	_, _ = s.db.Collection("sessions").DeleteMany(ctx, bson.M{
		"user_id":    userID,
		"expires_at": bson.M{"$lt": time.Now()},
	})

	cursor, err := s.db.Collection("sessions").Find(ctx, bson.M{
		"user_id":    userID,
		"expires_at": bson.M{"$gte": time.Now()},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var sessions []models.Session
	if err := cursor.All(ctx, &sessions); err != nil {
		return nil, err
	}

	return sessions, nil
}

// ValidateSession validates a refresh token and updates last used time
func (s *SessionService) ValidateSession(ctx context.Context, refreshToken string) (*models.Session, error) {
	hashedToken := hashToken(refreshToken)

	var session models.Session
	err := s.db.Collection("sessions").FindOne(ctx, bson.M{
		"refresh_token": hashedToken,
		"expires_at":    bson.M{"$gte": time.Now()},
	}).Decode(&session)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("invalid or expired session")
		}
		return nil, err
	}

	// Update last used time
	_, _ = s.db.Collection("sessions").UpdateOne(ctx, bson.M{"_id": session.ID}, bson.M{
		"$set": bson.M{"last_used_at": time.Now()},
	})

	return &session, nil
}

// RenameSession renames a session
func (s *SessionService) RenameSession(ctx context.Context, sessionID, userID primitive.ObjectID, newName string) error {
	result, err := s.db.Collection("sessions").UpdateOne(ctx, bson.M{
		"_id":     sessionID,
		"user_id": userID,
	}, bson.M{
		"$set": bson.M{"name": newName},
	})

	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("session not found")
	}

	return nil
}

// DeleteSession deletes a specific session
func (s *SessionService) DeleteSession(ctx context.Context, sessionID, userID primitive.ObjectID) error {
	result, err := s.db.Collection("sessions").DeleteOne(ctx, bson.M{
		"_id":     sessionID,
		"user_id": userID,
	})

	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("session not found")
	}

	return nil
}

// RevokeSession revokes a session by refresh token (used during logout)
func (s *SessionService) RevokeSession(ctx context.Context, refreshToken string) error {
	hashedToken := hashToken(refreshToken)

	_, err := s.db.Collection("sessions").DeleteOne(ctx, bson.M{
		"refresh_token": hashedToken,
	})

	return err
}

// RevokeAllUserSessions revokes all sessions for a user (except optionally the current one)
func (s *SessionService) RevokeAllUserSessions(ctx context.Context, userID primitive.ObjectID, exceptSessionID *primitive.ObjectID) error {
	filter := bson.M{"user_id": userID}

	if exceptSessionID != nil {
		filter["_id"] = bson.M{"$ne": *exceptSessionID}
	}

	_, err := s.db.Collection("sessions").DeleteMany(ctx, filter)
	return err
}

// CleanupExpiredSessions removes all expired sessions (should be run periodically)
func (s *SessionService) CleanupExpiredSessions(ctx context.Context) error {
	_, err := s.db.Collection("sessions").DeleteMany(ctx, bson.M{
		"expires_at": bson.M{"$lt": time.Now()},
	})
	return err
}

// hashToken creates a SHA-256 hash of the token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
