package dataoperations

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

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

// InsertVote persists a new vote unconditionally. Prefer
// InsertVoteIfAbsent for anything user-driven — this one carries no
// uniqueness guarantee of its own and only survives because the unique
// index rejects a duplicate outright.
func (do *DataOperations) InsertVote(v *models.Vote) error {
	return mongodb.InsertOne(do.DB, CollectionVotes, *v)
}

// InsertVoteIfAbsent atomically creates the (user, entry) vote and reports
// whether THIS call is the one that created it. A false return means the
// user had already voted and nothing was written.
//
// This is what makes the vote counter tamper-proof. The old read-then-write
// toggle let N concurrent requests all observe "no vote yet" and all insert,
// inflating both the votes collection and Entry.VoteCount by N for a single
// user. Here the decision and the write are one findOneAndUpdate, so exactly
// one racer sees created==true and exactly one increment follows. The unique
// index is the second line of defense: a racer that loses at the index level
// gets E11000, which we report as "already voted" rather than an error.
func (do *DataOperations) InsertVoteIfAbsent(v *models.Vote) (bool, error) {
	if v == nil || v.UserID == "" || v.EntryID == "" {
		return false, nil
	}
	filter := bson.M{"userid": v.UserID, "entryid": v.EntryID}
	update := bson.M{"$setOnInsert": bson.M{
		"_id":       v.ID,
		"userid":    v.UserID,
		"entryid":   v.EntryID,
		"createdat": v.CreatedAt,
	}}
	// Before-image semantics: a nil previous document means the upsert
	// inserted, which is precisely the "we created it" signal.
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.Before)
	prev, err := mongodb.FindOneAndUpdate[models.Vote](do.DB, CollectionVotes, filter, update, opts)
	if err != nil {
		if mongodb.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, err
	}
	return prev == nil, nil
}

// DeleteVote removes the user's vote for the entry. No-op when none
// exists. Kept for callers that don't care whether a row was there;
// the vote toggle uses DeleteVoteIfPresent instead.
func (do *DataOperations) DeleteVote(userID, entryID string) error {
	if userID == "" || entryID == "" {
		return nil
	}
	filter := bson.M{"userid": userID, "entryid": entryID}
	return mongodb.DeleteAll(do.DB, CollectionVotes, filter)
}

// DeleteVoteIfPresent removes the user's vote for the entry and reports
// whether a row was actually removed. The un-vote half of the toggle
// decrements Entry.VoteCount only on a true return, so two un-vote requests
// racing on the same vote can't decrement twice and drive the counter below
// the real number of rows.
func (do *DataOperations) DeleteVoteIfPresent(userID, entryID string) (bool, error) {
	if userID == "" || entryID == "" {
		return false, nil
	}
	filter := bson.M{"userid": userID, "entryid": entryID}
	n, err := mongodb.DeleteAllCount(do.DB, CollectionVotes, filter)
	if err != nil {
		return false, err
	}
	return n > 0, nil
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
