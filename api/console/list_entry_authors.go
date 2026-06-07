package console

import (
	"sort"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// entryAuthorListResponse is the envelope for the Author filter dropdown
// on the Console entries list. Each row is the same EntryCreator shape
// that's already attached to a rendered entry, so the frontend can match
// list rows to filter chips by ID without an extra lookup.
type entryAuthorListResponse struct {
	Data []api.EntryCreator `json:"data"`
} //@name EntryAuthorList

// ListEntryAuthorsHandler returns every distinct user that owns at least
// one entry, sorted by display name. Anonymous Portal entries (userid
// empty) and entries whose creator no longer exists are dropped — they
// can't be filtered on anyway.
//
//	@ID			console-list-entry-authors
//	@Summary	List entry authors (admin)
//	@Tags		Console
//	@Produce	json
//	@Success	200	{object}	entryAuthorListResponse
//	@Failure	500	{object}	response.ApiError
//	@Router		/api/console/entry-author [get]
func ListEntryAuthorsHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ids, err := do.ListEntryAuthorIDs()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		creators, err := api.LoadEntryCreators(do, ids)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		out := make([]api.EntryCreator, 0, len(creators))
		for _, u := range creators {
			out = append(out, u)
		}
		sort.Slice(out, func(i, j int) bool {
			return authorLabel(out[i]) < authorLabel(out[j])
		})
		response.Success(c, entryAuthorListResponse{Data: out})
	}
}

// authorLabel is the sort key — name first, then username, then ID. Keeps
// the dropdown order stable across requests.
func authorLabel(u api.EntryCreator) string {
	if u.Name != "" {
		return u.Name
	}
	if u.Username != "" {
		return u.Username
	}
	return u.ID
}
