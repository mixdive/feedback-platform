package models

import (
	"time"

	"github.com/google/uuid"
)

// Vote records that a User upvoted an Entry. One vote per (UserID,
// EntryID), enforced in two places: a unique partial index on
// (userid, entryid) created by DataOperations.EnsureIndexes, and an
// atomic upsert in InsertVoteIfAbsent. Neither depends on the handler
// reading before it writes, so a flood of concurrent vote requests from
// one user still yields exactly one row.
//
// Voting is a toggle: posting to the vote endpoint creates this record on
// the first call and deletes it on the second. The Entry's VoteCount
// mirrors the number of Vote rows pointing at it — the votes collection
// is the source of truth and the counter is a cache. Drift is repaired by
// DataOperations.ReconcileVoteCounts, which runs at startup and on a
// ticker.
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

// VoteArchiveReasonDuplicate marks a row the reconciliation pass pulled out
// of the votes collection because the same user already had an earlier vote
// on the same entry.
const VoteArchiveReasonDuplicate = "duplicate_user_entry_vote"

// ArchivedVote is a Vote that the reconciliation pass removed from the live
// votes collection, retained verbatim so no vote is ever destroyed. ID
// carries the removed row's original _id, which makes re-archiving the same
// row a no-op duplicate-key rather than a second copy, and makes restoring
// one a straight copy back into votes.
//
// The pass only ever archives *surplus* rows: for any (UserID, EntryID) the
// earliest vote always stays in the live collection, so a user never loses
// the vote they actually cast.
type ArchivedVote struct {
	ID         string `bson:"_id"`
	Vote       Vote
	Reason     string
	ArchivedAt time.Time
}

// NewArchivedVote wraps a vote for archival, preserving its original ID.
func NewArchivedVote(v Vote, reason string) *ArchivedVote {
	return &ArchivedVote{
		ID:         v.ID,
		Vote:       v,
		Reason:     reason,
		ArchivedAt: time.Now().UTC(),
	}
}
