package dataoperations

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/mongodb"
)

// VoteCountRepair records one entry whose cached VoteCount disagreed with
// the number of vote rows actually pointing at it.
type VoteCountRepair struct {
	EntryID string
	From    int
	To      int
}

// VoteReconcileReport summarizes one run of ReconcileVoteCounts. Both
// counters are zero on a healthy deployment, which is what makes the pass
// safe to run on a ticker — a clean run writes nothing at all.
type VoteReconcileReport struct {
	// DuplicatesArchived is the number of surplus vote rows moved out of
	// the votes collection into votes_archive.
	DuplicatesArchived int
	// EntriesRepaired is len(Repairs); entries whose counter was rewritten.
	EntriesRepaired int
	// Repairs carries the before/after for every rewritten counter, so the
	// caller can log an audit trail of exactly what changed.
	Repairs []VoteCountRepair
}

// Clean reports whether the run found nothing to fix.
func (r VoteReconcileReport) Clean() bool {
	return r.DuplicatesArchived == 0 && r.EntriesRepaired == 0
}

// Summary renders the report as a single log line.
func (r VoteReconcileReport) Summary() string {
	return fmt.Sprintf("%d duplicate vote row(s) archived, %d entry counter(s) repaired",
		r.DuplicatesArchived, r.EntriesRepaired)
}

// ReconcileVoteCounts re-syncs the denormalized Entry.VoteCount with the
// votes collection, which is the single source of truth for who voted on
// what.
//
// It runs in two steps:
//
//  1. Surplus rows — a second, third, … vote by the SAME user on the SAME
//     entry — are moved to votes_archive. This is what a vote-flooding tool
//     leaves behind, and it is also what blocks the unique index from
//     building (see EnsureIndexes).
//  2. Every entry's VoteCount is recomputed from the remaining rows and
//     rewritten only where it disagrees.
//
// Nothing is destroyed. The earliest vote for each (user, entry) pair always
// stays in the live collection, so no user loses the vote they cast; the
// surplus rows are copied to votes_archive BEFORE they leave votes, and they
// keep their original _id and fields, so restoring any of them is a copy
// back. A failure at any point aborts before the matching delete.
//
// Idempotent and safe to run concurrently on several instances: the archive
// insert tolerates a row already being archived, the delete is by explicit
// _id, and the counter rewrite converges on the same value regardless of who
// gets there first.
//
// Orphan rows (votes whose entry has since been deleted) are deliberately
// left alone — they cost nothing and removing them would be the one kind of
// cleanup this pass can't reverse from an entry that no longer exists.
func (do *DataOperations) ReconcileVoteCounts() (VoteReconcileReport, error) {
	var rep VoteReconcileReport

	archived, err := do.archiveDuplicateVotes()
	rep.DuplicatesArchived = archived
	if err != nil {
		return rep, err
	}

	repairs, err := do.repairEntryVoteCounts()
	rep.Repairs = repairs
	rep.EntriesRepaired = len(repairs)
	if err != nil {
		return rep, err
	}
	return rep, nil
}

// voteRef is one row inside a duplicate group — just enough to rebuild the
// full Vote (the pair's userid/entryid live on the group key).
type voteRef struct {
	ID        string    `bson:"id"`
	CreatedAt time.Time `bson:"createdat"`
}

// voteDuplicateGroup is one (userid, entryid) pair holding more than one
// vote row, as returned by the aggregation in archiveDuplicateVotes.
type voteDuplicateGroup struct {
	Key struct {
		UserID  string `bson:"userid"`
		EntryID string `bson:"entryid"`
	} `bson:"_id"`
	Votes []voteRef `bson:"votes"`
	N     int       `bson:"n"`
}

// pickSurvivingVote returns the index of the row that stays in the live
// collection: the earliest by createdat, with the lowest _id as the
// tie-break so the choice is deterministic across runs and across
// instances reconciling at the same moment. Everything else in the group
// is surplus.
//
// The user's real vote is the first one they cast, so keeping the earliest
// also preserves the original vote timestamp rather than a tool's replay
// of it. Returns -1 for an empty group.
func pickSurvivingVote(votes []voteRef) int {
	keep := -1
	for i := range votes {
		if votes[i].ID == "" {
			continue
		}
		if keep < 0 {
			keep = i
			continue
		}
		if votes[i].CreatedAt.Before(votes[keep].CreatedAt) ||
			(votes[i].CreatedAt.Equal(votes[keep].CreatedAt) && votes[i].ID < votes[keep].ID) {
			keep = i
		}
	}
	return keep
}

