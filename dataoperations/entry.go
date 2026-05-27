package dataoperations

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/mongodb"
)

// EntryListFilter describes the entry list options.
//
// IsInternal narrows by the per-entry flag — Portal "all" lists pass a
// pointer to false to hide internal records; Console lists leave it nil to
// see everything.
//
// OwnerID restricts the result to a single user's entries — used by the
// Portal "Mine" tab and the Console's Author filter.
//
// IncludeOwnerID widens the IsInternal=&false visibility gate so an
// authenticated visitor also sees their own internal records on the
// Portal "All" tab. Ignored when IsInternal is nil; the OR is built
// natively in entryFilter so it composes with Search.
//
// Status filters on the hardcoded EntryStatus enum value. Empty string
// means "no restriction".
//
// OpenOnly narrows the result to entries whose status is NOT closed
// (completed/cancelled). Composes with Status — when Status is set,
// OpenOnly is redundant and silently ignored.
type EntryListFilter struct {
	Search         string
	Sort           string // "top" | "new"
	Page           int
	Limit          int
	IsInternal     *bool
	OwnerID        string
	IncludeOwnerID string
	EntryType      models.EntryType
	Status         models.EntryStatus
	OpenOnly       bool
	TopicID        string
}

// ListEntries runs a paginated query against the entries collection.
// Newer records first by default; "top" sort orders by votecount.
func (do *DataOperations) ListEntries(f EntryListFilter) ([]models.Entry, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 25
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	// Every branch ends with `_id` desc as a stable tie-breaker so two
	// entries inserted in the same second don't flip order between
	// requests. Without it, dev datasets seeded in one batch surface
	// in arbitrary order and "Newest" looks broken.
	sort := bson.D{}
	switch f.Sort {
	case "top":
		sort = append(sort, bson.E{Key: "votecount", Value: -1}, bson.E{Key: "createdat", Value: -1}, bson.E{Key: "_id", Value: -1})
	default:
		sort = append(sort, bson.E{Key: "createdat", Value: -1}, bson.E{Key: "_id", Value: -1})
	}
	skip := int64((f.Page - 1) * f.Limit)
	opts := options.Find().SetSort(sort).SetSkip(skip).SetLimit(int64(f.Limit))
	return mongodb.Query[models.Entry](do.DB, CollectionEntries, entryFilter(f), opts)
}

// CountEntries returns the total number of records matching the same
// filter passed to ListEntries (ignoring Page/Limit/Sort).
func (do *DataOperations) CountEntries(f EntryListFilter) (int64, error) {
	return mongodb.Count(do.DB, CollectionEntries, entryFilter(f), nil)
}

// entryFilter assembles the Mongo predicate from the filter struct.
// Multiple clauses are combined with $and so two independent $or branches
// (search vs. visibility-with-IncludeOwnerID) don't overwrite each other.
func entryFilter(f EntryListFilter) bson.M {
	clauses := []bson.M{}
	if f.Search != "" {
		clauses = append(clauses, bson.M{"$or": bson.A{
			bson.M{"title": bson.M{"$regex": f.Search, "$options": "i"}},
			bson.M{"description": bson.M{"$regex": f.Search, "$options": "i"}},
		}})
	}
	if f.IsInternal != nil {
		if f.IncludeOwnerID != "" {
			clauses = append(clauses, bson.M{"$or": bson.A{
				bson.M{"isinternal": *f.IsInternal},
				bson.M{"userid": f.IncludeOwnerID},
			}})
		} else {
			clauses = append(clauses, bson.M{"isinternal": *f.IsInternal})
		}
	}
	if f.OwnerID != "" {
		clauses = append(clauses, bson.M{"userid": f.OwnerID})
	}
	if f.EntryType != "" {
		clauses = append(clauses, bson.M{"entrytype": string(f.EntryType)})
	}
	if f.Status != "" {
		clauses = append(clauses, bson.M{"status": string(f.Status)})
	} else if f.OpenOnly {
		clauses = append(clauses, bson.M{"status": bson.M{"$nin": bson.A{
			string(models.EntryStatusCompleted),
			string(models.EntryStatusCancelled),
		}}})
	}
	// Topics live on the entry as an array (DB allows multi-assign even
	// though the v0.1 UI sets one). The Console filter is single-select
	// and matches entries whose topicids array contains the selected ID.
	if f.TopicID != "" {
		clauses = append(clauses, bson.M{"topicids": f.TopicID})
	}
	switch len(clauses) {
	case 0:
		return bson.M{}
	case 1:
		return clauses[0]
	default:
		return bson.M{"$and": clauses}
	}
}

// FindEntryByID returns the entry with the given ID, or nil.
func (do *DataOperations) FindEntryByID(id string) (*models.Entry, error) {
	return mongodb.GetOneById[models.Entry](do.DB, CollectionEntries, id)
}

// entryAuthorRow is the projection returned by ListEntryAuthorIDs — the
// $group stage emits {_id: <userid>} for each distinct non-empty userid.
type entryAuthorRow struct {
	ID string `bson:"_id"`
}

// ListEntryAuthorIDs returns the distinct non-empty userids that own at
// least one entry. Powers the Console's Author filter dropdown — the
// handler resolves these IDs into user records before sending them on the
// wire. Anonymous entries (userid == "") are skipped.
func (do *DataOperations) ListEntryAuthorIDs() ([]string, error) {
	pipeline := []bson.D{
		{{Key: "$match", Value: bson.M{"userid": bson.M{"$nin": bson.A{"", nil}}}}},
		{{Key: "$group", Value: bson.M{"_id": "$userid"}}},
	}
	rows, err := mongodb.Aggregate[entryAuthorRow](do.DB, CollectionEntries, pipeline)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out, nil
}

