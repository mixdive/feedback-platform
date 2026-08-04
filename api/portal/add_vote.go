package portal

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// AddVoteHandler toggles the caller's vote on an entry. Posting once
// records the vote and bumps the entry's vote count; posting again with
// the same user removes the vote and decrements the count. RequireUser
// guarantees a user is attached, so the handler can always read it.
//
//	@ID			portal-add-vote
//	@Summary	Toggle vote on an entry
//	@Tags		Portal
//	@Produce	json
//	@Param		id	path		string	true	"entry id"
//	@Success	200	{object}	entryResponse
//	@Failure	401	{object}	response.ApiError
//	@Failure	404	{object}	response.ApiError
//	@Router		/api/portal/entry/{id}/vote [post]
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
		// Best-effort Slack notification, only when a vote was ADDED
		// (never on un-vote — a toggle-off isn't newsworthy) and never
		// for internal entries.
		if voted && !updated.IsInternal {
			noun := "votes"
			if updated.VoteCount == 1 {
				noun = "vote"
			}
			msg := fmt.Sprintf(":thumbsup: *New vote* by %s on %s — now %d %s",
				slackAuthorName(u),
				slackEntryRef(c, updated.ID, updated.Title),
				updated.VoteCount,
				noun,
			)
			notifySlack(do, func(s models.SlackIntegration) bool { return s.NotifyOnVote }, msg)
		}
		creators, err := api.LoadEntryCreators(do, []string{updated.UserID})
		if err != nil {
			response.SystemError(c, err)
			return
		}
		releases, err := api.LoadReleases(do, []string{updated.ReleaseID})
		if err != nil {
			response.SystemError(c, err)
			return
		}
		response.Success(c, entryToResponse(*updated, creators, releases, voted))
	}
}
