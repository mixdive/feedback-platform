package models

import (
	"time"

	"github.com/google/uuid"
)

// Vote records that a User upvoted an Entry. One vote per (UserID,
// EntryID) — uniqueness is enforced by the handler (read-then-write); a
// unique compound index on (userid, entryid) arrives with index
// management.
//
// Voting is a toggle: posting to the vote endpoint creates this record on
// the first call and deletes it on the second. The Entry's VoteCount
// mirrors the number of Vote rows pointing at it; drift is reconciled by
// the startup pass.
type Vote struct {
	ID        string `bson:"_id"`
	UserID    string
	EntryID   string
	CreatedAt time.Time
}

// NewVote constructs a Vote with a fresh ID and current timestamp. Caller
// fills UserID and EntryID.
func NewVote() *Vote {
	return &Vote{
		ID:        uuid.New().String(),
		CreatedAt: time.Now().UTC(),
	}
}
