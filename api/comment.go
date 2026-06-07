package api

import (
	"time"

	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// CommentResponse is the wire shape for a single comment, identical on
// Console and Portal — the data the two surfaces need to render is the
// same. Author reuses EntryCreator since comments and entry records
// reveal the same projection of the user. IsInternal is always false
// on the Portal payload (the Portal listing endpoint filters them out)
// and reflects the persisted flag on the Console payload.
type CommentResponse struct {
	ID         string        `json:"id"`
	EntryID    string        `json:"entryId"`
	Body       string        `json:"body"`
	IsInternal bool          `json:"isInternal"`
	Author     *EntryCreator `json:"author,omitempty"`
	// AuthorIsTeam marks the comment author as a member of the team
	// (admin or editor — anyone with Console access). Both surfaces render
	// a small "Team" badge next to the name when true. Roles themselves
	// never cross the wire (see EntryCreator) — this is the single derived
	// signal we expose, and it carries no admin/editor distinction.
	AuthorIsTeam bool   `json:"authorIsTeam,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
} //@name Comment

// CommentListResponse is the envelope returned by both surfaces' list
// endpoints. Pagination is intentionally absent — we ship every comment on
// the entry sorted oldest-first.
type CommentListResponse struct {
	Data []CommentResponse `json:"data"`
} //@name CommentList

// commentISO renders a UTC ISO-8601 string. Mirrors the helper in each
// api/{console,portal} package so callers don't need to import the
// per-surface helper.
func commentISO(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

// BuildCommentResponse renders a Comment for the wire, looking up the
// author from the supplied authors map. A nil author entry (deleted user
// or missing lookup) leaves the field unset — frontends fall back the
// same way they do for anonymous entries. team carries the subset of
// author IDs with Console access; a hit flips AuthorIsTeam.
func BuildCommentResponse(c models.Comment, authors map[string]EntryCreator, team map[string]bool) CommentResponse {
	out := CommentResponse{
		ID:         c.ID,
		EntryID:    c.EntryID,
		Body:       c.Body,
		IsInternal: c.IsInternal,
		CreatedAt:  commentISO(c.CreatedAt),
		UpdatedAt:  commentISO(c.UpdatedAt),
	}
	if c.UserID != "" {
		if a, ok := authors[c.UserID]; ok {
			out.Author = &a
		}
		out.AuthorIsTeam = team[c.UserID]
	}
	return out
}

// LoadCommentAuthors batch-resolves the unique non-empty user IDs across
// the supplied comments into an EntryCreator map plus a team set (author
// IDs with Console access), both keyed by user ID. The query is inlined
// rather than delegating to LoadEntryCreators so the roles needed for the
// team set come back in the same single round trip.
func LoadCommentAuthors(do dataoperations.Store, comments []models.Comment) (map[string]EntryCreator, map[string]bool, error) {
	uniq := make([]string, 0, len(comments))
	seen := map[string]struct{}{}
	for _, c := range comments {
		if c.UserID == "" {
			continue
		}
		if _, dup := seen[c.UserID]; dup {
			continue
		}
		seen[c.UserID] = struct{}{}
		uniq = append(uniq, c.UserID)
	}
	authors := map[string]EntryCreator{}
	team := map[string]bool{}
	if len(uniq) == 0 {
		return authors, team, nil
	}
	users, err := do.ListUsersByIDs(uniq)
	if err != nil {
		return nil, nil, err
	}
	for _, u := range users {
		authors[u.ID] = BuildEntryCreator(u)
		if u.HasConsoleAccess() {
			team[u.ID] = true
		}
	}
	return authors, team, nil
}