// UserEntryCounts is the per-user aggregate of entries authored,
// bucketed by feedback type. Other is the catch-all: every entry not
// classified as feature-request, bug, or support (including the empty/
// untyped value) lands here, so Total = FeatureRequest + Bug + Support
// + Other always holds.
type UserEntryCounts struct {
	Total          int
	FeatureRequest int
	Bug            int
	Support        int
	Other          int
}

// CountEntriesByUserAndEntryType aggregates non-anonymous entries by
// {userid, entrytype} and returns one summary per user keyed by
// user ID. Empty userid (anonymous Portal submissions) is skipped —
// those entries belong to no row on the Users page. Single $group pass
// per the "aggregate counts at list-time" memory.
func (do *DataOperations) CountEntriesByUserAndEntryType() (map[string]UserEntryCounts, error) {
	pipeline := []bson.D{
		{{Key: "$match", Value: bson.M{"userid": bson.M{"$nin": bson.A{"", nil}}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"userid": "$userid", "entrytype": "$entrytype"},
			"count": bson.M{"$sum": 1},
		}}},
	}
	type row struct {
		ID struct {
			UserID       string `bson:"userid"`
			EntryType string `bson:"entrytype"`
		} `bson:"_id"`
		Count int `bson:"count"`
	}
	rows, err := mongodb.Aggregate[row](do.DB, CollectionEntries, pipeline)
	if err != nil {
		return nil, err
	}
	out := map[string]UserEntryCounts{}
	for _, r := range rows {
		c := out[r.ID.UserID]
		c.Total += r.Count
		switch models.EntryType(r.ID.EntryType) {
		case models.EntryTypeFeatureRequest:
			c.FeatureRequest += r.Count
		case models.EntryTypeBug:
			c.Bug += r.Count
		case models.EntryTypeSupport:
			c.Support += r.Count
		default:
			// "other" enum value AND untyped ("") both fall here so
			// the Users page row always sums to Total.
			c.Other += r.Count
		}
		out[r.ID.UserID] = c
	}
	return out, nil
}

// InsertEntry persists a new entry record.
func (do *DataOperations) InsertEntry(e *models.Entry) error {
	return mongodb.InsertOne(do.DB, CollectionEntries, *e)
}

// SetEntryGitHubIssue persists the GitHubIssue sub-doc on a single
// entry and touches updatedat. Used by the create-GitHub-issue handler
// after a successful upstream POST. Once written, the link is forever
// in v0.1 — there is no unlink path.
func (do *DataOperations) SetEntryGitHubIssue(id string, issue models.GitHubIssue) error {
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "githubissue", issue); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionEntries, id, "updatedat", time.Now().UTC())
}

// IncrementEntryVoteCount adjusts an entry's vote counter atomically.
func (do *DataOperations) IncrementEntryVoteCount(id string, delta int) error {
	if err := mongodb.IncrementValue(do.DB, CollectionEntries, id, "votecount", delta); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionEntries, id, "updatedat", time.Now().UTC())
}

// IncrementEntryCommentCount adjusts an entry's comment counter
// atomically. Called from the comment-create handler with delta=+1 after a
// successful insert; the startup reconciliation pass repairs any drift.
func (do *DataOperations) IncrementEntryCommentCount(id string, delta int) error {
	if err := mongodb.IncrementValue(do.DB, CollectionEntries, id, "commentcount", delta); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionEntries, id, "updatedat", time.Now().UTC())
}

// SetEntryIsInternal flips the IsInternal flag on a single entry and
// touches updatedat.
func (do *DataOperations) SetEntryIsInternal(id string, value bool) error {
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "isinternal", value); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionEntries, id, "updatedat", time.Now().UTC())
}

// SetEntryType writes the entry-type field. Empty value
// clears the type. Caller is responsible for validating the value via
// models.IsValidEntryType before calling.
//
// Setting a non-empty value also stamps entrytypeanalysis.status to
// "disabled" so the AI worker stops re-evaluating this entry. Covers
// both the manual-edit path and the "Apply suggestion" path (which
// reaches this method via the regular update endpoint). Clearing the
// type does not reset the analysis status — the decision to skip AI
// is one-way.
//
// Reconciles the author's feature-request quota
// (User.FeatureRequestsOpen) when the type crosses the
// feature-request boundary on an OPEN entry: non-FR → FR deducts the
// author by 1, FR → non-FR refunds. Closed entries never consume
// feature-request quota in the first place, so re-typing them doesn't
// touch the counter. The floor-at-zero guard inside
// IncrementUserFeatureRequestsOpen absorbs a refund on an entry whose
// author was an admin/editor at submission (and therefore never paid
// into the counter).
func (do *DataOperations) SetEntryType(id string, ft models.EntryType) error {
	existing, err := do.FindEntryByID(id)
	if err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "entrytype", string(ft)); err != nil {
		return err
	}
	if ft != "" {
		if err := mongodb.SetValue(do.DB, CollectionEntries, id, "entrytypeanalysis.status", models.AnalysisStatusDisabled); err != nil {
			return err
		}
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "updatedat", time.Now().UTC()); err != nil {
		return err
	}
	if existing == nil || existing.UserID == "" || !models.IsEntryStatusOpen(existing.Status) {
		return nil
	}
	wasFR := existing.EntryType == models.EntryTypeFeatureRequest
	isFR := ft == models.EntryTypeFeatureRequest
	if wasFR == isFR {
		return nil
	}
	delta := -1
	if isFR {
		delta = 1
	}
	return do.IncrementUserFeatureRequestsOpen(existing.UserID, delta)
}

