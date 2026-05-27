package dataoperations

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/mongodb"
)

// FindVote returns the user's vote for the given entry, or nil if the
// user has not voted on it.
func (do *DataOperations) FindVote(userID, entryID string) (*models.Vote, error) {
	if userID == "" || entryID == "" {
		return nil, nil
	}
	filter := bson.M{"userid": userID, "entryid": entryID}
	return mongodb.QueryOne[models.Vote](do.DB, CollectionVotes, filter, nil)
}

// InsertVote persists a new vote. The handler is responsible for checking
// no vote exists already; we don't have a unique index to lean on.
func (do *DataOperations) InsertVote(v *models.Vote) error {
	return mongodb.InsertOne(do.DB, CollectionVotes, *v)
}

// DeleteVote removes the user's vote for the entry. No-op when none
// exists.
func (do *DataOperations) DeleteVote(userID, entryID string) error {
	if userID == "" || entryID == "" {
		return nil
	}
	filter := bson.M{"userid": userID, "entryid": entryID}
	return mongodb.DeleteAll(do.DB, CollectionVotes, filter)
}

// CountVotesForEntry returns the number of vote rows pointing at an entry.
// Used by the merge handler to recompute Entry.VoteCount after vote
// migration so it stays in sync with the votes collection.
func (do *DataOperations) CountVotesForEntry(entryID string) (int, error) {
	if entryID == "" {
		return 0, nil
	}
	n, err := mongodb.Count(do.DB, CollectionVotes, bson.M{"entryid": entryID}, nil)
	return int(n), err
}

// MigrateVotesToEntry retargets every vote on srcEntryID to dstEntryID,
// dropping votes from users who already voted on the destination so the
// per-(user, entry) uniqueness invariant survives the merge. Returns the
// number of votes successfully retargeted (transferred); the difference
// against the original src count is the number of duplicate votes that
// were deleted instead.
//
// Anonymous votes (UserID == "") cannot be deduplicated against the
// destination — they're retargeted unconditionally.
func (do *DataOperations) MigrateVotesToEntry(srcEntryID, dstEntryID string) (int, error) {
	if srcEntryID == "" || dstEntryID == "" || srcEntryID == dstEntryID {
		return 0, nil
	}
	srcVotes, err := mongodb.Query[models.Vote](do.DB, CollectionVotes, bson.M{"entryid": srcEntryID}, nil)
	if err != nil {
		return 0, err
	}
	if len(srcVotes) == 0 {
		return 0, nil
	}
	dstVotes, err := mongodb.Query[models.Vote](do.DB, CollectionVotes, bson.M{"entryid": dstEntryID}, nil)
	if err != nil {
		return 0, err
	}
	dstByUser := make(map[string]struct{}, len(dstVotes))
	for _, v := range dstVotes {
		if v.UserID == "" {
			continue
		}
		dstByUser[v.UserID] = struct{}{}
	}
	transferred := 0
	for _, v := range srcVotes {
		if v.UserID != "" {
			if _, exists := dstByUser[v.UserID]; exists {
				if err := mongodb.DeleteOne(do.DB, CollectionVotes, v.ID); err != nil {
					return transferred, err
				}
				continue
			}
			dstByUser[v.UserID] = struct{}{}
		}
		if err := mongodb.SetValue(do.DB, CollectionVotes, v.ID, "entryid", dstEntryID); err != nil {
			return transferred, err
		}
		transferred++
	}
	return transferred, nil
}

// SetEntryVoteCount writes the votecount field directly. Used by the
// merge handler after a vote migration to reset both entries' counters
// to a recomputed truth value, rather than chasing the delta with
// IncrementEntryVoteCount.
func (do *DataOperations) SetEntryVoteCount(entryID string, count int) error {
	if entryID == "" {
		return nil
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, entryID, "votecount", count); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionEntries, entryID, "updatedat", time.Now().UTC())
}

// VotedEntryIDs returns the subset of entryIDs the user has voted on
// as a set keyed by entry ID. List handlers call this once and look up
// each record's voted state from the resulting map. Empty inputs
// short-circuit without a Mongo round trip — anonymous viewers and empty
// pages cost nothing.
func (do *DataOperations) VotedEntryIDs(userID string, entryIDs []string) (map[string]bool, error) {
	out := map[string]bool{}
	if userID == "" || len(entryIDs) == 0 {
		return out, nil
	}
	filter := bson.M{"userid": userID, "entryid": bson.M{"$in": entryIDs}}
	votes, err := mongodb.Query[models.Vote](do.DB, CollectionVotes, filter, nil)
	if err != nil {
		return nil, err
	}
	for _, v := range votes {
		out[v.EntryID] = true
	}
	return out, nil
}
