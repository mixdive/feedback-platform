// Package demo provides a hardcoded, in-memory, read-only implementation
// of dataoperations.Store. It is what the server runs on when booted in
// DEMO mode: no MongoDB connection is required at all.
//
// DEMO is the single sanctioned exception to the "only MONGO_URI is
// accepted as a deployment env var" rule in CLAUDE.md. In demo mode a
// middleware auto-authenticates every visitor as the synthetic admin
// (DemoAdmin) so both Console and Portal are fully browsable, and another
// middleware rejects every mutating request with a 403 — so the data is
// strictly read-only.
//
// ----------------------------------------------------------------------
// MAINTENANCE — keep this package in lockstep with the product.
//
// The demo is used for marketing screenshots, so it must exercise as much
// of the product as possible AND stay a faithful read mirror of the Mongo
// store. Two rules:
//
//  1. When you add a method to dataoperations.Store, add it here too. The
//     `var _ dataoperations.Store = (*Repo)(nil)` assertion at the bottom
//     fails the build until you do. Reads should serve the in-memory data;
//     mutations return dataoperations.ErrReadOnly.
//  2. When you add a model/feature, extend dataset.go (content) and
//     build.go (assembly) so the demo keeps showing it.
//
// The dataset currently covers: settings + branding, feedback policy,
// GitHub integration, faux-AI badges (type/topic/relation provenance with
// reasons), admin+editor+portal users, admin & AI topics, completed +
// planned releases, ~40 entries across every type/status incl. internal,
// public + internal comments, votes, a merged-away
// duplicate, a GitHub-linked entry, and the full activity timeline.
// ----------------------------------------------------------------------
package demo

