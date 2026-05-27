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
	CreatedAt  string        `json:"createdAt"`
	UpdatedAt  string        `json:"updatedAt"`
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
// same way they do for anonymous entries.
func BuildCommentResponse(c models.Comment, authors map[string]EntryCreator) CommentResponse {
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
	}
	return out
}

// LoadCommentAuthors batch-resolves the unique non-empty user IDs across
// the supplied comments into an EntryCreator map keyed by user ID.
// Mirrors LoadEntryCreators so list handlers stay one round trip.
func LoadCommentAuthors(do *dataoperations.DataOperations, comments []models.Comment) (map[string]EntryCreator, error) {
	ids := make([]string, 0, len(comments))
	for _, c := range comments {
		ids = append(ids, c.UserID)
	}
	return LoadEntryCreators(do, ids)
}
