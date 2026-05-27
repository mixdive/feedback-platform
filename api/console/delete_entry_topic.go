package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// DeleteEntryTopicHandler removes a topic and cascade-removes its
// ID from every entry that referenced it. Admin-only.
//
// The cascade runs BEFORE the topic delete so a transient failure
// leaves the system in a consistent state: topic still exists,
// every entry still references it.
//
//	@ID			console-delete-entry-topic
//	@Summary	Delete entry topic (admin)
//	@Tags		Console
//	@Param		id	path	string	true	"topic id"
//	@Success	204
//	@Failure	404	{object}	response.ApiError
//	@Router		/api/console/entry-topic/{id} [delete]
func DeleteEntryTopicHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
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
		if err := do.RemoveTopicIDFromAllEntries(id); err != nil {
			response.SystemError(c, err)
			return
		}
		if err := do.DeleteEntryTopic(id); err != nil {
			response.SystemError(c, err)
			return
		}
		response.NoContent(c)
	}
}
