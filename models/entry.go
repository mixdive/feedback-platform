package models

import (
	"time"

	"github.com/google/uuid"
)

// EntryType groups entries by user-visible kind. Hardcoded enum —
// admins can't add or rename values. Empty string is also valid and
// means "untyped" (new submission with no type picked, or the AI
// analyzer hasn't run yet). Frontends mirror this list inline; the
// Go constants are the source of truth.
type EntryType string

const (
	EntryTypeFeatureRequest EntryType = "feature-request"
	EntryTypeBug            EntryType = "bug"
	EntryTypeSupport        EntryType = "support"
	EntryTypeOther          EntryType = "other"
)

// IsValidEntryType reports whether v is a known entry type value.
// The empty string is treated as valid (untyped) — caller decides
// whether to require a non-empty value.
func IsValidEntryType(v EntryType) bool {
	switch v {
	case "", EntryTypeFeatureRequest, EntryTypeBug, EntryTypeSupport, EntryTypeOther:
		return true
	}
	return false
}

// EntryStatus is the workflow stage of an entry. Hardcoded enum — admins
// can't add or rename values. Frontends mirror this list inline; the Go
// constants are the source of truth.
type EntryStatus string

const (
	EntryStatusNew        EntryStatus = "new"
	EntryStatusEvaluation EntryStatus = "evaluation"
	EntryStatusInProgress EntryStatus = "in-progress"
	EntryStatusCompleted  EntryStatus = "completed"
	EntryStatusCancelled  EntryStatus = "cancelled"
)

// EntryStatusDefault is the status assigned to every freshly-submitted
// entry and to any entry whose stored value is empty/unknown.
const EntryStatusDefault = EntryStatusNew

// EntryStatusCancelTarget is where merged-away entries land. Cancelled
// is the convention — it tells the admin reading the merge audit trail
// that the entry was abandoned, not shipped.
const EntryStatusCancelTarget = EntryStatusCancelled

// IsValidEntryStatus reports whether v is a known status value.
// Handlers reject unknown values rather than persist them.
func IsValidEntryStatus(v EntryStatus) bool {
	switch v {
	case EntryStatusNew, EntryStatusEvaluation, EntryStatusInProgress, EntryStatusCompleted, EntryStatusCancelled:
		return true
	}
	return false
}

// IsEntryStatusOpen reports whether v counts as an "open" workflow
// stage for vote-quota purposes — votes on open entries consume the
// per-user quota; votes on closed entries (completed/cancelled) do
// not. Empty / unknown values are treated as open so a missing field
// on a legacy entry doesn't accidentally exempt votes from the cap.
func IsEntryStatusOpen(v EntryStatus) bool {
	switch v {
	case EntryStatusCompleted, EntryStatusCancelled:
		return false
	}
	return true
}

// EntrySourceType records which surface created an entry record.
// Defined as a named string so handlers can't accidentally pass an
// arbitrary literal.
type EntrySourceType string

const (
	EntrySourcePortal EntrySourceType = "portal"
)

// EntryRelationType is the kind of link between two entries. Locked to
// a Go enum in v0.1 — admins cannot define new types. Default behavior
// when no specific type fits is EntryRelationTypeRelated; the AI
// analyzer falls back to it as a catch-all.
type EntryRelationType string

const (
	EntryRelationTypeDuplicate EntryRelationType = "duplicate"
	EntryRelationTypeRelated   EntryRelationType = "related"
)

// IsValidEntryRelationType reports whether v is a known relation type.
// Handlers and the analyzer should reject unknown values rather than
// persist them.
func IsValidEntryRelationType(v EntryRelationType) bool {
	switch v {
	case EntryRelationTypeDuplicate, EntryRelationTypeRelated:
		return true
	}
	return false
}

// EntryRelation is one half of a symmetric link between two entries.
// EntryID is the peer entry's ID — the entry on which this struct is
// stored is implicitly the other side. The pair is mirrored on the
// peer entry so a Console reader can pull all relations from a single
// document without a join.
//
// Uniqueness within Entry.Relations is by EntryID: at most one
// relation per peer. Re-applying a relation to the same peer overwrites
// the existing Type rather than appending.
type EntryRelation struct {
	EntryID string
	Type    EntryRelationType
}