// SetEntryStatus writes the status field. Caller is responsible for
// validating the value via models.IsValidEntryStatus before calling.
//
// Also reconciles per-user vote quotas (User.VotesSpent) when the
// status transitions across the open/closed boundary: open→closed
// refunds every voter on this entry by 1, closed→open deducts every
// voter by 1. Voting on a closed entry never consumes quota in the
// first place; this is the symmetric correction so a status flip
// leaves the invariant "VotesSpent counts only votes on open entries"
// intact.
//
// The author's feature-request quota (User.FeatureRequestsOpen) is
// reconciled on the same boundary, but only when the entry's current
// feedback type is feature-request — bug/support/other entries never
// consumed feature-request quota at submission, so they have nothing
// to refund. Admin/editor authors never consumed quota either; the
// floor-at-zero guard inside IncrementUserFeatureRequestsOpen absorbs
// the spurious refund without going negative.
func (do *DataOperations) SetEntryStatus(id string, status models.EntryStatus) error {
	existing, err := do.FindEntryByID(id)
	if err != nil {
		return err
	}
	wasOpen := false
	if existing != nil {
		wasOpen = models.IsEntryStatusOpen(existing.Status)
	}
	isOpen := models.IsEntryStatusOpen(status)
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "status", string(status)); err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "updatedat", time.Now().UTC()); err != nil {
		return err
	}
	if existing == nil || wasOpen == isOpen {
		return nil
	}
	delta := -1
	if isOpen {
		delta = 1
	}
	if err := do.adjustVotersQuota(id, delta); err != nil {
		return err
	}
	if existing.EntryType == models.EntryTypeFeatureRequest && existing.UserID != "" {
		if err := do.IncrementUserFeatureRequestsOpen(existing.UserID, delta); err != nil {
			return err
		}
	}
	return nil
}

// adjustVotersQuota walks the votes collection for one entry, groups
// by user, and bumps each user's VotesSpent by delta. Anonymous votes
// (UserID == "") are skipped — no quota row to adjust. Used by
// SetEntryStatus on open/closed transitions; the floor-at-zero guard
// inside IncrementUserVotesSpent handles a refund that would drive
// the counter below zero.
func (do *DataOperations) adjustVotersQuota(entryID string, delta int) error {
	if entryID == "" || delta == 0 {
		return nil
	}
	votes, err := mongodb.Query[models.Vote](do.DB, CollectionVotes, bson.M{"entryid": entryID}, nil)
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, v := range votes {
		if v.UserID == "" {
			continue
		}
		if _, dup := seen[v.UserID]; dup {
			continue
		}
		seen[v.UserID] = struct{}{}
		if err := do.IncrementUserVotesSpent(v.UserID, delta); err != nil {
			return err
		}
	}
	return nil
}

// SetEntryTypeAnalysis overwrites the entrytypeanalysis
// sub-document on a single entry. Used by the entry-type analyzer
// after a successful LLM call. Caller passes a struct with ClaimedAt
// left zero, which implicitly releases the claim and marks the entry
// done. Does NOT touch updatedat — AI fields are advisory metadata,
// not user edits.
func (do *DataOperations) SetEntryTypeAnalysis(id string, a models.EntryTypeAnalysis) error {
	return mongodb.SetValue(do.DB, CollectionEntries, id, "entrytypeanalysis", a)
}

// FailEntryTypeAnalysis stamps status=failed and the error
// message on the entrytypeanalysis sub-document, and clears
// ClaimedAt so the next worker tick can retry. Other AI fields
// (suggested type, analyzed-at) are left intact so a previously good
// analysis stays visible until the retry overwrites it.
func (do *DataOperations) FailEntryTypeAnalysis(id, errMsg string) error {
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "entrytypeanalysis.status", models.AnalysisStatusFailed); err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "entrytypeanalysis.errormessage", errMsg); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionEntries, id, "entrytypeanalysis.claimedat", time.Time{})
}

// entryNeedsEntryTypeAnalysisFilter matches entries that still need
// entry-type analysis run. Three classes of entries qualify:
//
//   - Never analyzed: status missing/empty.
//   - Failed: status=="failed" — auto-retry next tick.
//   - Stale: status=="done" but the entry's updatedat is newer than
//     entrytypeanalysis.analyzedat (the entry was edited after its
//     last analysis). $expr lets the filter compare two fields of the
//     same document.
//
// Entries with status=="disabled" are excluded outright — that stamp
// is set when the entry is created with a feedback type, or when an
// admin later assigns one. The decision is one-way: once disabled, an
// entry is never re-evaluated.
//
// Stale-detection makes the design fully poller-driven: entry-edit
// handlers don't need any AI awareness — bumping updatedat (which
// every mutation already does) is enough to re-trigger analysis.
func entryNeedsEntryTypeAnalysisFilter() bson.M {
	return bson.M{"$and": bson.A{
		bson.M{"entrytypeanalysis.status": bson.M{"$ne": models.AnalysisStatusDisabled}},
		bson.M{"$or": bson.A{
			bson.M{"entrytypeanalysis": bson.M{"$exists": false}},
			bson.M{"entrytypeanalysis.status": bson.M{"$exists": false}},
			bson.M{"entrytypeanalysis.status": ""},
			bson.M{"entrytypeanalysis.status": models.AnalysisStatusFailed},
			bson.M{"$expr": bson.M{"$lt": bson.A{"$entrytypeanalysis.analyzedat", "$updatedat"}}},
		}},
	}}
}

// claimAvailableFilter matches entries whose entry-type analysis is
// not currently claimed by any instance — claimedat is missing, zero,
// or older than now-claimTTL (the original claimer crashed or stalled).
func claimAvailableFilter(claimTTL time.Duration) bson.M {
	cutoff := time.Now().UTC().Add(-claimTTL)
	return bson.M{"$or": bson.A{
		bson.M{"entrytypeanalysis.claimedat": bson.M{"$exists": false}},
		bson.M{"entrytypeanalysis.claimedat": time.Time{}},
		bson.M{"entrytypeanalysis.claimedat": bson.M{"$lt": cutoff}},
	}}
}

