package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/sainaif/holy-home/internal/models"
	"github.com/sainaif/holy-home/internal/repository"
)

type SessionService struct {
	sessions repository.SessionRepository
}

func NewSessionService(sessions repository.SessionRepository) *SessionService {
	return &SessionService{sessions: sessions}
}

// CreateSession creates a new session with a refresh token
func (s *SessionService) CreateSession(ctx context.Context, userID string, refreshToken, name, ipAddress, userAgent string, expiresAt time.Time) error {
	// Hash the refresh token before storing
	hashedToken := hashToken(refreshToken)

	session := models.Session{
		ID:           uuid.New().String(),
		UserID:       userID,
		RefreshToken: hashedToken,
		Name:         name,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    time.Now(),
		LastUsedAt:   time.Now(),
		ExpiresAt:    expiresAt,
	}

	if err := s.sessions.Create(ctx, &session); err != nil {
		return err
	}

	log.Printf("[SESSION] Created: user ID %s from IP %s (session ID: %s, name: %q)", userID, ipAddress, session.ID, name)
	return nil
}

// GetUserSessions retrieves all sessions for a user
func (s *SessionService) GetUserSessions(ctx context.Context, userID string) ([]models.Session, error) {
	// Clean up expired sessions first
	_ = s.sessions.DeleteExpired(ctx)

	return s.sessions.ListByUserID(ctx, userID)
}

// ValidateSession validates a refresh token and updates last used time
func (s *SessionService) ValidateSession(ctx context.Context, refreshToken string) (*models.Session, error) {
	hashedToken := hashToken(refreshToken)

	session, err := s.sessions.GetByRefreshToken(ctx, hashedToken)
	if err != nil {
		return nil, errors.New("invalid or expired session")
	}

	// Check if session was found (repository returns nil, nil for not found)
	if session == nil {
		return nil, errors.New("invalid or expired session")
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, errors.New("session expired")
	}

	// Update last used time
	session.LastUsedAt = time.Now()
	_ = s.sessions.Update(ctx, session)

	return session, nil
}

// RenameSession renames a session
func (s *SessionService) RenameSession(ctx context.Context, sessionID, userID string, newName string) error {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return errors.New("session not found")
	}

	// Check if session was found (repository returns nil, nil for not found)
	if session == nil {
		return errors.New("session not found")
	}

	// Verify the session belongs to the user
	if session.UserID != userID {
		return errors.New("session not found")
	}

	session.Name = newName
	return s.sessions.Update(ctx, session)
}

// DeleteSession deletes a specific session
func (s *SessionService) DeleteSession(ctx context.Context, sessionID, userID string) error {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return errors.New("session not found")
	}

	// Check if session was found (repository returns nil, nil for not found)
	if session == nil {
		return errors.New("session not found")
	}

	// Verify the session belongs to the user
	if session.UserID != userID {
		return errors.New("session not found")
	}

	if err := s.sessions.Delete(ctx, sessionID); err != nil {
		return err
	}

	log.Printf("[SESSION] Deleted: session ID %s for user ID %s", sessionID, userID)
	return nil
}

// RevokeSession revokes a session by refresh token (used during logout)
func (s *SessionService) RevokeSession(ctx context.Context, refreshToken string) error {
	hashedToken := hashToken(refreshToken)

	session, err := s.sessions.GetByRefreshToken(ctx, hashedToken)
	if err != nil {
		// Session doesn't exist or already revoked - that's fine
		return nil
	}

	// Check if session was found (repository returns nil, nil for not found)
	if session == nil {
		// Session doesn't exist or already revoked - that's fine
		return nil
	}

	if err := s.sessions.Delete(ctx, session.ID); err != nil {
		return err
	}

	log.Printf("[SESSION] Revoked: session ID %s for user ID %s", session.ID, session.UserID)
	return nil
}

// RevokeAllUserSessions revokes all sessions for a user (except optionally the current one)
func (s *SessionService) RevokeAllUserSessions(ctx context.Context, userID string, exceptSessionID *string) error {
	sessions, err := s.sessions.ListByUserID(ctx, userID)
	if err != nil {
		return err
	}

	revokedCount := 0
	for _, session := range sessions {
		if exceptSessionID != nil && session.ID == *exceptSessionID {
			continue
		}
		_ = s.sessions.Delete(ctx, session.ID)
		revokedCount++
	}

	if revokedCount > 0 {
		log.Printf("[SESSION] Revoked all sessions: %d sessions revoked for user ID %s", revokedCount, userID)
	}

	return nil
}

// CleanupExpiredSessions removes all expired sessions (should be run periodically)
func (s *SessionService) CleanupExpiredSessions(ctx context.Context) error {
	return s.sessions.DeleteExpired(ctx)
}

// hashToken creates a SHA-256 hash of the token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
