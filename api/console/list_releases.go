package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// releaseListItem extends the cross-surface release projection with
// the "entries linked to this release" count rendered next to each
// row in the Console settings list, plus the per-entry-type breakdown
// so the page can render one chip per type without a second round-trip.
type releaseListItem struct {
	api.ReleaseResponse
	EntryCount      int64                   `json:"entryCount"`
	EntryTypeCounts entryTypeCountsResponse `json:"entryTypeCounts"`
} //@name ConsoleReleaseListItem

// releaseListResponse is the envelope for release lists.
type releaseListResponse struct {
	Data []releaseListItem `json:"data"`
} //@name ConsoleReleaseList

// ListReleasesHandler returns every release ordered newest first.
// Editors AND admins can read — assigning entries to a release lives
// on the entry edit form which both roles use.
//
//	@ID			console-list-releases
//	@Summary	List releases
//	@Tags		Console
//	@Produce	json
//	@Success	200	{object}	releaseListResponse
//	@Router		/api/console/release [get]
func ListReleasesHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		releases, err := do.ListReleases()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		typeCounts, err := do.CountEntriesPerReleaseByEntryType()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		out := make([]releaseListItem, 0, len(releases))
		for _, r := range releases {
			tc := typeCounts[r.ID]
			out = append(out, releaseListItem{
				ReleaseResponse: api.BuildRelease(r),
				EntryCount:      tc.Total,
				EntryTypeCounts: entryTypeCountsResponse{
					Total:          tc.Total,
					FeatureRequest: tc.FeatureRequest,
					Bug:            tc.Bug,
					Support:        tc.Support,
					Other:          tc.Other,
				},
			})
		}
		response.Success(c, releaseListResponse{Data: out})
	}
}
