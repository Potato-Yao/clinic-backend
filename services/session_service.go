package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"clinic-backend/models"

	"gorm.io/gorm"
)

// ErrSessionNotFound is returned when a session token does not exist or has expired.
var ErrSessionNotFound = errors.New("session not found")

// SessionService manages opaque server-side sessions.
type SessionService struct {
	db  *gorm.DB
	ttl time.Duration
}

// NewSessionService creates a session service with the given database and TTL.
func NewSessionService(db *gorm.DB, ttl time.Duration) *SessionService {
	return &SessionService{db: db, ttl: ttl}
}

// Create generates a new session for a staff member and returns the opaque
// session token and the corresponding CSRF token.
func (s *SessionService) Create(staffID int, role, casTicket string) (sessionToken, csrfToken string, err error) {
	sessionToken, err = generateToken()
	if err != nil {
		return "", "", err
	}
	csrfToken, err = generateToken()
	if err != nil {
		return "", "", err
	}

	sess := models.AuthSession{
		TokenHash:     hashToken(sessionToken),
		StaffID:       staffID,
		Role:          role,
		CSRFTokenHash: hashToken(csrfToken),
		CASTicket:     casTicket,
		ExpiresAt:     time.Now().Add(s.ttl),
	}
	if err := s.db.Create(&sess).Error; err != nil {
		return "", "", fmt.Errorf("create session: %w", err)
	}
	return sessionToken, csrfToken, nil
}

// Get resolves a session token to its active session record.
func (s *SessionService) Get(token string) (models.AuthSession, error) {
	var sess models.AuthSession
	err := s.db.Where(
		"token_hash = ? AND expires_at > ?",
		hashToken(token), time.Now(),
	).First(&sess).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.AuthSession{}, ErrSessionNotFound
		}
		return models.AuthSession{}, fmt.Errorf("lookup session: %w", err)
	}
	return sess, nil
}

// ValidateCSRF checks whether the supplied CSRF token matches the session.
func (s *SessionService) ValidateCSRF(sessionToken, csrfToken string) bool {
	sess, err := s.Get(sessionToken)
	if err != nil {
		return false
	}
	return sess.CSRFTokenHash == hashToken(csrfToken)
}

// Delete removes a session by token.
func (s *SessionService) Delete(token string) error {
	res := s.db.Where("token_hash = ?", hashToken(token)).Delete(&models.AuthSession{})
	if err := res.Error; err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteExpired removes all sessions past their expiration time.
func (s *SessionService) DeleteExpired() error {
	if err := s.db.Where("expires_at <= ?", time.Now()).Delete(&models.AuthSession{}).Error; err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