import (
	"sort"
	"strings"
	"time"

	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// Repo is the in-memory, read-only Store. Built once at construction and
// never mutated; safe for concurrent reads without locking.
type Repo struct {
	d *data

	entriesByID map[string]*models.Entry
	usersByID   map[string]*models.User
	topicsByID  map[string]*models.EntryTopic
}

// New builds the demo dataset and returns a ready-to-serve repo.
func New() *Repo {
	d := buildData()
	r := &Repo{
		d:           d,
		entriesByID: make(map[string]*models.Entry, len(d.entries)),
		usersByID:   make(map[string]*models.User, len(d.users)),
		topicsByID:  make(map[string]*models.EntryTopic, len(d.topics)),
	}
	for i := range d.entries {
		r.entriesByID[d.entries[i].ID] = &d.entries[i]
	}
	for i := range d.users {
		r.usersByID[d.users[i].ID] = &d.users[i]
	}
	for i := range d.topics {
		r.topicsByID[d.topics[i].ID] = &d.topics[i]
	}
	return r
}

// DemoAdmin returns the synthetic admin every demo visitor is
// auto-authenticated as. The router wires this into the demo auth
// middleware.
func (r *Repo) DemoAdmin() *models.User { return r.d.adminUser }

// Close is a no-op — there is no connection to release.
func (r *Repo) Close() {}

// ---------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------

func (r *Repo) GetSettings() (*models.Settings, error) {
	s := *r.d.settings
	return &s, nil
}

// ---------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------

func (r *Repo) FindUserByID(id string) (*models.User, error) {
	if u, ok := r.usersByID[id]; ok {
		c := *u
		return &c, nil
	}
	return nil, nil
}

func (r *Repo) FindUserByKey(key string) (*models.User, error) {
	if key == "" {
		return nil, nil
	}
	for i := range r.d.users {
		for _, k := range r.d.users[i].Keys {
			if k == key {
				c := r.d.users[i]
				return &c, nil
			}
		}
	}
	return nil, nil
}

func (r *Repo) ListUsersByIDs(ids []string) ([]models.User, error) {
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	out := []models.User{}
	for i := range r.d.users {
		if want[r.d.users[i].ID] {
			out = append(out, r.d.users[i])
		}
	}
	return out, nil
}

func (r *Repo) ListUsers(f dataoperations.UserListFilter) ([]models.User, error) {
	matched := r.filterUsers(f)
	sortUsers(matched, f.Sort)
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	page := f.Page
	if page <= 0 {
		page = 1
	}
	return paginate(matched, page, limit), nil
}

func (r *Repo) CountUsers(f dataoperations.UserListFilter) (int64, error) {
	return int64(len(r.filterUsers(f))), nil
}

func (r *Repo) ListAllUsers(f dataoperations.UserListFilter) ([]models.User, error) {
	return r.filterUsers(f), nil
}

func (r *Repo) ListUsersWithRoles() ([]models.User, error) {
	out := []models.User{}
	for i := range r.d.users {
		if len(r.d.users[i].Roles) > 0 {
			out = append(out, r.d.users[i])
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *Repo) CountActiveAdmins() (int64, error) {
	var n int64
	for i := range r.d.users {
		u := r.d.users[i]
		if u.IsBlocked || u.IsDeleted {
			continue
		}
		if u.IsAdmin() {
			n++
		}
	}
	return n, nil
}

// filterUsers replicates dataoperations.userListFilter: soft-deleted are
// always excluded; search matches keys + account names; role/status
// narrow the set.
func (r *Repo) filterUsers(f dataoperations.UserListFilter) []models.User {
	out := []models.User{}
	search := strings.ToLower(strings.TrimSpace(f.Search))
	for i := range r.d.users {
		u := r.d.users[i]
		if u.IsDeleted {
			continue
		}
		if search != "" && !userMatchesSearch(u, search) {
			continue
		}
		switch f.Role {
		case "admin":
			if !hasRole(u, models.RoleAdmin) {
				continue
			}
		case "editor":
			if !hasRole(u, models.RoleEditor) {
				continue
			}
		case "none":
			if len(u.Roles) > 0 {
				continue
			}
		}
		switch f.Status {
		case "active":
			if u.IsBlocked {
				continue
			}
		case "blocked":
			if !u.IsBlocked {
				continue
			}
		}
		out = append(out, u)
	}
	return out
}

func userMatchesSearch(u models.User, lc string) bool {
	for _, k := range u.Keys {
		if strings.Contains(strings.ToLower(k), lc) {
			return true
		}
	}
	for _, a := range u.Accounts {
		if strings.Contains(strings.ToLower(a.Name), lc) || strings.Contains(strings.ToLower(a.Username), lc) {
			return true
		}
	}
	return false
}

func hasRole(u models.User, role models.UserRole) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

func sortUsers(list []models.User, key string) {
	switch key {
	case "old":
		sort.SliceStable(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
	case "email":
		sort.SliceStable(list, func(i, j int) bool {
			ai, _ := list[i].EmailAccount()
			aj, _ := list[j].EmailAccount()
			if ai.Email != aj.Email {
				return ai.Email < aj.Email
			}
			return list[i].CreatedAt.After(list[j].CreatedAt)
		})
	default: // "new"
		sort.SliceStable(list, func(i, j int) bool { return list[i].CreatedAt.After(list[j].CreatedAt) })
	}
}

// ---------------------------------------------------------------------
// Sessions (auth is injected in demo; these are inert)
// ---------------------------------------------------------------------

func (r *Repo) ResolveSession(token string) (*models.User, error) { return nil, nil }
func (r *Repo) DeleteSessionByToken(token string) error           { return nil }

// ---------------------------------------------------------------------
// Entries
// ---------------------------------------------------------------------

func (r *Repo) ListEntries(f dataoperations.EntryListFilter) ([]models.Entry, error) {
	matched := r.filterEntries(f)
	sortEntries(matched, f.Sort)
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	page := f.Page
	if page <= 0 {
		page = 1
	}
	return paginate(matched, page, limit), nil
}

func (r *Repo) CountEntries(f dataoperations.EntryListFilter) (int64, error) {
	return int64(len(r.filterEntries(f))), nil
}

func (r *Repo) FindEntryByID(id string) (*models.Entry, error) {
	if e, ok := r.entriesByID[id]; ok {
		c := *e
		return &c, nil
	}
	return nil, nil
}

func (r *Repo) ListEntryAuthorIDs() ([]string, error) {
	seen := map[string]bool{}
	out := []string{}
	for i := range r.d.entries {
		uid := r.d.entries[i].UserID
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		out = append(out, uid)
	}
	return out, nil
}

func (r *Repo) CountEntriesByUserAndEntryType() (map[string]dataoperations.UserEntryCounts, error) {
	out := map[string]dataoperations.UserEntryCounts{}
	for i := range r.d.entries {
		e := r.d.entries[i]
		if e.UserID == "" {
			continue
		}
		c := out[e.UserID]
		c.Total++
		switch e.EntryType {
		case models.EntryTypeFeatureRequest:
			c.FeatureRequest++
		case models.EntryTypeBug:
			c.Bug++
		case models.EntryTypeSupport:
			c.Support++
		default:
			c.Other++
		}
		out[e.UserID] = c
	}
	return out, nil
}

func (r *Repo) ListEntriesByReleaseID(releaseID string, publicOnly bool) ([]models.Entry, error) {
	if releaseID == "" {
		return []models.Entry{}, nil
	}
	out := []models.Entry{}
	for i := range r.d.entries {
		e := r.d.entries[i]
		if e.ReleaseID != releaseID {
			continue
		}
		if publicOnly && e.IsInternal {
			continue
		}
		out = append(out, e)
	}
	sortEntries(out, "new")
	return out, nil
}

func (r *Repo) ListEntriesForSimilarity(publicOnly bool) ([]models.Entry, error) {
	out := []models.Entry{}
	for i := range r.d.entries {
		if publicOnly && r.d.entries[i].IsInternal {
			continue
		}
		out = append(out, r.d.entries[i])
	}
	sortEntries(out, "new")
	return out, nil
}

func (r *Repo) ListEntriesForRelationAnalysis(excludeID string) ([]models.Entry, error) {
	out := []models.Entry{}
	for i := range r.d.entries {
		if r.d.entries[i].ID == excludeID {
			continue
		}
		out = append(out, r.d.entries[i])
	}
	sortEntries(out, "new")
	return out, nil
}

// filterEntries replicates dataoperations.entryFilter.
func (r *Repo) filterEntries(f dataoperations.EntryListFilter) []models.Entry {
	out := []models.Entry{}
	search := strings.ToLower(strings.TrimSpace(f.Search))
	for i := range r.d.entries {
		e := r.d.entries[i]
		if search != "" {
			if !strings.Contains(strings.ToLower(e.Title), search) && !strings.Contains(strings.ToLower(e.Description), search) {
				continue
			}
		}
		if f.IsInternal != nil {
			if f.IncludeOwnerID != "" {
				if e.IsInternal != *f.IsInternal && e.UserID != f.IncludeOwnerID {
					continue
				}
			} else if e.IsInternal != *f.IsInternal {
				continue
			}
		}
		if f.OwnerID != "" && e.UserID != f.OwnerID {
			continue
		}
		if f.EntryType != "" && e.EntryType != f.EntryType {
			continue
		}
		if f.Status != "" {
			if e.Status != f.Status {
				continue
			}
		} else if f.OpenOnly && !models.IsEntryStatusOpen(e.Status) {
			continue
		}
		if f.TopicID != "" && !contains(e.TopicIDs, f.TopicID) {
			continue
		}
		if f.ReleaseID != "" && e.ReleaseID != f.ReleaseID {
			continue
		}
		out = append(out, e)
	}
	return out
}

func sortEntries(list []models.Entry, key string) {
	switch key {
	case "top":
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].VoteCount != list[j].VoteCount {
				return list[i].VoteCount > list[j].VoteCount
			}
			if !list[i].CreatedAt.Equal(list[j].CreatedAt) {
				return list[i].CreatedAt.After(list[j].CreatedAt)
			}
			return list[i].ID > list[j].ID
		})
	default: // "new"
		sort.SliceStable(list, func(i, j int) bool {
			if !list[i].CreatedAt.Equal(list[j].CreatedAt) {
				return list[i].CreatedAt.After(list[j].CreatedAt)
			}
			return list[i].ID > list[j].ID
		})
	}
}

// ---------------------------------------------------------------------
// Comments
// ---------------------------------------------------------------------

func (r *Repo) FindCommentByID(id string) (*models.Comment, error) {
	for i := range r.d.comments {
		if r.d.comments[i].ID == id {
			c := r.d.comments[i]
			return &c, nil
		}
	}
	return nil, nil
}

func (r *Repo) ListCommentsByEntryID(entryID string, includeInternal bool) ([]models.Comment, error) {
	if entryID == "" {
		return []models.Comment{}, nil
	}
	out := []models.Comment{}
	for i := range r.d.comments {
		c := r.d.comments[i]
		if c.EntryID != entryID {
			continue
		}
		if !includeInternal && c.IsInternal {
			continue
		}
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *Repo) CountCommentsByEntryID(entryID string) (int64, error) {
	var n int64
	for i := range r.d.comments {
		if r.d.comments[i].EntryID == entryID {
			n++
		}
	}
	return n, nil
}

// ---------------------------------------------------------------------
// Votes
// ---------------------------------------------------------------------

func (r *Repo) FindVote(userID, entryID string) (*models.Vote, error) {
	for i := range r.d.votes {
		if r.d.votes[i].UserID == userID && r.d.votes[i].EntryID == entryID {
			c := r.d.votes[i]
			return &c, nil
		}
	}
	return nil, nil
}

func (r *Repo) CountVotesForEntry(entryID string) (int, error) {
	n := 0
	for i := range r.d.votes {
		if r.d.votes[i].EntryID == entryID {
			n++
		}
	}
	return n, nil
}

func (r *Repo) VotedEntryIDs(userID string, entryIDs []string) (map[string]bool, error) {
	out := map[string]bool{}
	if userID == "" || len(entryIDs) == 0 {
		return out, nil
	}
	want := map[string]bool{}
	for _, id := range entryIDs {
		want[id] = true
	}
	for i := range r.d.votes {
		v := r.d.votes[i]
		if v.UserID == userID && want[v.EntryID] {
			out[v.EntryID] = true
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------
// Entry topics
// ---------------------------------------------------------------------

func (r *Repo) ListEntryTopics() ([]models.EntryTopic, error) {
	out := make([]models.EntryTopic, len(r.d.topics))
	copy(out, r.d.topics)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (r *Repo) FindEntryTopicByID(id string) (*models.EntryTopic, error) {
	if t, ok := r.topicsByID[id]; ok {
		c := *t
		return &c, nil
	}
	return nil, nil
}

func (r *Repo) FindEntryTopicByTitle(title string) (*models.EntryTopic, error) {
	for i := range r.d.topics {
		if r.d.topics[i].Title == title {
			c := r.d.topics[i]
			return &c, nil
		}
	}
	return nil, nil
}

func (r *Repo) CountEntriesPerTopic() (map[string]int64, error) {
	out := map[string]int64{}
	for i := range r.d.entries {
		for _, tid := range r.d.entries[i].TopicIDs {
			if tid == "" {
				continue
			}
			out[tid]++
		}
	}
	return out, nil
}

func (r *Repo) CountEntriesPerTopicByEntryType() (map[string]dataoperations.EntryTypeCounts, error) {
	out := map[string]dataoperations.EntryTypeCounts{}
	for i := range r.d.entries {
		e := r.d.entries[i]
		for _, tid := range e.TopicIDs {
			if tid == "" {
				continue
			}
			c := out[tid]
			bumpTypeCounts(&c, e.EntryType)
			out[tid] = c
		}
	}
	return out, nil
}

func (r *Repo) RemoveTopicIDFromAllEntries(topicID string) error { return dataoperations.ErrReadOnly }

// ---------------------------------------------------------------------
// Releases
// ---------------------------------------------------------------------

func (r *Repo) ListReleases() ([]models.Release, error) {
	out := make([]models.Release, len(r.d.releases))
	copy(out, r.d.releases)
	sortReleases(out)
	return out, nil
}

func (r *Repo) ListReleasesByState(state models.ReleaseState) ([]models.Release, error) {
	out := []models.Release{}
	for i := range r.d.releases {
		if r.d.releases[i].State == state {
			out = append(out, r.d.releases[i])
		}
	}
	sortReleases(out)
	return out, nil
}

func (r *Repo) FindReleaseByID(id string) (*models.Release, error) {
	if id == "" {
		return nil, nil
	}
	for i := range r.d.releases {
		if r.d.releases[i].ID == id {
			c := r.d.releases[i]
			return &c, nil
		}
	}
	return nil, nil
}

func (r *Repo) FindReleaseByVersionName(versionName string) (*models.Release, error) {
	for i := range r.d.releases {
		if r.d.releases[i].VersionName == versionName {
			c := r.d.releases[i]
			return &c, nil
		}
	}
	return nil, nil
}

func (r *Repo) CountEntriesByReleaseID(releaseID string) (int64, error) {
	if releaseID == "" {
		return 0, nil
	}
	var n int64
	for i := range r.d.entries {
		if r.d.entries[i].ReleaseID == releaseID {
			n++
		}
	}
	return n, nil
}

func (r *Repo) CountEntriesPerReleaseByEntryType() (map[string]dataoperations.EntryTypeCounts, error) {
	out := map[string]dataoperations.EntryTypeCounts{}
	for i := range r.d.entries {
		e := r.d.entries[i]
		if e.ReleaseID == "" {
			continue
		}
		c := out[e.ReleaseID]
		bumpTypeCounts(&c, e.EntryType)
		out[e.ReleaseID] = c
	}
	return out, nil
}

func sortReleases(list []models.Release) {
	sort.SliceStable(list, func(i, j int) bool {
		if !list[i].ReleaseDate.Equal(list[j].ReleaseDate) {
			return list[i].ReleaseDate.After(list[j].ReleaseDate)
		}
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
}

// ---------------------------------------------------------------------
// Activities
// ---------------------------------------------------------------------

func (r *Repo) ListActivitiesByEntryID(entryID string) ([]models.Activity, error) {
	out := []models.Activity{}
	for i := range r.d.activities {
		if r.d.activities[i].EntryID == entryID {
			out = append(out, r.d.activities[i])
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// ---------------------------------------------------------------------
// Files (none in the demo)
// ---------------------------------------------------------------------

func (r *Repo) FindFileByID(id string) (*models.File, error) { return nil, nil }

// ---------------------------------------------------------------------
// Dashboard
// ---------------------------------------------------------------------

func (r *Repo) GetDashboardStats() (*dataoperations.DashboardStats, error) {
	out := &dataoperations.DashboardStats{
		EntriesByType:    map[string]int64{},
		EntriesByStatus:  map[string]int64{},
		EntriesPerDay:    []dataoperations.DashboardDayBucket{},
		TopEntriesByVote: []dataoperations.DashboardEntryRow{},
		RecentEntries:    []dataoperations.DashboardEntryRow{},
	}
	out.TotalEntries = int64(len(r.d.entries))
	for i := range r.d.entries {
		e := r.d.entries[i]
		if e.IsInternal {
			out.InternalEntries++
		}
		if !models.IsEntryStatusOpen(e.Status) {
			out.ClosedEntries++
		}
		// EntriesByType (untyped/other folds into "other").
		typeKey := string(e.EntryType)
		switch e.EntryType {
		case models.EntryTypeFeatureRequest, models.EntryTypeBug, models.EntryTypeSupport:
		default:
			typeKey = string(models.EntryTypeOther)
		}
		out.EntriesByType[typeKey]++
		// EntriesByStatus.
		statusKey := string(e.Status)
		if !models.IsValidEntryStatus(e.Status) {
			statusKey = string(models.EntryStatusDefault)
		}
		out.EntriesByStatus[statusKey]++
	}
	out.PublicEntries = out.TotalEntries - out.InternalEntries
	out.OpenEntries = out.TotalEntries - out.ClosedEntries
	out.TotalUsers = int64(len(r.d.users))
	out.TotalTopics = int64(len(r.d.topics))
	out.TotalReleases = int64(len(r.d.releases))
	out.TotalComments = int64(len(r.d.comments))
	out.TotalVotes = int64(len(r.d.votes))

	// 30-day submission trend.
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -29).Truncate(24 * time.Hour)
	byDate := map[string]int64{}
	for i := range r.d.entries {
		if r.d.entries[i].CreatedAt.Before(start) {
			continue
		}
		byDate[r.d.entries[i].CreatedAt.Format("2006-01-02")]++
	}
	for i := 0; i < 30; i++ {
		day := start.AddDate(0, 0, i).Format("2006-01-02")
		out.EntriesPerDay = append(out.EntriesPerDay, dataoperations.DashboardDayBucket{Date: day, Count: byDate[day]})
	}

	top, _ := r.ListEntries(dataoperations.EntryListFilter{Sort: "top", Limit: 5})
	for _, e := range top {
		out.TopEntriesByVote = append(out.TopEntriesByVote, dashboardRow(e))
	}
	recent, _ := r.ListEntries(dataoperations.EntryListFilter{Sort: "new", Limit: 5})
	for _, e := range recent {
		out.RecentEntries = append(out.RecentEntries, dashboardRow(e))
	}
	return out, nil
}

func dashboardRow(e models.Entry) dataoperations.DashboardEntryRow {
	return dataoperations.DashboardEntryRow{
		ID:           e.ID,
		Title:        e.Title,
		EntryType:    string(e.EntryType),
		Status:       string(e.Status),
		VoteCount:    e.VoteCount,
		CommentCount: e.CommentCount,
		IsInternal:   e.IsInternal,
		CreatedAt:    e.CreatedAt,
	}
}

// ---------------------------------------------------------------------
// Worker claim/count reads — inert (no worker runs against this store).
// ---------------------------------------------------------------------

func (r *Repo) ClaimNextPendingForEntryTypeAnalysis(time.Duration) (*models.Entry, error) {
	return nil, nil
}
func (r *Repo) ClaimNextPendingForTopicAnalysis(time.Duration) (*models.Entry, error) {
	return nil, nil
}
func (r *Repo) ClaimNextPendingForRelationAnalysis(time.Duration) (*models.Entry, error) {
	return nil, nil
}
func (r *Repo) CountEntriesPendingEntryTypeAnalysis(time.Duration) (int, error)  { return 0, nil }
func (r *Repo) CountEntriesInFlightEntryTypeAnalysis(time.Duration) (int, error) { return 0, nil }
func (r *Repo) CountEntriesPendingTopicAnalysis(time.Duration) (int, error)      { return 0, nil }
func (r *Repo) CountEntriesInFlightTopicAnalysis(time.Duration) (int, error)     { return 0, nil }
func (r *Repo) CountEntriesPendingRelationAnalysis(time.Duration) (int, error)   { return 0, nil }
func (r *Repo) CountEntriesInFlightRelationAnalysis(time.Duration) (int, error)  { return 0, nil }

// ---------------------------------------------------------------------
// Mutations — all rejected. The demo is read-only; a middleware blocks
// mutating HTTP requests before they reach a handler, so these are
// defense-in-depth.
// ---------------------------------------------------------------------

func (r *Repo) InsertSettings(*models.Settings) error         { return dataoperations.ErrReadOnly }
func (r *Repo) UpdateSettings(map[string]any) error           { return dataoperations.ErrReadOnly }
func (r *Repo) EnsureFeedbackDefaults() error                 { return dataoperations.ErrReadOnly }
func (r *Repo) MigrateEntryTypeTemplatesToMultiLang() error   { return dataoperations.ErrReadOnly }
func (r *Repo) RecordAISettingsSuccess() error                { return dataoperations.ErrReadOnly }
func (r *Repo) RecordAISettingsError(string) error            { return dataoperations.ErrReadOnly }
func (r *Repo) InsertUser(*models.User) error                 { return dataoperations.ErrReadOnly }
func (r *Repo) UpdateUser(*models.User) error                 { return dataoperations.ErrReadOnly }
func (r *Repo) CreateSession(string) (*models.Session, error) { return nil, dataoperations.ErrReadOnly }
func (r *Repo) InsertEntry(*models.Entry) error               { return dataoperations.ErrReadOnly }
func (r *Repo) SetEntryGitHubIssue(string, models.GitHubIssue) error {
	return dataoperations.ErrReadOnly
}
func (r *Repo) IncrementEntryVoteCount(string, int) error       { return dataoperations.ErrReadOnly }
func (r *Repo) IncrementEntryCommentCount(string, int) error    { return dataoperations.ErrReadOnly }
func (r *Repo) SetEntryIsInternal(string, bool) error           { return dataoperations.ErrReadOnly }
func (r *Repo) SetEntryType(string, models.EntryType) error     { return dataoperations.ErrReadOnly }
func (r *Repo) SetEntryStatus(string, models.EntryStatus) error { return dataoperations.ErrReadOnly }
func (r *Repo) SetEntryRelease(string, string) error            { return dataoperations.ErrReadOnly }
func (r *Repo) SetEntryTopics(string, []string) error           { return dataoperations.ErrReadOnly }
func (r *Repo) SetEntryRelations(string, []models.EntryRelation) error {
	return dataoperations.ErrReadOnly
}
func (r *Repo) AddEntryRelation(string, string, models.EntryRelationType) error {
	return dataoperations.ErrReadOnly
}
func (r *Repo) RemoveEntryRelation(string, string) error { return dataoperations.ErrReadOnly }
func (r *Repo) SetEntryTypeAnalysis(string, models.EntryTypeAnalysis) error {
	return dataoperations.ErrReadOnly
}
func (r *Repo) FailEntryTypeAnalysis(string, string) error { return dataoperations.ErrReadOnly }
func (r *Repo) SetEntryTopicsFromAnalyzer(string, []string) error {
	return dataoperations.ErrReadOnly
}
func (r *Repo) SetEntryTopicAnalysis(string, models.EntryTopicAnalysis) error {
	return dataoperations.ErrReadOnly
}
func (r *Repo) FailEntryTopicAnalysis(string, string) error { return dataoperations.ErrReadOnly }
func (r *Repo) SetEntryRelationsFromAnalyzer(string, []models.EntryRelation) error {
	return dataoperations.ErrReadOnly
}
func (r *Repo) SetEntryRelationAnalysis(string, models.EntryRelationAnalysis) error {
	return dataoperations.ErrReadOnly
}
func (r *Repo) FailEntryRelationAnalysis(string, string) error { return dataoperations.ErrReadOnly }
func (r *Repo) InsertComment(*models.Comment) error            { return dataoperations.ErrReadOnly }
func (r *Repo) SetCommentIsInternal(string, bool) error        { return dataoperations.ErrReadOnly }
func (r *Repo) InsertVote(*models.Vote) error                  { return dataoperations.ErrReadOnly }
func (r *Repo) InsertVoteIfAbsent(*models.Vote) (bool, error) {
	return false, dataoperations.ErrReadOnly
}
func (r *Repo) DeleteVote(string, string) error { return dataoperations.ErrReadOnly }
func (r *Repo) DeleteVoteIfPresent(string, string) (bool, error) {
	return false, dataoperations.ErrReadOnly
}

// ReconcileVoteCounts and EnsureIndexes are no-ops in demo mode: the
// dataset is in-memory, immutable, and already consistent, and there is no
// Mongo connection to build an index on.
func (r *Repo) ReconcileVoteCounts() (dataoperations.VoteReconcileReport, error) {
	return dataoperations.VoteReconcileReport{}, nil
}
func (r *Repo) EnsureIndexes() error                            { return nil }
func (r *Repo) MigrateVotesToEntry(string, string) (int, error) { return 0, dataoperations.ErrReadOnly }
func (r *Repo) SetEntryVoteCount(string, int) error             { return dataoperations.ErrReadOnly }
func (r *Repo) InsertEntryTopic(*models.EntryTopic) error       { return dataoperations.ErrReadOnly }
func (r *Repo) UpdateEntryTopicFields(string, *string, *string, *string, *int) error {
	return dataoperations.ErrReadOnly
}
func (r *Repo) DeleteEntryTopic(string) error       { return dataoperations.ErrReadOnly }
func (r *Repo) InsertRelease(*models.Release) error { return dataoperations.ErrReadOnly }
func (r *Repo) UpdateReleaseFields(string, *string, *string, *string, *time.Time, *models.ReleaseState, *dataoperations.ReleasePdfPatch) error {
	return dataoperations.ErrReadOnly
}
func (r *Repo) DeleteRelease(string) error                 { return dataoperations.ErrReadOnly }
func (r *Repo) RemoveReleaseIDFromAllEntries(string) error { return dataoperations.ErrReadOnly }
func (r *Repo) InsertActivity(*models.Activity) error      { return dataoperations.ErrReadOnly }
func (r *Repo) EnsureEntryCreatedActivities() (int, error) { return 0, dataoperations.ErrReadOnly }
func (r *Repo) InsertFile(*models.File) error              { return dataoperations.ErrReadOnly }

// ---------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func bumpTypeCounts(c *dataoperations.EntryTypeCounts, t models.EntryType) {
	c.Total++
	switch t {
	case models.EntryTypeFeatureRequest:
		c.FeatureRequest++
	case models.EntryTypeBug:
		c.Bug++
	case models.EntryTypeSupport:
		c.Support++
	default:
		c.Other++
	}
}

func paginate[T any](list []T, page, limit int) []T {
	skip := (page - 1) * limit
	if skip >= len(list) {
		return []T{}
	}
	end := skip + limit
	if end > len(list) {
		end = len(list)
	}
	return list[skip:end]
}

// Compile-time assertion that the demo repo satisfies the data-access
// contract. Fails the build the moment Store gains a method this repo
// doesn't implement.
var _ dataoperations.Store = (*Repo)(nil)