// GitHubIssue is the per-entry record of a created GitHub issue. Zero
// value (Number == 0, URL == "") reads as "no issue linked"; once
// populated, the link is forever — v0.1 has no unlink flow.
//
// Owner / Repo / Number are stored alongside the full URL so we never
// have to parse the URL to round-trip back to the GitHub API. CreatedBy
// is the admin user ID who clicked the Console button.
type GitHubIssue struct {
	URL       string
	Owner     string
	Repo      string
	Number    int
	CreatedAt time.Time
	CreatedBy string
}

// Entry is the minimum-viable user-submitted record: a title + description
// with a per-entry vote counter.
//
// UserID points at the User who created the entry. Empty string means
// the record was created anonymously (Portal write with no session and
// custom auth disabled).
//
// EntryType discriminates the kind of entry. Today every record is
// EntryTypeFeedback; future ingestion paths (tweets, app-store reviews,
// etc.) will introduce new values without breaking the existing shape.
//
// Source captures which surface created the entry (portal / console).
// IsInternal hides an entry from the Portal when true; Portal-created
// records default to false, Console-created records default to true, and
// admins can flip the flag from the Console.
//
// AITopicIDs / AIRelations are the per-assignment AI provenance
// subsets: the IDs (or relation pairs) in TopicIDs / Relations that
// were applied by an AI analyzer rather than by an admin. Maintained by:
//   - the topic/relation analyzers, which mirror their result
//     into both the main slice and the AI subset on auto-apply;
//   - SetEntryTopics / SetEntryRelations, which intersect the prior
//     AI subset with the incoming list so a manually-kept AI
//     assignment stays flagged, a manually-removed one is dropped
//     from both, and a manually-added one is never AI-flagged;
//   - RemoveTopicIDFromAllEntries / RemoveEntryFromAllRelations,
//     which pull the deleted ID from both arrays during cascade.
//
// Relations carry an extra wrinkle vs topics: each link is mirrored
// on the peer entry, so writers must update two documents (and the
// cascade-on-archive pass scrubs the deleted entry's ID from every
// peer's Relations / AIRelations).
//
// Empty/missing on existing documents reads as "all assignments were
// manual" — no backfill required.
type Entry struct {
	ID                string `bson:"_id"`
	UserID            string
	EntryType         EntryType
	Status            EntryStatus
	TopicIDs          []string
	AITopicIDs        []string
	Relations         []EntryRelation
	AIRelations       []EntryRelation
	Title             string
	Description       string
	VoteCount         int
	CommentCount      int
	Source            EntrySourceType
	IsInternal        bool
	ReleaseID         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	GitHubIssue       GitHubIssue
	EntryTypeAnalysis EntryTypeAnalysis
	TopicAnalysis     EntryTopicAnalysis
	RelationAnalysis  EntryRelationAnalysis
}