// claimInFlightFilter is the inverse of claimAvailableFilter — matches
// entries currently claimed by some instance (claim within TTL).
func claimInFlightFilter(claimTTL time.Duration) bson.M {
	cutoff := time.Now().UTC().Add(-claimTTL)
	return bson.M{"entrytypeanalysis.claimedat": bson.M{"$gt": cutoff}}
}

// CountEntriesPendingEntryTypeAnalysis returns how many entries
// need entry-type analysis run AND are not currently claimed by any
// instance. Drives the "items waiting" gauge on the Console AI
// settings page.
func (do *DataOperations) CountEntriesPendingEntryTypeAnalysis(claimTTL time.Duration) (int, error) {
	filter := bson.M{"$and": bson.A{
		entryNeedsEntryTypeAnalysisFilter(),
		claimAvailableFilter(claimTTL),
	}}
	n, err := mongodb.Count(do.DB, CollectionEntries, filter, nil)
	return int(n), err
}

// CountEntriesInFlightEntryTypeAnalysis returns how many entries
// are currently claimed by some instance for entry-type analysis
// (claim is within the TTL window). Drives the "in flight" gauge.
func (do *DataOperations) CountEntriesInFlightEntryTypeAnalysis(claimTTL time.Duration) (int, error) {
	n, err := mongodb.Count(do.DB, CollectionEntries, claimInFlightFilter(claimTTL), nil)
	return int(n), err
}

// intersectReasons returns the subset of an entry's per-ID reasons
// map that still applies after a write — keyed only by IDs in
// surviving (typically the new aitopicids set). Returns an empty
// (non-nil) map when the source map is empty so Mongo persists
// `{}` rather than null.
func intersectReasons(existing *models.Entry, pick func(*models.Entry) map[string]string, surviving []string) map[string]string {
	out := map[string]string{}
	if existing == nil {
		return out
	}
	src := pick(existing)
	if len(src) == 0 || len(surviving) == 0 {
		return out
	}
	for _, id := range surviving {
		if r, ok := src[id]; ok {
			out[id] = r
		}
	}
	return out
}

// ClaimNextPendingForEntryTypeAnalysis atomically claims one entry
// that needs entry-type analysis and returns it. Returns (nil, nil)
// when nothing is available (no pending entries, or all pending are
// already claimed by other instances).
//
// Atomicity is provided by Mongo's findOneAndUpdate — two Cloud Run
// instances polling at the same instant will never claim the same
// entry. The TTL semantics (a stale claim is reclaimable after
// claimTTL elapses) handle crashed-instance recovery automatically;
// no separate sweeper job is needed.
//
// The returned entry has Status=pending and ClaimedAt=now. The caller
// (the worker) runs the analyzer's Process on it, which persists the
// final state via SetEntryTypeAnalysis (success) or
// FailEntryTypeAnalysis (failure).
func (do *DataOperations) ClaimNextPendingForEntryTypeAnalysis(claimTTL time.Duration) (*models.Entry, error) {
	filter := bson.M{"$and": bson.A{
		entryNeedsEntryTypeAnalysisFilter(),
		claimAvailableFilter(claimTTL),
	}}
	now := time.Now().UTC()
	update := bson.M{"$set": bson.M{
		"entrytypeanalysis.status":    models.AnalysisStatusPending,
		"entrytypeanalysis.claimedat": now,
	}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	return mongodb.FindOneAndUpdate[models.Entry](do.DB, CollectionEntries, filter, update, opts)
}

// SetEntryTopics overwrites the topicids array on a single entry
// and touches updatedat. Caller is responsible for validating the
// IDs against the topics collection before calling. A nil or empty
// slice is persisted as an empty array.
//
// Setting a non-empty topic list also stamps topicanalysis.status
// to "disabled" so the AI worker stops considering this entry.
// Once the human has chosen topics, the AI never overrides that
// choice.
//
// AI provenance: aitopicids is rewritten as the intersection of its
// prior value with the new topicIDs. Topics the admin keeps
// (originally AI-applied) stay flagged; topics the admin removes
// drop from both arrays; topics the admin adds are NOT AI-flagged.
// Per-topic reasons (topicanalysis.suggestedtopicreasons) are
// intersected in lockstep with aitopicids: a reason key survives
// only when its topic is still AI-flagged.
func (do *DataOperations) SetEntryTopics(id string, topicIDs []string) error {
	if topicIDs == nil {
		topicIDs = []string{}
	}
	existing, err := do.FindEntryByID(id)
	if err != nil {
		return err
	}
	aiTopicIDs := []string{}
	if existing != nil && len(existing.AITopicIDs) > 0 && len(topicIDs) > 0 {
		keep := make(map[string]struct{}, len(topicIDs))
		for _, t := range topicIDs {
			keep[t] = struct{}{}
		}
		for _, t := range existing.AITopicIDs {
			if _, ok := keep[t]; ok {
				aiTopicIDs = append(aiTopicIDs, t)
			}
		}
	}
	reasons := intersectReasons(existing, func(e *models.Entry) map[string]string {
		return e.TopicAnalysis.SuggestedTopicReasons
	}, aiTopicIDs)
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "topicids", topicIDs); err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "aitopicids", aiTopicIDs); err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "topicanalysis.suggestedtopicreasons", reasons); err != nil {
		return err
	}
	if len(topicIDs) > 0 {
		if err := mongodb.SetValue(do.DB, CollectionEntries, id, "topicanalysis.status", models.AnalysisStatusDisabled); err != nil {
			return err
		}
	}
	return mongodb.SetValue(do.DB, CollectionEntries, id, "updatedat", time.Now().UTC())
}

