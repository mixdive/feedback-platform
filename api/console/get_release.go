package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// releaseDetailResponse extends the cross-surface release projection
// with the list of entries linked to this release. The Console release
// edit page renders this list as a read-only "linked entries" panel
// so admins can see what's already assigned without leaving the
// settings screen.
type releaseDetailResponse struct {
	api.ReleaseResponse
	Entries []releaseEntrySummary `json:"entries"`
} //@name ConsoleReleaseDetail

// releaseEntrySummary is the cell-style projection of an entry on the
// release detail page. Lighter than the full Console entryResponse
// since this is a context view, not a full management surface.
type releaseEntrySummary struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	IsInternal bool   `json:"isInternal"`
} //@name ConsoleReleaseEntrySummary

// GetReleaseHandler returns one release plus its linked entries.
//
//	@ID			console-get-release
//	@Summary	Get release
//	@Tags		Console
//	@Produce	json
//	@Param		id	path		string	true	"release id"
//	@Success	200	{object}	releaseDetailResponse
//	@Failure	404	{object}	response.ApiError
//	@Router		/api/console/release/{id} [get]
func GetReleaseHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		r, err := findReleaseOrNotFound(do, id)
		if err != nil {
			response.BadRequestWithMessage(c, err.Error())
			return
		}
		if r == nil {
			response.NotFoundWithMessage(c, "Release not found.")
			return
		}
		linked, err := do.ListEntriesByReleaseID(id, false)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		entries := make([]releaseEntrySummary, 0, len(linked))
		for _, e := range linked {
			entries = append(entries, releaseEntrySummary{
				ID:         e.ID,
				Title:      e.Title,
				IsInternal: e.IsInternal,
			})
		}
		response.Success(c, releaseDetailResponse{
			ReleaseResponse: api.BuildRelease(*r),
			Entries:         entries,
		})
	}
}
