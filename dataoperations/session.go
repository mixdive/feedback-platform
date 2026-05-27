package dataoperations

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/mongodb"
)

// SessionTTL is how long a freshly issued admin session remains valid.
// 14 days mirrors the cookie Max-Age — a routine refresh lands well before
// the next deploy on the pilot.
const SessionTTL = 14 * 24 * time.Hour

// CreateSession persists a new session for the given user and returns it.
func (do *DataOperations) CreateSession(userID string) (*models.Session, error) {
	s, err := models.NewSession(userID, SessionTTL)
	if err != nil {
		return nil, err
	}
	if err := mongodb.InsertOne(do.DB, CollectionSessions, *s); err != nil {
		return nil, err
	}
	return s, nil
}

// ResolveSession returns the user behind a session token, or nil if the
// token is unknown or the session has expired. Expired sessions are deleted
// opportunistically so the collection doesn't grow without bound.
func (do *DataOperations) ResolveSession(token string) (*models.User, error) {
	if token == "" {
		return nil, nil
	}
	s, err := mongodb.QueryOne[models.Session](do.DB, CollectionSessions, bson.M{"token": token}, nil)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	if time.Now().UTC().After(s.ExpiresAt) {
		_ = mongodb.DeleteOne(do.DB, CollectionSessions, s.ID)
		return nil, nil
	}
	return do.FindUserByID(s.UserID)
}

// DeleteSessionByToken removes the session matching the given token. Used by
// the logout handler. Missing tokens are not an error — logout should be
// idempotent.
func (do *DataOperations) DeleteSessionByToken(token string) error {
	if token == "" {
		return nil
	}
	return mongodb.DeleteAll(do.DB, CollectionSessions, bson.M{"token": token})
}