// NewEntry constructs an Entry with a fresh ID and current timestamps.
// EntryType is left empty so the AI analyzer (or the submitter's
// explicit pick) sets it.
//
// EntryTypeAnalysis, TopicAnalysis and RelationAnalysis are
// intentionally left at their zero values — Status="" reads as
// "not yet analyzed" for each. The AI dispatcher stamps Status to
// "pending" only after the entry is successfully inserted and only
// when AI is enabled in Settings, avoiding a race where a Settings
// flip ships a misleading "pending" on an entry that will never
// actually be queued.
//
// TopicIDs, AITopicIDs, Relations and AIRelations are initialized
// to empty slices (not nil) so Mongo persists them as empty arrays
// and the analyzer "is the X set empty?" filters have a single
// shape to match.
func NewEntry() *Entry {
	now := time.Now().UTC()
	return &Entry{
		ID:          uuid.New().String(),
		Status:      EntryStatusDefault,
		TopicIDs:    []string{},
		AITopicIDs:  []string{},
		Relations:   []EntryRelation{},
		AIRelations: []EntryRelation{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// AnalysisStatus is the lifecycle stamp on a per-analyzer sub-struct
// embedded on Entry. Stored kebab-case per project convention. Zero
// value ("") reads as "never dispatched" — covers both pre-existing
// entries written before the analyzer existed and entries created while
// AI was disabled, with no migration.
type AnalysisStatus string

const (
	AnalysisStatusPending  AnalysisStatus = "pending"
	AnalysisStatusDone     AnalysisStatus = "done"
	AnalysisStatusFailed   AnalysisStatus = "failed"
	AnalysisStatusDisabled AnalysisStatus = "disabled"
)

// EntryTypeAnalysis is the result of the entry-type analyzer.
// Each future analyzer adds its own sibling sub-struct in the same
// shape: lifecycle fields (Status / Model / AnalyzedAt / ClaimedAt /
// ErrorMessage) plus its own result fields. Missing fields read as zero
// values, so a new analyzer landing in a future release does not
// require backfilling existing Entry documents.
//
// Advisory only: the analyzer never overwrites Entry.EntryType. The
// Console UI surfaces "Suggested: Bug" with an Apply button that goes
// through the existing SetEntryType path.
//
// ClaimedAt is the distributed-lock timestamp the worker stamps when it
// atomically claims an entry for processing. Combined with a TTL, this
// makes the analyzer pipeline safe across multiple Cloud Run instances:
// findOneAndUpdate is atomic across the cluster, so two pollers will
// never claim the same entry; if a claim's instance dies, the TTL lets
// the survivor reclaim. Zero value (or older than now-TTL) means the
// entry is free to be claimed.
type EntryTypeAnalysis struct {
	Status                   AnalysisStatus
	Model                    string
	AnalyzedAt               time.Time
	ClaimedAt                time.Time
	SuggestedEntryTypeID     string
	SuggestedEntryTypeReason string
	ErrorMessage             string
}

// EntryTopicAnalysis is the result of the topic analyzer. Mirrors
// EntryTypeAnalysis (lifecycle fields plus a result field).
//
// Authoritative — not advisory. The topic analyzer writes its
// result directly to Entry.TopicIDs in addition to
// SuggestedTopicIDs. Trigger condition is one-shot at creation: the
// worker only claims entries where TopicIDs is empty AND the
// analysis has never run (or previously failed). Once status is
// "done" or "disabled", the analyzer never re-runs.
//
// Unlike the tag analyzer, the topic analyzer is allowed to create
// a brand-new EntryTopic when no configured topic fits — an admin
// is not required to seed the topic list before AI is useful.
// SuggestedTopicReasons holds one short justification per AI-applied
// topic, keyed by topic ID. The map mirrors the live AI subset on
// Entry.AITopicIDs: SetEntryTopics drops keys whose topic was manually
// removed, and the cascade in RemoveTopicIDFromAllEntries unsets the
// dotted key path so no orphan reasons survive a topic delete. v0.1
// only applies a single topic per entry, but the map shape extends
// naturally when multi-topic AI assignment lands.
type EntryTopicAnalysis struct {
	Status                AnalysisStatus
	Model                 string
	AnalyzedAt            time.Time
	ClaimedAt             time.Time
	SuggestedTopicIDs     []string
	SuggestedTopicReasons map[string]string
	ErrorMessage          string
}

// EntryRelationAnalysis is the result of the relation analyzer.
// Mirrors EntryTopicAnalysis (lifecycle fields plus a result field) but
// the result is a list of EntryRelation pairs (peer ID + type) rather
// than bare IDs.
//
// Authoritative — the analyzer writes its result directly to
// Entry.Relations and to the peer entries' Relations arrays in
// addition to SuggestedRelations. Trigger condition is one-shot at
// creation: the worker only claims entries where Relations is empty
// AND the analysis has never run (or previously failed). Once status
// is "done" or "disabled", the analyzer never re-runs, even if an
// admin later clears every relation — keep human edits final.
//
// When the LLM finds a candidate but no specific type fits, the
// analyzer falls back to EntryRelationTypeRelated rather than
// dropping the match.
// SuggestedRelationReasons holds one short justification per
// AI-applied relation, keyed by peer entry ID. Stored only on the
// source entry — the analyzer that ran is the authoritative narrator.
// A peer entry's mirrored AIRelation entry has no reason field;
// Console UI rendering on the peer side must look up the source
// entry's analysis to surface the reason. SetEntryRelations drops
// keys whose peer left AIRelations.
type EntryRelationAnalysis struct {
	Status                   AnalysisStatus
	Model                    string
	AnalyzedAt               time.Time
	ClaimedAt                time.Time
	SuggestedRelations       []EntryRelation
	SuggestedRelationReasons map[string]string
	ErrorMessage             string
}
