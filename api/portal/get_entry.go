package portal

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// GetEntryHandler returns a single entry.
//
//	@ID			portal-get-entry
//	@Summary	Get entry
//	@Tags		Portal
//	@Produce	json
//	@Param		id	path		string	true	"entry id"
//	@Success	200	{object}	entryResponse
//	@Failure	404	{object}	response.ApiError
//	@Router		/api/portal/entry/{id} [get]
func GetEntryHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		e, err := do.FindEntryByID(id)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if e == nil {
			response.NotFoundWithMessage(c, "Entry not found.")
			return
		}
		viewer := middlewares.CurrentUser(c)
		if e.IsInternal {
			response.NotFoundWithMessage(c, "Entry not found.")
			return
		}
		creators, err := api.LoadEntryCreators(do, []string{e.UserID})
		if err != nil {
			response.SystemError(c, err)
			return
		}
		releases, err := api.LoadReleases(do, []string{e.ReleaseID})
		if err != nil {
			response.SystemError(c, err)
			return
		}
		isVoted := false
		if viewer != nil {
			v, err := do.FindVote(viewer.ID, id)
			if err != nil {
				response.SystemError(c, err)
				return
			}
			isVoted = v != nil
		}
		response.Success(c, entryToResponse(*e, creators, releases, isVoted))
	}
}
