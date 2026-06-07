package portal

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// changelogEntrySummary is the lightweight projection of an entry
// rendered under each release on the Portal changelog page. Carries
// only what the changelog needs (linkable id, title, optional
// description preview). Internal entries never appear here — the
// upstream query filters them out.
type changelogEntrySummary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
} //@name PortalChangelogEntry

// changelogItem is one release card on the Portal changelog page.
// Mirrors the Checkly layout: date header, version label, title,
// markdown body, then the list of entries that shipped in the
// release.
type changelogItem struct {
	api.ReleaseResponse
	Entries []changelogEntrySummary `json:"entries"`
} //@name PortalChangelogItem

// changelogListResponse is the envelope for the Portal changelog
// page. Releases are returned newest first.
type changelogListResponse struct {
	Data []changelogItem `json:"data"`
} //@name PortalChangelogList

// ListChangelogHandler returns every release (planned and completed)
// with its linked entries, ordered by release date — newest first.
// Powers the Portal /changelog page. Planned releases are surfaced
// alongside completed ones so visitors see both shipped work and
// upcoming schedule on the same timeline; the badge state on each
// item lets the UI render the distinction.
//
//	@ID			portal-list-changelog
//	@Summary	List changelog releases
//	@Tags		Portal
//	@Produce	json
//	@Success	200	{object}	changelogListResponse
//	@Router		/api/portal/changelog [get]
func ListChangelogHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		releases, err := do.ListReleases()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		out := make([]changelogItem, 0, len(releases))
		for _, r := range releases {
			linked, err := do.ListEntriesByReleaseID(r.ID, true)
			if err != nil {
				response.SystemError(c, err)
				return
			}
			entries := make([]changelogEntrySummary, 0, len(linked))
			for _, e := range linked {
				entries = append(entries, changelogEntrySummary{
					ID:          e.ID,
					Title:       e.Title,
					Description: e.Description,
				})
			}
			out = append(out, changelogItem{
				ReleaseResponse: api.BuildRelease(r),
				Entries:         entries,
			})
		}
		response.Success(c, changelogListResponse{Data: out})
	}
}