// SetEntryTopicsFromAnalyzer writes the topicids array AND the
// aitopicids AI-provenance subset on an entry, without touching
// updatedat and without stamping topicanalysis.status to
// "disabled". Used by the topic analyzer's auto-apply path; the
// analyzer pairs this with a SetEntryTopicAnalysis(status=done)
// write to record the audit trail.
//
// aitopicids is set to the same value as topicids: every ID the
// analyzer applies is an AI assignment. Subsequent admin edits go
// through SetEntryTopics, which intersects the prior AI subset with
// the new list.
//
// Skipping updatedat is deliberate: bumping it here would re-fire
// the category analyzer's stale-edit check on every topic-analyzer
// run.
func (do *DataOperations) SetEntryTopicsFromAnalyzer(id string, topicIDs []string) error {
	if topicIDs == nil {
		topicIDs = []string{}
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "topicids", topicIDs); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionEntries, id, "aitopicids", topicIDs)
}

// SetEntryTopicAnalysis overwrites the topicanalysis sub-document
// on a single entry. Used by the topic analyzer after a successful
// LLM call. Caller passes a struct with ClaimedAt left zero, which
// implicitly releases the claim and marks the entry done. Does NOT
// touch updatedat — AI fields are advisory metadata, not user
// edits.
func (do *DataOperations) SetEntryTopicAnalysis(id string, a models.EntryTopicAnalysis) error {
	return mongodb.SetValue(do.DB, CollectionEntries, id, "topicanalysis", a)
}

// FailEntryTopicAnalysis stamps status=failed and the error message
// on the topicanalysis sub-document, and clears ClaimedAt so the
// next worker tick can retry.
func (do *DataOperations) FailEntryTopicAnalysis(id, errMsg string) error {
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "topicanalysis.status", models.AnalysisStatusFailed); err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "topicanalysis.errormessage", errMsg); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionEntries, id, "topicanalysis.claimedat", time.Time{})
}

// entryNeedsTopicAnalysisFilter matches entries that still need
// topic analysis run. One-shot at creation, never re-runs after
// admin clears topics.
func entryNeedsTopicAnalysisFilter() bson.M {
	return bson.M{"$and": bson.A{
		bson.M{"topicanalysis.status": bson.M{"$ne": models.AnalysisStatusDisabled}},
		bson.M{"topicanalysis.status": bson.M{"$ne": models.AnalysisStatusDone}},
		bson.M{"$or": bson.A{
			bson.M{"topicids": bson.M{"$exists": false}},
			bson.M{"topicids": bson.M{"$size": 0}},
		}},
	}}
}

// topicClaimAvailableFilter matches entries whose topic analysis is
// not currently claimed by any instance.
func topicClaimAvailableFilter(claimTTL time.Duration) bson.M {
	cutoff := time.Now().UTC().Add(-claimTTL)
	return bson.M{"$or": bson.A{
		bson.M{"topicanalysis.claimedat": bson.M{"$exists": false}},
		bson.M{"topicanalysis.claimedat": time.Time{}},
		bson.M{"topicanalysis.claimedat": bson.M{"$lt": cutoff}},
	}}
}

// topicClaimInFlightFilter matches entries currently claimed by
// some instance for topic analysis.
func topicClaimInFlightFilter(claimTTL time.Duration) bson.M {
	cutoff := time.Now().UTC().Add(-claimTTL)
	return bson.M{"topicanalysis.claimedat": bson.M{"$gt": cutoff}}
}

// CountEntriesPendingTopicAnalysis returns how many entries need
// topic analysis run AND are not currently claimed.
func (do *DataOperations) CountEntriesPendingTopicAnalysis(claimTTL time.Duration) (int, error) {
	filter := bson.M{"$and": bson.A{
		entryNeedsTopicAnalysisFilter(),
		topicClaimAvailableFilter(claimTTL),
	}}
	n, err := mongodb.Count(do.DB, CollectionEntries, filter, nil)
	return int(n), err
}

// CountEntriesInFlightTopicAnalysis returns how many entries are
// currently claimed by some instance for topic analysis.
func (do *DataOperations) CountEntriesInFlightTopicAnalysis(claimTTL time.Duration) (int, error) {
	n, err := mongodb.Count(do.DB, CollectionEntries, topicClaimInFlightFilter(claimTTL), nil)
	return int(n), err
}

// ClaimNextPendingForTopicAnalysis atomically claims one entry that
// needs topic analysis and returns it. Returns (nil, nil) when
// nothing is available.
func (do *DataOperations) ClaimNextPendingForTopicAnalysis(claimTTL time.Duration) (*models.Entry, error) {
	filter := bson.M{"$and": bson.A{
		entryNeedsTopicAnalysisFilter(),
		topicClaimAvailableFilter(claimTTL),
	}}
	now := time.Now().UTC()
	update := bson.M{"$set": bson.M{
		"topicanalysis.status":    models.AnalysisStatusPending,
		"topicanalysis.claimedat": now,
	}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	return mongodb.FindOneAndUpdate[models.Entry](do.DB, CollectionEntries, filter, update, opts)
}

// AddEntryRelation upserts a single relation pair on entryID and
// mirrors the symmetric pair onto peerEntryID. Idempotent — re-applying
// the same (peerEntryID, type) is a no-op.
//
// Stamps relationanalysis.status to "disabled" on entryID so the AI
// worker stops considering it (mirrors the topic one-way disable).
// Touches updatedat on entryID. Caller is responsible for validating
// the type via models.IsValidEntryRelationType and that both entries
// exist before calling; entryID == peerEntryID is rejected as a
// no-op.
func (do *DataOperations) AddEntryRelation(entryID, peerEntryID string, relType models.EntryRelationType) error {
	if entryID == peerEntryID {
		return nil
	}
	rel := models.EntryRelation{EntryID: peerEntryID, Type: relType}
	if err := do.upsertRelationOnEntry(entryID, rel, false); err != nil {
		return err
	}
	mirror := models.EntryRelation{EntryID: entryID, Type: relType}
	if err := do.upsertRelationOnEntry(peerEntryID, mirror, false); err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, entryID, "relationanalysis.status", models.AnalysisStatusDisabled); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionEntries, entryID, "updatedat", time.Now().UTC())
}