// archiveDuplicateVotes moves every surplus (user, entry) vote row into
// votes_archive and returns how many were moved. The earliest row per pair
// — oldest createdat, _id as the tie-break so the choice is stable across
// runs and across instances — is the one that stays.
//
// Rows with an empty userid are skipped entirely: they can't be attributed
// to anyone, so there is no way to tell a duplicate from a distinct vote,
// and guessing would mean discarding real ones.
func (do *DataOperations) archiveDuplicateVotes() (int, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"userid": bson.M{"$gt": ""}}},
		{"$group": bson.M{
			"_id":   bson.M{"userid": "$userid", "entryid": "$entryid"},
			"votes": bson.M{"$push": bson.M{"id": "$_id", "createdat": "$createdat"}},
			"n":     bson.M{"$sum": 1},
		}},
		{"$match": bson.M{"n": bson.M{"$gt": 1}}},
	}
	groups, err := mongodb.Aggregate[voteDuplicateGroup](do.DB, CollectionVotes, pipeline)
	if err != nil {
		return 0, err
	}

	archived := 0
	for _, g := range groups {
		if len(g.Votes) < 2 || g.Key.UserID == "" {
			continue
		}
		keep := pickSurvivingVote(g.Votes)
		if keep < 0 {
			continue
		}

		surplusIDs := make([]string, 0, len(g.Votes)-1)
		inserts := make([]mongo.WriteModel, 0, len(g.Votes)-1)
		for i, v := range g.Votes {
			if i == keep || v.ID == "" {
				continue
			}
			row := models.Vote{
				ID:        v.ID,
				UserID:    g.Key.UserID,
				EntryID:   g.Key.EntryID,
				CreatedAt: v.CreatedAt,
			}
			inserts = append(inserts, mongo.NewInsertOneModel().
				SetDocument(*models.NewArchivedVote(row, models.VoteArchiveReasonDuplicate)))
			surplusIDs = append(surplusIDs, v.ID)
		}
		if len(surplusIDs) == 0 {
			continue
		}

		// Archive first, delete second. A duplicate _id here means a
		// previous run (or another instance) already archived the row —
		// harmless, and BulkWrite reports it rather than failing. Any other
		// error returns before the delete, so a row can never leave votes
		// without a copy already sitting in votes_archive.
		if _, err := mongodb.BulkWrite(do.DB, CollectionVotesArchive, inserts); err != nil {
			return archived, err
		}
		n, err := mongodb.DeleteAllCount(do.DB, CollectionVotes,
			bson.M{"_id": bson.M{"$in": surplusIDs}})
		if err != nil {
			return archived, err
		}
		archived += int(n)
	}
	return archived, nil
}

// voteTally is one entry's true vote-row count.
type voteTally struct {
	EntryID string `bson:"_id"`
	N       int    `bson:"n"`
}

// entryVoteCount is the projection repairEntryVoteCounts reads — just the
// cached counter, so the pass never pulls entry bodies into memory.
type entryVoteCount struct {
	ID        string `bson:"_id"`
	VoteCount int    `bson:"votecount"`
}

// repairEntryVoteCounts rewrites every Entry.VoteCount that disagrees with
// the number of vote rows pointing at it, and returns one record per
// rewrite. Entries already in agreement are not written at all.
func (do *DataOperations) repairEntryVoteCounts() ([]VoteCountRepair, error) {
	tallies, err := mongodb.Aggregate[voteTally](do.DB, CollectionVotes, []bson.M{
		{"$group": bson.M{"_id": "$entryid", "n": bson.M{"$sum": 1}}},
	})
	if err != nil {
		return nil, err
	}
	truth := make(map[string]int, len(tallies))
	for _, t := range tallies {
		truth[t.EntryID] = t.N
	}

	opts := options.Find().SetProjection(bson.M{"votecount": 1})
	entries, err := mongodb.Query[entryVoteCount](do.DB, CollectionEntries, bson.M{}, opts)
	if err != nil {
		return nil, err
	}

	repairs := []VoteCountRepair{}
	for _, e := range entries {
		want := truth[e.ID] // absent from the map == no vote rows == 0
		if want == e.VoteCount {
			continue
		}
		// Deliberately not SetEntryVoteCount: that helper also bumps
		// updatedat, which is right for a merge but wrong for a janitor
		// pass — it would shove every repaired entry to the top of the
		// recently-updated ordering.
		if err := mongodb.SetValue(do.DB, CollectionEntries, e.ID, "votecount", want); err != nil {
			return repairs, err
		}
		repairs = append(repairs, VoteCountRepair{EntryID: e.ID, From: e.VoteCount, To: want})
	}
	return repairs, nil
}
