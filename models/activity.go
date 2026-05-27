package models

import (
	"time"

	"github.com/google/uuid"
)

// ActivityType is the kebab-case discriminator for a single entry of
// the audit log. New values can be added without migration — older
// rows keep their existing type and Console renderers fall back to a
// generic "did something" line when the type is unknown.
type ActivityType string

const (
	ActivityTypeEntryCreated     ActivityType = "entry-created"
	ActivityTypeStatusChanged    ActivityType = "status-changed"
	ActivityTypeEntryTypeChanged ActivityType = "entry-type-changed"
	ActivityTypeTopicAdded       ActivityType = "topic-added"
	ActivityTypeTopicRemoved     ActivityType = "topic-removed"
	ActivityTypeReleaseSet       ActivityType = "release-set"
	ActivityTypeReleaseCleared   ActivityType = "release-cleared"
	ActivityTypeRelationAdded    ActivityType = "relation-added"
	ActivityTypeRelationRemoved  ActivityType = "relation-removed"
	ActivityTypeInternalEnabled  ActivityType = "internal-enabled"
	ActivityTypeInternalDisabled ActivityType = "internal-disabled"
	ActivityTypeMergedInto       ActivityType = "merged-into"
	// ActivityTypeGitHubIssueCreated is logged when an admin clicks
	// "Create GitHub issue" on a feature-request or bug entry detail.
	// TargetID holds the created issue's full URL so renderers don't
	// have to read the Entry to render a clickable timeline row; the
	// Entry.GitHubIssue sub-doc remains the source of truth.
	ActivityTypeGitHubIssueCreated ActivityType = "github-issue-created"
)

// ActivitySourceType records who originated the activity. Mirrors the
// AI provenance pattern used elsewhere (EntryTopic.Source) so the
// Console can render an "AI" badge purely from this field, without a
// synthetic user record. "user" is the portal author (entry-created
// for portal submissions); "admin" is any console-access user
// performing triage; "ai" is one of the analyzers.
type ActivitySourceType string

const (
	ActivitySourceUser  ActivitySourceType = "user"
	ActivitySourceAdmin ActivitySourceType = "admin"
	ActivitySourceAI    ActivitySourceType = "ai"
)

// Activity is one row in the per-entry timeline that the Console
// surfaces under the entry description, interleaved with comments by
// CreatedAt. The Portal never sees activities — they are Console-only
// triage signal, same rule as Entry.IsInternal.
//
// Source + ActorID together identify the originator. ActorID is empty
// when Source=="ai" (we explicitly avoid a pseudo-user — see
// [feedback_system_comments_via_acting_user]) or when an anonymous
// portal visitor created the entry.
//
// FromValue / ToValue carry the before/after snapshot for transitions
// (status, entry type). TargetID points at the related entity for
// scoped events (topic ID, release ID, peer entry ID for relations,
// target entry ID for merges). Not every field is populated for
// every Type — the consumer interprets the payload based on Type.
type Activity struct {
	ID        string `bson:"_id"`
	EntryID   string
	Type      ActivityType
	Source    ActivitySourceType
	ActorID   string
	FromValue string
	ToValue   string
	TargetID  string
	CreatedAt time.Time
}

// NewActivity constructs an Activity with a fresh ID and CreatedAt
// set to now. Caller fills EntryID, Type, Source, and whichever
// payload fields the Type calls for.
func NewActivity() *Activity {
	return &Activity{
		ID:        uuid.New().String(),
		CreatedAt: time.Now().UTC(),
	}
}
