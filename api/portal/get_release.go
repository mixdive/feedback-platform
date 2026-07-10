package portal

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// GetChangelogReleaseHandler returns one release plus its linked
// entries, used by the Portal release detail page that backs
// shareable links from social media. Both planned and completed
// releases are reachable so a "coming soon" announcement can link
// to a planned record.
//
// The :slug param is matched first against VersionName (so
// `/changelog/v1.0.0` resolves cleanly), then falls back to ID — so
// pre-existing UUID links keep working after an admin edits a
// version name.
//
//	@ID			portal-get-changelog-release
//	@Summary	Get changelog release
//	@Tags		Portal
//	@Produce	json
//	@Param		slug	path		string	true	"version name or release id"
//	@Success	200		{object}	changelogItem
//	@Failure	404		{object}	response.ApiError
//	@Router		/api/portal/changelog/{slug} [get]
func GetChangelogReleaseHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		if slug == "" {
			response.NotFoundWithMessage(c, "Release not found.")
			return
		}
		r, err := resolveChangelogRelease(do, slug)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if r == nil {
			response.NotFoundWithMessage(c, "Release not found.")
			return
		}
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
				Status:      api.BuildEntryStatus(e.Status),
			})
		}
		response.Success(c, changelogItem{
			ReleaseResponse: api.BuildRelease(*r),
			Entries:         entries,
		})
	}
}

// resolveChangelogRelease looks up a release by version name first,
// then by ID. Returns (nil, nil) when neither matches.
func resolveChangelogRelease(do dataoperations.Store, slug string) (*models.Release, error) {
	if r, err := do.FindReleaseByVersionName(slug); err != nil || r != nil {
		return r, err
	}
	return do.FindReleaseByID(slug)
}
