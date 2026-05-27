package models

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
)

// SessionTokenBytes is the size of the random session token before base64
// encoding. 32 bytes (256 bits) is well above any realistic guessing budget.
const SessionTokenBytes = 32

// Session is the cookie-backed admin session. The cookie value is the Token
// field (URL-safe base64); the document is keyed by the same value to make
// resolution a single GetOneById lookup. ID and Token are kept separate to
// preserve the "all IDs are UUIDs" convention while keeping resolution cheap.
type Session struct {
	ID        string `bson:"_id"`
	Token     string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// NewSession constructs a Session for the given user with a fresh random
// token and an expiration set TTL into the future.
func NewSession(userID string, ttl time.Duration) (*Session, error) {
	token, err := generateSessionToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return &Session{
		ID:        uuid.New().String(),
		Token:     token,
		UserID:    userID,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}, nil
}

func generateSessionToken() (string, error) {
	b := make([]byte, SessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
