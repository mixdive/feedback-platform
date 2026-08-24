package dataoperations

import (
	"errors"
	"time"

	"github.com/mixdive/feedback-platform/models"
)

// ErrReadOnly is returned by every write method of a read-only Store
// implementation (the in-memory demo repo). Handlers normally never see
// it: in demo mode a middleware rejects mutating requests before they
// reach a handler. It exists as defense-in-depth so a write that somehow
// slips through fails loudly instead of silently no-op'ing.
var ErrReadOnly = errors.New("dataoperations: store is read-only")

// Store is the data-access contract every HTTP handler, middleware, and
// the AI worker depend on. There are two implementations:
//
//   - *DataOperations — the MongoDB-backed production store (this package).
//   - *demo.Repo      — a hardcoded in-memory, read-only store used when
//     the deployment boots in DEMO mode (the demo package).
//
// Keeping handlers behind this interface is what lets DEMO mode run with
// no Mongo connection at all. When you add a method to *DataOperations,
// add it here too (and to the demo repo) — the var assertions below and
// the demo package's own assertion fail the build until you do.
//
// The method set is generated from *DataOperations; the grouping mirrors
// the per-model files in this package.
type Store interface {
	// Lifecycle.
	Close()
	EnsureIndexes() error

	// Settings.
	GetSettings() (*models.Settings, error)
	InsertSettings(s *models.Settings) error
	UpdateSettings(set map[string]any) error
	EnsureFeedbackDefaults() error
	MigrateEntryTypeTemplatesToMultiLang() error
	RecordAISettingsSuccess() error
	RecordAISettingsError(msg string) error

	// Users.
	FindUserByID(id string) (*models.User, error)
	FindUserByKey(key string) (*models.User, error)
	ListUsersByIDs(ids []string) ([]models.User, error)
	ListUsers(f UserListFilter) ([]models.User, error)
	ListAllUsers(f UserListFilter) ([]models.User, error)
	CountUsers(f UserListFilter) (int64, error)
	ListUsersWithRoles() ([]models.User, error)
	CountActiveAdmins() (int64, error)
	InsertUser(u *models.User) error
	UpdateUser(u *models.User) error

	// Sessions.
	CreateSession(userID string) (*models.Session, error)
	ResolveSession(token string) (*models.User, error)
	DeleteSessionByToken(token string) error

	// Entries.
	ListEntries(f EntryListFilter) ([]models.Entry, error)
	CountEntries(f EntryListFilter) (int64, error)
	FindEntryByID(id string) (*models.Entry, error)
	ListEntryAuthorIDs() ([]string, error)
	CountEntriesByUserAndEntryType() (map[string]UserEntryCounts, error)
	InsertEntry(e *models.Entry) error
	SetEntryGitHubIssue(id string, issue models.GitHubIssue) error
	IncrementEntryVoteCount(id string, delta int) error
	IncrementEntryCommentCount(id string, delta int) error
	SetEntryIsInternal(id string, value bool) error
	SetEntryType(id string, ft models.EntryType) error
	SetEntryStatus(id string, status models.EntryStatus) error
	SetEntryRelease(id, releaseID string) error
	SetEntryTopics(id string, topicIDs []string) error
	SetEntryRelations(entryID string, relations []models.EntryRelation) error
	AddEntryRelation(entryID, peerEntryID string, relType models.EntryRelationType) error
	RemoveEntryRelation(entryID, peerEntryID string) error
	ListEntriesForSimilarity(publicOnly bool) ([]models.Entry, error)

	// Entry AI analysis (worker + analyzers).
	SetEntryTypeAnalysis(id string, a models.EntryTypeAnalysis) error
	FailEntryTypeAnalysis(id, errMsg string) error
	CountEntriesPendingEntryTypeAnalysis(claimTTL time.Duration) (int, error)
	CountEntriesInFlightEntryTypeAnalysis(claimTTL time.Duration) (int, error)
	ClaimNextPendingForEntryTypeAnalysis(claimTTL time.Duration) (*models.Entry, error)
	SetEntryTopicsFromAnalyzer(id string, topicIDs []string) error
	SetEntryTopicAnalysis(id string, a models.EntryTopicAnalysis) error
	FailEntryTopicAnalysis(id, errMsg string) error
	CountEntriesPendingTopicAnalysis(claimTTL time.Duration) (int, error)
	CountEntriesInFlightTopicAnalysis(claimTTL time.Duration) (int, error)
	ClaimNextPendingForTopicAnalysis(claimTTL time.Duration) (*models.Entry, error)
	SetEntryRelationsFromAnalyzer(entryID string, relations []models.EntryRelation) error
	SetEntryRelationAnalysis(id string, a models.EntryRelationAnalysis) error
	FailEntryRelationAnalysis(id, errMsg string) error
	CountEntriesPendingRelationAnalysis(claimTTL time.Duration) (int, error)
	CountEntriesInFlightRelationAnalysis(claimTTL time.Duration) (int, error)
	ClaimNextPendingForRelationAnalysis(claimTTL time.Duration) (*models.Entry, error)
	ListEntriesForRelationAnalysis(excludeID string) ([]models.Entry, error)

	// Comments.
	InsertComment(c *models.Comment) error
	FindCommentByID(id string) (*models.Comment, error)
	ListCommentsByEntryID(entryID string, includeInternal bool) ([]models.Comment, error)
	SetCommentIsInternal(id string, value bool) error
	CountCommentsByEntryID(entryID string) (int64, error)

	// Votes.
	FindVote(userID, entryID string) (*models.Vote, error)
	InsertVote(v *models.Vote) error
	InsertVoteIfAbsent(v *models.Vote) (bool, error)
	DeleteVote(userID, entryID string) error
	DeleteVoteIfPresent(userID, entryID string) (bool, error)
	ReconcileVoteCounts() (VoteReconcileReport, error)
	CountVotesForEntry(entryID string) (int, error)
	MigrateVotesToEntry(srcEntryID, dstEntryID string) (int, error)
	SetEntryVoteCount(entryID string, count int) error
	VotedEntryIDs(userID string, entryIDs []string) (map[string]bool, error)

	// Entry topics.
	ListEntryTopics() ([]models.EntryTopic, error)
	FindEntryTopicByID(id string) (*models.EntryTopic, error)
	FindEntryTopicByTitle(title string) (*models.EntryTopic, error)
	InsertEntryTopic(t *models.EntryTopic) error
	UpdateEntryTopicFields(id string, title, description, color *string, sortOrder *int) error
	DeleteEntryTopic(id string) error
	CountEntriesPerTopic() (map[string]int64, error)
	CountEntriesPerTopicByEntryType() (map[string]EntryTypeCounts, error)
	RemoveTopicIDFromAllEntries(topicID string) error

	// Releases.
	ListReleases() ([]models.Release, error)
	ListReleasesByState(state models.ReleaseState) ([]models.Release, error)
	FindReleaseByID(id string) (*models.Release, error)
	FindReleaseByVersionName(versionName string) (*models.Release, error)
	InsertRelease(r *models.Release) error
	UpdateReleaseFields(id string, versionName, title, description *string, releaseDate *time.Time, state *models.ReleaseState, pdf *ReleasePdfPatch) error
	DeleteRelease(id string) error
	RemoveReleaseIDFromAllEntries(releaseID string) error
	CountEntriesByReleaseID(releaseID string) (int64, error)
	CountEntriesPerReleaseByEntryType() (map[string]EntryTypeCounts, error)
	ListEntriesByReleaseID(releaseID string, publicOnly bool) ([]models.Entry, error)

	// Activities.
	InsertActivity(a *models.Activity) error
	ListActivitiesByEntryID(entryID string) ([]models.Activity, error)
	EnsureEntryCreatedActivities() (int, error)

	// Files.
	InsertFile(f *models.File) error
	FindFileByID(id string) (*models.File, error)

	// Dashboard.
	GetDashboardStats() (*DashboardStats, error)
}

// Compile-time assertion that the Mongo-backed store satisfies the
// interface. The demo package carries the matching assertion for its repo.
var _ Store = (*DataOperations)(nil)
