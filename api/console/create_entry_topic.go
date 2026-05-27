package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// createEntryTopicRequest is the body for POST
// /api/console/entry-topic.
type createEntryTopicRequest struct {
	Title       string `json:"title"        binding:"required,min=1,max=60"`
	Description string `json:"description,omitempty"        binding:"max=240"`
	Color       string `json:"color,omitempty"              binding:"max=32"`
	SortOrder   int    `json:"sortOrder,omitempty"`
} //@name consoleCreateEntryTopicRequest

// CreateEntryTopicHandler creates a new entry topic. Admin-only —
// editors may apply topics to entries but cannot manage the topic
// set itself.
//
//	@ID			console-create-entry-topic
//	@Summary	Create entry topic (admin)
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		request	body		createEntryTopicRequest	true	"Topic"
//	@Success	201		{object}	EntryTopicResponse
//	@Failure	400		{object}	response.ApiError
//	@Failure	409		{object}	response.ApiError
//	@Router		/api/console/entry-topic [post]
func CreateEntryTopicHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createEntryTopicRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		existing, err := do.FindEntryTopicByTitle(req.Title)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if existing != nil {
			response.ConflictWithMessage(c, "A topic with this title already exists.")
			return
		}
		t := models.NewEntryTopic()
		t.Title = req.Title
		t.Description = req.Description
		t.Color = req.Color
		t.SortOrder = req.SortOrder
		if err := do.InsertEntryTopic(t); err != nil {
			response.SystemError(c, err)
			return
		}
		response.Created(c, BuildEntryTopic(*t))
	}
}
