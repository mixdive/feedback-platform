package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// AddVoteHandler toggles the calling admin's vote on an entry. Admins
// vote under the same per-user uniqueness rule as Portal users; an admin
// who has voted from either surface sees the entry as voted on the
// other. RequireAdmin upstream guarantees a user is attached.
//
//	@ID			console-add-vote
//	@Summary	Toggle vote on an entry (admin)
//	@Tags		Console
//	@Produce	json
//	@Param		id	path		string	true	"entry id"
//	@Success	200	{object}	entryResponse
//	@Failure	404	{object}	response.ApiError
//	@Router		/api/console/entry/{id}/vote [post]
func AddVoteHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := middlewares.CurrentUser(c)
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
		existing, err := do.FindVote(u.ID, id)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		var voted bool
		if existing != nil {
			if err := do.DeleteVote(u.ID, id); err != nil {
				response.SystemError(c, err)
				return
			}
			if err := do.IncrementEntryVoteCount(id, -1); err != nil {
				response.SystemError(c, err)
				return
			}
			voted = false
		} else {
			v := models.NewVote()
			v.UserID = u.ID
			v.EntryID = id
			if err := do.InsertVote(v); err != nil {
				response.SystemError(c, err)
				return
			}
			if err := do.IncrementEntryVoteCount(id, 1); err != nil {
				response.SystemError(c, err)
				return
			}
			voted = true
		}
		updated, err := do.FindEntryByID(id)
		if err != nil || updated == nil {
			response.SystemError(c, err)
			return
		}
		creators, err := api.LoadEntryCreators(do, []string{updated.UserID})
		if err != nil {
			response.SystemError(c, err)
			return
		}
		topics, err := LoadEntryTopics(do, updated.TopicIDs)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		releases, err := api.LoadReleases(do, []string{updated.ReleaseID})
		if err != nil {
			response.SystemError(c, err)
			return
		}
		relations, err := resolveRelationsForEntry(do, *updated)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		response.Success(c, entryToResponse(*updated, creators, topics, releases, voted, relations))
	}
}
