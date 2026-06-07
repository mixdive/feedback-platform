// Package api hosts cross-surface HTTP handlers (auth, setup, health) and
// the wire types shared between api/console and api/portal.
//
// Surface-specific handlers live in their respective subpackages
// (api/console, api/portal). Anything reused by both surfaces — the
// authoring projection rendered alongside an entry record, for instance
// — sits here and is exported.
package api

import (
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// EntryCreator is the lightweight projection of a User shown next to an
// entry on both the Console and Portal. Populated by BuildEntryCreator
// from the record's UserID lookup; records are rendered without a creator
// when UserID is empty (anonymous Portal submission) or when the user no
// longer exists.
//
// Email is deliberately absent from the wire shape — entry cards must
// not reveal someone else's address. Frontends fall back name → username →
// "Anonymous". Sensitive fields (PasswordHash, Keys, Roles) never leave
// the server.
type EntryCreator struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Username string `json:"username,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
} //@name EntryCreator

// BuildEntryCreator collapses the user's accounts (Email first, then
// Custom) into the wire shape, mirroring the precedence used by /api/me.
func BuildEntryCreator(u models.User) EntryCreator {
	out := EntryCreator{ID: u.ID}
	if a, ok := u.EmailAccount(); ok {
		if out.Name == "" {
			out.Name = a.Name
		}
		if out.Username == "" {
			out.Username = a.Username
		}
		if out.ImageURL == "" {
			out.ImageURL = a.ImageURL
		}
	}
	if a, ok := u.CustomAccount(); ok {
		if out.Name == "" {
			out.Name = a.Name
		}
		if out.Username == "" {
			out.Username = a.Username
		}
		if out.ImageURL == "" {
			out.ImageURL = a.ImageURL
		}
	}
	return out
}

// EntryTypeResponse is the cross-surface projection of an entry type.
// The set is a hardcoded enum (models.EntryType) — there is no
// database collection. The wire shape carries the value plus the
// display metadata so consumers don't have to bind enum-to-display in
// two places per render. Frontends keep an identical palette inline.
type EntryTypeResponse struct {
	Value string `json:"value"`
	Title string `json:"title"`
	Color string `json:"color,omitempty"`
	Icon  string `json:"icon,omitempty"`
} //@name EntryType

// entryTypeDisplay maps each enum value to its title, color, and icon.
// Frontends mirror this list in src/utils/entry-type.ts.
var entryTypeDisplay = map[models.EntryType]EntryTypeResponse{
	models.EntryTypeFeatureRequest: {Value: string(models.EntryTypeFeatureRequest), Title: "Feature Request", Color: "#10B981", Icon: "lightbulb"},
	models.EntryTypeBug:            {Value: string(models.EntryTypeBug), Title: "Bug", Color: "#E11D48", Icon: "bug"},
	models.EntryTypeSupport:        {Value: string(models.EntryTypeSupport), Title: "Support", Color: "#8B5CF6", Icon: "life-buoy"},
	models.EntryTypeOther:          {Value: string(models.EntryTypeOther), Title: "Other", Color: "#64748B", Icon: "ellipsis"},
}

// BuildEntryType returns the wire payload for t. Returns (zero value,
// false) for the empty string — entries with no type serialise the
// field as null/absent.
func BuildEntryType(t models.EntryType) (EntryTypeResponse, bool) {
	if t == "" {
		return EntryTypeResponse{}, false
	}
	r, ok := entryTypeDisplay[t]
	return r, ok
}

// EntryStatusResponse is the cross-surface projection of a status. The
// status set is a hardcoded enum (models.EntryStatus) — there is no
// database collection. Frontends carry the same palette inline; this
// response shape is what list/detail endpoints emit on the wire so
// consumers don't have to look up the enum value's display name.
type EntryStatusResponse struct {
	Value string `json:"value"`
	Title string `json:"title"`
	Color string `json:"color,omitempty"`
	Icon  string `json:"icon,omitempty"`
} //@name EntryStatus

// entryStatusDisplay maps each enum value to its title, color, and icon.
// Frontends carry the same metadata in their own `entry-status.ts`
// modules; the wire payload includes the resolved fields so the React
// apps don't have to bind enum-to-display in two places per render.
var entryStatusDisplay = map[models.EntryStatus]EntryStatusResponse{
	models.EntryStatusNew:        {Value: string(models.EntryStatusNew), Title: "New", Color: "#6366F1", Icon: "sparkles"},
	models.EntryStatusEvaluation: {Value: string(models.EntryStatusEvaluation), Title: "Evaluation", Color: "#0EA5E9", Icon: "search"},
	models.EntryStatusInProgress: {Value: string(models.EntryStatusInProgress), Title: "In Progress", Color: "#F59E0B", Icon: "loader"},
	models.EntryStatusCompleted:  {Value: string(models.EntryStatusCompleted), Title: "Completed", Color: "#10B981", Icon: "check"},
	models.EntryStatusCancelled:  {Value: string(models.EntryStatusCancelled), Title: "Cancelled", Color: "#E11D48", Icon: "x"},
}

// BuildEntryStatus returns the wire payload for s. Unknown values fall
// back to the default (so a stray document with a typo doesn't surface
// as a blank chip).
func BuildEntryStatus(s models.EntryStatus) EntryStatusResponse {
	if r, ok := entryStatusDisplay[s]; ok {
		return r
	}
	return entryStatusDisplay[models.EntryStatusDefault]
}

// ReleaseResponse is the cross-surface projection of a Release. Both
// the Console (settings list, entry edit form, entry badges) and the
// Portal (changelog page, entry badges) render this same shape. State
// is "planned" or "completed". ReleaseDate is YYYY-MM-DD (date-only).
type ReleaseResponse struct {
	ID          string `json:"id"`
	VersionName string `json:"versionName"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	ReleaseDate string `json:"releaseDate,omitempty"`
	State       string `json:"state"`
	// Optional dedicated PDF attachment. URL points at /api/files/...;
	// the Console renders an upload/clear control and the Portal
	// renders a download link. Omitted from the wire when no PDF is
	// attached.
	PdfFileURL  string `json:"pdfFileUrl,omitempty"`
	PdfFileName string `json:"pdfFileName,omitempty"`
	PdfFileSize int64  `json:"pdfFileSize,omitempty"`
} //@name Release

