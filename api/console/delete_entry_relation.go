package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// DeleteEntryRelationHandler removes a single relation between two
// entries. Symmetric — the mirrored pair on the peer entry is also
// pulled in the same call. Idempotent: removing a non-existent link
// returns the entry unchanged.
//
//	@ID			console-delete-entry-relation
//	@Summary	Remove entry relation (admin)
//	@Tags		Console
//	@Produce	json
//	@Param		id			path		string	true	"entry id"
//	@Param		peerEntryId	path		string	true	"peer entry id"
//	@Success	200			{object}	entryResponse
//	@Failure	404			{object}	response.ApiError
//	@Router		/api/console/entry/{id}/relation/{peerEntryId} [delete]
func DeleteEntryRelationHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		peerID := c.Param("peerEntryId")
		entry, err := do.FindEntryByID(id)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if entry == nil {
			response.NotFoundWithMessage(c, "Entry not found.")
			return
		}
		// Detect whether the pair actually exists before the call so
		// the activity log skips no-op delete clicks. Captures the
		// type so the activity row can render "Removed duplicate of X".
		removedType := ""
		for _, r := range entry.Relations {
			if r.EntryID == peerID {
				removedType = string(r.Type)
				break
			}
		}
		if err := do.RemoveEntryRelation(id, peerID); err != nil {
			response.SystemError(c, err)
			return
		}
		if removedType != "" {
			actorID := ""
			if u := middlewares.CurrentUser(c); u != nil {
				actorID = u.ID
			}
			if err := recordAdminActivity(do, id, actorID, models.ActivityTypeRelationRemoved, removedType, "", peerID); err != nil {
				response.SystemError(c, err)
				return
			}
			if err := recordAdminActivity(do, peerID, actorID, models.ActivityTypeRelationRemoved, removedType, "", id); err != nil {
				response.SystemError(c, err)
				return
			}
		}
		writeEntryDetailResponse(c, do, id)
	}
}
