package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// GetEntryHandler returns one entry (admin view).
//
//	@ID			console-get-entry
//	@Summary	Get entry (admin)
//	@Tags		Console
//	@Produce	json
//	@Param		id	path		string	true	"entry id"
//	@Success	200	{object}	entryResponse
//	@Failure	404	{object}	response.ApiError
//	@Router		/api/console/entry/{id} [get]
func GetEntryHandler(do dataoperations.Store) gin.HandlerFunc {
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
		creators, err := api.LoadEntryCreators(do, []string{e.UserID})
		if err != nil {
			response.SystemError(c, err)
			return
		}
		topics, err := LoadEntryTopics(do, e.TopicIDs)
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
		if u := middlewares.CurrentUser(c); u != nil {
			v, err := do.FindVote(u.ID, id)
			if err != nil {
				response.SystemError(c, err)
				return
			}
			isVoted = v != nil
		}
		relations, err := resolveRelationsForEntry(do, *e)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		response.Success(c, entryToResponse(*e, creators, topics, releases, isVoted, relations))
	}
}
