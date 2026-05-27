package models

import (
	"time"

	"github.com/google/uuid"
)

// Comment is a single markdown-bodied response posted against an Entry.
// One UserID per comment — anonymous comments are not allowed (write
// endpoints are gated by RequireUser). The Entry's CommentCount field
// mirrors the number of Comment rows pointing at it; drift will be
// reconciled by the same startup pass that watches VoteCount.
//
// IsInternal hides the comment from the Portal — Console always shows
// every comment, Portal only shows IsInternal=false. Portal-created
// comments default to external (false); Console-created ones default to
// internal (true). Admins and editors can flip the flag from the Console.
type Comment struct {
	ID         string `bson:"_id"`
	EntryID    string
	UserID     string
	Body       string
	IsInternal bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewComment constructs a Comment with a fresh ID and current timestamps.
// Caller fills EntryID, UserID, Body.
func NewComment() *Comment {
	now := time.Now().UTC()
	return &Comment{
		ID:        uuid.New().String(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}