// RemoveEntryRelation pulls every relation between entryID and
// peerEntryID from both entries' relations and airelations arrays.
// Idempotent — no-op when the pair isn't currently linked. Touches
// updatedat on entryID.
//
// Also unsets the per-peer reason key on each side's
// relationanalysis.suggestedrelationreasons so a stale tooltip never
// outlives the relation it justified.
func (do *DataOperations) RemoveEntryRelation(entryID, peerEntryID string) error {
	if entryID == peerEntryID {
		return nil
	}
	if err := do.removeRelationFromEntry(entryID, peerEntryID); err != nil {
		return err
	}
	if err := do.removeRelationFromEntry(peerEntryID, entryID); err != nil {
		return err
	}
	if err := do.unsetRelationReason(entryID, peerEntryID); err != nil {
		return err
	}
	if err := do.unsetRelationReason(peerEntryID, entryID); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionEntries, entryID, "updatedat", time.Now().UTC())
}

// unsetRelationReason removes a single peer's key from
// entryID.relationanalysis.suggestedrelationreasons. UUIDs are safe
// as Mongo dotted-path field names — they contain no `.` or `$`.
func (do *DataOperations) unsetRelationReason(entryID, peerEntryID string) error {
	if entryID == "" || peerEntryID == "" {
		return nil
	}
	model := mongo.NewUpdateOneModel().
		SetFilter(bson.M{"_id": entryID}).
		SetUpdate(bson.M{"$unset": bson.M{
			"relationanalysis.suggestedrelationreasons." + peerEntryID: "",
		}})
	_, err := mongodb.BulkUpdate(do.DB, CollectionEntries, []mongo.WriteModel{model})
	return err
}

// SetEntryRelations overwrites the relations array on a single entry,
// mirrors the change onto every affected peer's relations / airelations
// arrays, intersects the prior airelations subset on entryID with the
// new (peerID, type) pairs, and touches updatedat.
//
// Mirror semantics:
//   - Peers that disappear from the new list have (entryID, *) pulled
//     from their relations AND airelations.
//   - Peers added or whose type changed get (entryID, type) upserted on
//     their relations array (replacing any prior pair to entryID).
//     Because this is an admin action, the mirrored pair is also
//     dropped from the peer's airelations — admin edits override AI
//     provenance on the peer side.
//   - Peers whose (peerID, type) is unchanged still get any
//     (entryID, *) pulled from their airelations (admin override).
//
// Setting a non-empty relation list also stamps relationanalysis.status
// to "disabled" so the AI worker stops considering this entry.
//
// Caller is responsible for validating relations (peer IDs exist,
// types are valid via models.IsValidEntryRelationType, no self-relation,
// no duplicates by peerID) before calling.
func (do *DataOperations) SetEntryRelations(entryID string, relations []models.EntryRelation) error {
	if relations == nil {
		relations = []models.EntryRelation{}
	}
	existing, err := do.FindEntryByID(entryID)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}

	aiRelations := []models.EntryRelation{}
	if len(existing.AIRelations) > 0 && len(relations) > 0 {
		keep := make(map[string]struct{}, len(relations))
		for _, r := range relations {
			keep[r.EntryID+"|"+string(r.Type)] = struct{}{}
		}
		for _, r := range existing.AIRelations {
			if _, ok := keep[r.EntryID+"|"+string(r.Type)]; ok {
				aiRelations = append(aiRelations, r)
			}
		}
	}
	survivingPeerIDs := make([]string, 0, len(aiRelations))
	for _, r := range aiRelations {
		survivingPeerIDs = append(survivingPeerIDs, r.EntryID)
	}
	reasons := intersectReasons(existing, func(e *models.Entry) map[string]string {
		return e.RelationAnalysis.SuggestedRelationReasons
	}, survivingPeerIDs)

	if err := do.applyRelationMirror(entryID, existing.Relations, relations, false); err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, entryID, "relations", relations); err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, entryID, "airelations", aiRelations); err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, entryID, "relationanalysis.suggestedrelationreasons", reasons); err != nil {
		return err
	}
	if len(relations) > 0 {
		if err := mongodb.SetValue(do.DB, CollectionEntries, entryID, "relationanalysis.status", models.AnalysisStatusDisabled); err != nil {
			return err
		}
	}
	return mongodb.SetValue(do.DB, CollectionEntries, entryID, "updatedat", time.Now().UTC())
}

// SetEntryRelationsFromAnalyzer writes the relations array AND the
// airelations AI-provenance subset on an entry, mirrors the change
// onto every affected peer (treating the mirrored pair as AI on the
// peer's airelations too), without touching updatedat and without
// stamping relationanalysis.status to "disabled".
//
// Skipping updatedat is deliberate: bumping it would re-fire the
// category analyzer's stale-edit check on every relation-analyzer
// run.
func (do *DataOperations) SetEntryRelationsFromAnalyzer(entryID string, relations []models.EntryRelation) error {
	if relations == nil {
		relations = []models.EntryRelation{}
	}
	existing, err := do.FindEntryByID(entryID)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}
	if err := do.applyRelationMirror(entryID, existing.Relations, relations, true); err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, entryID, "relations", relations); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionEntries, entryID, "airelations", relations)
}