// BuildRelease projects the persisted release into wire shape. The
// release date is rendered as YYYY-MM-DD because the field carries
// no time component.
func BuildRelease(r models.Release) ReleaseResponse {
	out := ReleaseResponse{
		ID:          r.ID,
		VersionName: r.VersionName,
		Title:       r.Title,
		Description: r.Description,
		State:       string(r.State),
		PdfFileURL:  r.PdfFileURL,
		PdfFileName: r.PdfFileName,
		PdfFileSize: r.PdfFileSize,
	}
	if !r.ReleaseDate.IsZero() {
		out.ReleaseDate = r.ReleaseDate.UTC().Format("2006-01-02")
	}
	return out
}

// LoadReleases batch-resolves the unique non-empty IDs in releaseIDs
// into a ReleaseResponse map keyed by release ID. Missing releases
// (deleted between the entry write and this read, before the cascade
// ran) are absent from the map — the caller silently drops unknown
// IDs from the wire shape.
//
// One ListReleases call covers any number of input IDs; list-page
// queries that span dozens of entry releaseIds still hit Mongo once.
func LoadReleases(do dataoperations.Store, releaseIDs []string) (map[string]ReleaseResponse, error) {
	uniq := map[string]struct{}{}
	for _, id := range releaseIDs {
		if id == "" {
			continue
		}
		uniq[id] = struct{}{}
	}
	out := map[string]ReleaseResponse{}
	if len(uniq) == 0 {
		return out, nil
	}
	all, err := do.ListReleases()
	if err != nil {
		return nil, err
	}
	for _, r := range all {
		if _, ok := uniq[r.ID]; ok {
			out[r.ID] = BuildRelease(r)
		}
	}
	return out, nil
}

// GitHubIssueResponse is the cross-surface wire shape for a linked
// GitHub issue. Both the Console (the create button flips to a "View
// on GitHub" link) and the Portal ("Tracked on GitHub" badge) render
// the same payload. Number is included alongside URL so the Console
// can show "#42" without parsing the URL.
//
// Owner / Repo are intentionally omitted — the URL already carries
// them and exposing them separately invites duplicate sources of
// truth in the React apps.
type GitHubIssueResponse struct {
	URL       string `json:"url"`
	Number    int    `json:"number"`
	CreatedAt string `json:"createdAt,omitempty"`
} //@name GitHubIssue

// BuildGitHubIssue projects the embedded model into wire shape. Returns
// (zero value, false) when no issue is linked (Number == 0) so
// entryToResponse can omit the field entirely.
func BuildGitHubIssue(g models.GitHubIssue) (GitHubIssueResponse, bool) {
	if g.Number == 0 || g.URL == "" {
		return GitHubIssueResponse{}, false
	}
	out := GitHubIssueResponse{
		URL:    g.URL,
		Number: g.Number,
	}
	if !g.CreatedAt.IsZero() {
		out.CreatedAt = g.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	return out, true
}

// LoadEntryCreators batch-resolves the unique non-empty IDs in userIDs
// into an EntryCreator map keyed by user ID. Missing users (deleted,
// never existed) are simply absent from the map — the caller falls back to
// rendering the entry without a creator. The empty-string ID is skipped
// so anonymously authored records never trigger a lookup.
func LoadEntryCreators(do dataoperations.Store, userIDs []string) (map[string]EntryCreator, error) {
	uniq := make([]string, 0, len(userIDs))
	seen := map[string]struct{}{}
	for _, id := range userIDs {
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	out := map[string]EntryCreator{}
	if len(uniq) == 0 {
		return out, nil
	}
	users, err := do.ListUsersByIDs(uniq)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		out[u.ID] = BuildEntryCreator(u)
	}
	return out, nil
}
