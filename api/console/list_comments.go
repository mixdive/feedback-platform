package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// ListCommentsHandler returns every comment on an entry, oldest first.
//
//	@ID			console-list-comments
//	@Summary	List comments on an entry (admin)
//	@Tags		Console
//	@Produce	json
//	@Param		id	path		string	true	"entry id"
//	@Success	200	{object}	api.CommentListResponse
//	@Failure	404	{object}	response.ApiError
//	@Router		/api/console/entry/{id}/comment [get]
func ListCommentsHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		entry, err := do.FindEntryByID(id)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if entry == nil {
			response.NotFoundWithMessage(c, "Entry not found.")
			return
		}
		comments, err := do.ListCommentsByEntryID(id, true)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		authors, team, err := api.LoadCommentAuthors(do, comments)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		out := make([]api.CommentResponse, 0, len(comments))
		for _, cm := range comments {
			out = append(out, api.BuildCommentResponse(cm, authors, team))
		}
		response.Success(c, api.CommentListResponse{Data: out})
	}
}