// applyRelationMirror reconciles every peer's relations / airelations
// arrays so they reflect the change on entryID. additionsAreAI controls
// peer-side provenance for new pairs:
//   - false (admin write): mirrored pair lands on peer.relations only;
//     any prior (entryID, *) is scrubbed from peer.airelations.
//   - true  (analyzer write): mirrored pair lands on both peer.relations
//     and peer.airelations.
//
// Peers that disappear from the new list have (entryID, *) pulled from
// both arrays regardless of additionsAreAI.
func (do *DataOperations) applyRelationMirror(entryID string, oldRels, newRels []models.EntryRelation, additionsAreAI bool) error {
	oldByPeer := map[string]models.EntryRelationType{}
	for _, r := range oldRels {
		oldByPeer[r.EntryID] = r.Type
	}
	newByPeer := map[string]models.EntryRelationType{}
	for _, r := range newRels {
		newByPeer[r.EntryID] = r.Type
	}
	for peerID := range oldByPeer {
		if _, kept := newByPeer[peerID]; kept {
			continue
		}
		if err := do.removeRelationFromEntry(peerID, entryID); err != nil {
			return err
		}
	}
	for peerID, t := range newByPeer {
		mirrored := models.EntryRelation{EntryID: entryID, Type: t}
		if oldType, ok := oldByPeer[peerID]; ok && oldType == t {
			if !additionsAreAI {
				if err := do.scrubRelationFromAI(peerID, entryID); err != nil {
					return err
				}
			}
			continue
		}
		if err := do.upsertRelationOnEntry(peerID, mirrored, additionsAreAI); err != nil {
			return err
		}
	}
	return nil
}

// removeRelationFromEntry pulls every relation to peerEntryID from
// entryID's relations and airelations arrays, and unsets the matching
// reason key so the lockstep invariant (a reason exists only while its
// peer is still AI-flagged) holds. No-op when the entry is missing or
// carries no such relation.
func (do *DataOperations) removeRelationFromEntry(entryID, peerEntryID string) error {
	e, err := do.FindEntryByID(entryID)
	if err != nil {
		return err
	}
	if e == nil {
		return nil
	}
	rels := filterRelationsExcluding(e.Relations, peerEntryID)
	ais := filterRelationsExcluding(e.AIRelations, peerEntryID)
	if len(rels) == len(e.Relations) && len(ais) == len(e.AIRelations) {
		return nil
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, entryID, "relations", rels); err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, entryID, "airelations", ais); err != nil {
		return err
	}
	return do.unsetRelationReason(entryID, peerEntryID)
}

// upsertRelationOnEntry writes rel onto entryID's relations array,
// replacing any prior pair to rel.EntryID. When isAI is true, the same
// pair is also written to the airelations subset; when false, any
// prior airelations entry to rel.EntryID is dropped (admin override),
// AND the matching reason key is unset so the reasons map never points
// at a peer that's no longer AI-flagged.
func (do *DataOperations) upsertRelationOnEntry(entryID string, rel models.EntryRelation, isAI bool) error {
	e, err := do.FindEntryByID(entryID)
	if err != nil {
		return err
	}
	if e == nil {
		return nil
	}
	rels := upsertRelationByPeer(e.Relations, rel)
	var ais []models.EntryRelation
	if isAI {
		ais = upsertRelationByPeer(e.AIRelations, rel)
	} else {
		ais = filterRelationsExcluding(e.AIRelations, rel.EntryID)
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, entryID, "relations", rels); err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, entryID, "airelations", ais); err != nil {
		return err
	}
	if !isAI {
		return do.unsetRelationReason(entryID, rel.EntryID)
	}
	return nil
}

// scrubRelationFromAI removes any (peerEntryID, *) entry from
// entryID's airelations array, leaving relations untouched, and
// unsets the matching reason key. Used on the unchanged-pair branch
// of admin mirror writes.
func (do *DataOperations) scrubRelationFromAI(entryID, peerEntryID string) error {
	e, err := do.FindEntryByID(entryID)
	if err != nil {
		return err
	}
	if e == nil {
		return nil
	}
	ais := filterRelationsExcluding(e.AIRelations, peerEntryID)
	if len(ais) == len(e.AIRelations) {
		return nil
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, entryID, "airelations", ais); err != nil {
		return err
	}
	return do.unsetRelationReason(entryID, peerEntryID)
}

// filterRelationsExcluding returns a new slice with every relation
// targeting peerEntryID removed. Returns an empty (non-nil) slice when
// the input is empty so Mongo persists [] rather than null.
func filterRelationsExcluding(rels []models.EntryRelation, peerEntryID string) []models.EntryRelation {
	out := make([]models.EntryRelation, 0, len(rels))
	for _, r := range rels {
		if r.EntryID != peerEntryID {
			out = append(out, r)
		}
	}
	return out
}

// upsertRelationByPeer returns a copy of rels with rel either
// appended (no prior entry to rel.EntryID) or in-place replacing the
// prior entry to rel.EntryID. Caller-side dedupe by peer.
func upsertRelationByPeer(rels []models.EntryRelation, rel models.EntryRelation) []models.EntryRelation {
	out := make([]models.EntryRelation, 0, len(rels)+1)
	replaced := false
	for _, r := range rels {
		if r.EntryID == rel.EntryID {
			out = append(out, rel)
			replaced = true
		} else {
			out = append(out, r)
		}
	}
	if !replaced {
		out = append(out, rel)
	}
	return out
}

