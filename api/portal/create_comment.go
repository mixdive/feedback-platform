package portal

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// createCommentRequest is the body for POST
// /api/portal/entry/{id}/comment.
type createCommentRequest struct {
	Body string `json:"body" binding:"required,min=1,max=10000"`
} //@name portalCreateCommentRequest

// CreateCommentHandler appends a comment to an entry. RequireUser
// upstream guarantees a user is attached — comments are never anonymous,
// even on portals that otherwise allow anonymous voting/submission.
//
//	@ID			portal-create-comment
//	@Summary	Add a comment to an entry
//	@Tags		Portal
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"entry id"
//	@Param		request	body		createCommentRequest	true	"Comment"
//	@Success	201		{object}	api.CommentResponse
//	@Failure	400		{object}	response.ApiError
//	@Failure	401		{object}	response.ApiError
//	@Failure	404		{object}	response.ApiError
//	@Router		/api/portal/entry/{id}/comment [post]
func CreateCommentHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req createCommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		entry, err := do.FindEntryByID(id)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if entry == nil {
			response.NotFoundWithMessage(c, "Entry not found.")
			return
		}
		u := middlewares.CurrentUser(c)
		if entry.IsInternal && entry.UserID != u.ID {
			response.NotFoundWithMessage(c, "Entry not found.")
			return
		}
		cm := models.NewComment()
		cm.EntryID = id
		cm.UserID = u.ID
		cm.Body = req.Body
		cm.IsInternal = false
		if err := do.InsertComment(cm); err != nil {
			response.SystemError(c, err)
			return
		}
		if err := do.IncrementEntryCommentCount(id, 1); err != nil {
			response.SystemError(c, err)
			return
		}
		authors := map[string]api.EntryCreator{u.ID: api.BuildEntryCreator(*u)}
		team := map[string]bool{}
		if u.HasConsoleAccess() {
			team[u.ID] = true
		}
		response.Created(c, api.BuildCommentResponse(*cm, authors, team))
	}
}
