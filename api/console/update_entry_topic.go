package console

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// updateEntryTopicRequest is the body for PATCH
// /api/console/entry-topic/{id}.
//
// Sparse: only fields present on the wire are applied. Pointer to
// "" blanks the description/color; nil pointer leaves them alone.
type updateEntryTopicRequest struct {
	Title       *string `json:"title,omitempty"        binding:"omitempty,min=1,max=60"`
	Description *string `json:"description,omitempty"  binding:"omitempty,max=240"`
	Color       *string `json:"color,omitempty"        binding:"omitempty,max=32"`
	SortOrder   *int    `json:"sortOrder,omitempty"`
} //@name consoleUpdateEntryTopicRequest

// UpdateEntryTopicHandler patches a single topic. Admin-only.
//
//	@ID			console-update-entry-topic
//	@Summary	Update entry topic (admin)
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"topic id"
//	@Param		request	body		updateEntryTopicRequest	true	"Patch"
//	@Success	200		{object}	EntryTopicResponse
//	@Failure	404		{object}	response.ApiError
//	@Failure	409		{object}	response.ApiError
//	@Router		/api/console/entry-topic/{id} [patch]
func UpdateEntryTopicHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		existing, err := findTopicOrNotFound(do, id)
		if err != nil {
			response.BadRequestWithMessage(c, err.Error())
			return
		}
		if existing == nil {
			response.NotFoundWithMessage(c, "Topic not found.")
			return
		}
		var req updateEntryTopicRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		if req.Title != nil && !strings.EqualFold(*req.Title, existing.Title) {
			dup, err := do.FindEntryTopicByTitle(*req.Title)
			if err != nil {
				response.SystemError(c, err)
				return
			}
			if dup != nil && dup.ID != id {
				response.ConflictWithMessage(c, "A topic with this title already exists.")
				return
			}
		}
		if err := do.UpdateEntryTopicFields(id, req.Title, req.Description, req.Color, req.SortOrder); err != nil {
			response.SystemError(c, err)
			return
		}
		updated, err := do.FindEntryTopicByID(id)
		if err != nil || updated == nil {
			response.SystemError(c, err)
			return
		}
		response.Success(c, BuildEntryTopic(*updated))
	}
}