// SetEntryRelationAnalysis overwrites the relationanalysis sub-document
// on a single entry. Used by the relation analyzer after a successful
// LLM call. Caller passes a struct with ClaimedAt left zero, which
// implicitly releases the claim and marks the entry done. Does NOT
// touch updatedat — AI fields are advisory metadata, not user edits.
func (do *DataOperations) SetEntryRelationAnalysis(id string, a models.EntryRelationAnalysis) error {
	return mongodb.SetValue(do.DB, CollectionEntries, id, "relationanalysis", a)
}

// FailEntryRelationAnalysis stamps status=failed and the error message
// on the relationanalysis sub-document, and clears ClaimedAt so the
// next worker tick can retry. Other AI fields are left intact so a
// previously good analysis stays visible until the retry overwrites
// it.
func (do *DataOperations) FailEntryRelationAnalysis(id, errMsg string) error {
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "relationanalysis.status", models.AnalysisStatusFailed); err != nil {
		return err
	}
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "relationanalysis.errormessage", errMsg); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionEntries, id, "relationanalysis.claimedat", time.Time{})
}

// entryNeedsRelationAnalysisFilter matches entries that still need
// relation analysis run. One-shot at creation, never re-runs after
// admin clears relations.
func entryNeedsRelationAnalysisFilter() bson.M {
	return bson.M{"$and": bson.A{
		bson.M{"relationanalysis.status": bson.M{"$ne": models.AnalysisStatusDisabled}},
		bson.M{"relationanalysis.status": bson.M{"$ne": models.AnalysisStatusDone}},
		bson.M{"$or": bson.A{
			bson.M{"relations": bson.M{"$exists": false}},
			bson.M{"relations": bson.M{"$size": 0}},
		}},
	}}
}

// relationClaimAvailableFilter matches entries whose relation analysis
// is not currently claimed by any instance.
func relationClaimAvailableFilter(claimTTL time.Duration) bson.M {
	cutoff := time.Now().UTC().Add(-claimTTL)
	return bson.M{"$or": bson.A{
		bson.M{"relationanalysis.claimedat": bson.M{"$exists": false}},
		bson.M{"relationanalysis.claimedat": time.Time{}},
		bson.M{"relationanalysis.claimedat": bson.M{"$lt": cutoff}},
	}}
}

// relationClaimInFlightFilter matches entries currently claimed by
// some instance for relation analysis.
func relationClaimInFlightFilter(claimTTL time.Duration) bson.M {
	cutoff := time.Now().UTC().Add(-claimTTL)
	return bson.M{"relationanalysis.claimedat": bson.M{"$gt": cutoff}}
}

// CountEntriesPendingRelationAnalysis returns how many entries need
// relation analysis run AND are not currently claimed.
func (do *DataOperations) CountEntriesPendingRelationAnalysis(claimTTL time.Duration) (int, error) {
	filter := bson.M{"$and": bson.A{
		entryNeedsRelationAnalysisFilter(),
		relationClaimAvailableFilter(claimTTL),
	}}
	n, err := mongodb.Count(do.DB, CollectionEntries, filter, nil)
	return int(n), err
}

// CountEntriesInFlightRelationAnalysis returns how many entries are
// currently claimed by some instance for relation analysis.
func (do *DataOperations) CountEntriesInFlightRelationAnalysis(claimTTL time.Duration) (int, error) {
	n, err := mongodb.Count(do.DB, CollectionEntries, relationClaimInFlightFilter(claimTTL), nil)
	return int(n), err
}

// ClaimNextPendingForRelationAnalysis atomically claims one entry that
// needs relation analysis and returns it. Returns (nil, nil) when
// nothing is available.
func (do *DataOperations) ClaimNextPendingForRelationAnalysis(claimTTL time.Duration) (*models.Entry, error) {
	filter := bson.M{"$and": bson.A{
		entryNeedsRelationAnalysisFilter(),
		relationClaimAvailableFilter(claimTTL),
	}}
	now := time.Now().UTC()
	update := bson.M{"$set": bson.M{
		"relationanalysis.status":    models.AnalysisStatusPending,
		"relationanalysis.claimedat": now,
	}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	return mongodb.FindOneAndUpdate[models.Entry](do.DB, CollectionEntries, filter, update, opts)
}

// ListEntriesForRelationAnalysis returns every existing entry except
// the one given. Used by the relation analyzer to enumerate the
// candidate pool against which to compare a freshly-created entry.
//
// v0.1 returns the full set unfiltered — pilot data is small enough.
// When vector search lands in v0.2 this becomes a similarity-narrowed
// subset.
func (do *DataOperations) ListEntriesForRelationAnalysis(excludeID string) ([]models.Entry, error) {
	filter := bson.M{"_id": bson.M{"$ne": excludeID}}
	opts := options.Find().SetSort(bson.D{{Key: "createdat", Value: -1}})
	return mongodb.Query[models.Entry](do.DB, CollectionEntries, filter, opts)
}

// ListEntriesForSimilarity returns the candidate pool for the
// create-entry similarity finder. Used by the synchronous "find
// similar entries" endpoint on both Portal and Console.
//
// publicOnly=true narrows to non-internal entries — Portal callers
// must not see internal records. publicOnly=false returns the full
// set (Console). Newest first, like the relation analyzer pool.
//
// Pilot dataset is small enough to enumerate inline. When entries
// grow into the thousands a text-search prefilter or vector search
// (v0.2) replaces this method.
func (do *DataOperations) ListEntriesForSimilarity(publicOnly bool) ([]models.Entry, error) {
	filter := bson.M{}
	if publicOnly {
		filter = bson.M{"isinternal": false}
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdat", Value: -1}})
	return mongodb.Query[models.Entry](do.DB, CollectionEntries, filter, opts)
}
