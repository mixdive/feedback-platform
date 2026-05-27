package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// updateCommentRequest is the body for PATCH
// /api/console/entry/{id}/comment/{commentId}.
//
// Sparse update: only IsInternal can be flipped today. RequireConsoleAccess
// upstream guarantees an admin or editor is making the call — both roles
// share the same edit privilege on this flag.
type updateCommentRequest struct {
	IsInternal *bool `json:"isInternal,omitempty"`
} //@name consoleUpdateCommentRequest

// UpdateCommentHandler patches a single comment from the Console.
//
//	@ID			console-update-comment
//	@Summary	Update comment (admin)
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		id			path		string					true	"entry id"
//	@Param		commentId	path		string					true	"comment id"
//	@Param		request		body		updateCommentRequest	true	"Patch"
//	@Success	200			{object}	api.CommentResponse
//	@Failure	404			{object}	response.ApiError
//	@Router		/api/console/entry/{id}/comment/{commentId} [patch]
func UpdateCommentHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		entryID := c.Param("id")
		commentID := c.Param("commentId")
		var req updateCommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		existing, err := do.FindCommentByID(commentID)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if existing == nil || existing.EntryID != entryID {
			response.NotFoundWithMessage(c, "Comment not found.")
			return
		}
		if req.IsInternal != nil && *req.IsInternal != existing.IsInternal {
			if err := do.SetCommentIsInternal(commentID, *req.IsInternal); err != nil {
				response.SystemError(c, err)
				return
			}
		}
		updated, err := do.FindCommentByID(commentID)
		if err != nil || updated == nil {
			response.SystemError(c, err)
			return
		}
		authors, err := api.LoadEntryCreators(do, []string{updated.UserID})
		if err != nil {
			response.SystemError(c, err)
			return
		}
		response.Success(c, api.BuildCommentResponse(*updated, authors))
	}
}
