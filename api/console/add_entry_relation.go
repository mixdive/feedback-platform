package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// addEntryRelationRequest is the body for POST
// /api/console/entry/{id}/relation. Type is the kebab-case enum
// (`duplicate` / `related`); PeerEntryID points at the other entry the
// admin is linking. The link is symmetric — the same pair lands on
// both entries.
type addEntryRelationRequest struct {
	PeerEntryID string `json:"peerEntryId" binding:"required"`
	Type        string `json:"type"        binding:"required"`
} //@name consoleAddEntryRelationRequest

// AddEntryRelationHandler creates or updates a single relation between
// two entries. Symmetric — the mirrored pair lands on the peer entry
// in the same call. Idempotent: re-applying the same (peerEntryId,
// type) is a no-op. Re-applying a different type to the same peer
// overwrites the prior pair.
//
// Console-only: relations never appear on the Portal.
//
//	@ID			console-add-entry-relation
//	@Summary	Add entry relation (admin)
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"entry id"
//	@Param		request	body		addEntryRelationRequest		true	"Relation"
//	@Success	200		{object}	entryResponse
//	@Failure	400		{object}	response.ApiError
//	@Failure	404		{object}	response.ApiError
//	@Router		/api/console/entry/{id}/relation [post]
func AddEntryRelationHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req addEntryRelationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		if req.PeerEntryID == id {
			response.BadRequestWithMessage(c, "An entry cannot be related to itself.")
			return
		}
		relType, ok := validateRelationType(req.Type)
		if !ok {
			response.BadRequestWithMessage(c, "Unknown relation type.")
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
		peer, err := do.FindEntryByID(req.PeerEntryID)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if peer == nil {
			response.BadRequestWithMessage(c, "Peer entry not found.")
			return
		}
		// Track whether this is a genuine change so the activity log
		// doesn't pick up no-op re-clicks. Re-typing a relation still
		// counts as a change worth recording.
		hadSameType := false
		for _, r := range entry.Relations {
			if r.EntryID == req.PeerEntryID && r.Type == relType {
				hadSameType = true
				break
			}
		}
		if err := do.AddEntryRelation(id, req.PeerEntryID, relType); err != nil {
			response.SystemError(c, err)
			return
		}
		if !hadSameType {
			actorID := ""
			if u := middlewares.CurrentUser(c); u != nil {
				actorID = u.ID
			}
			// Symmetric activity rows: both entries' timelines show
			// the link from the perspective of "the other entry is
			// the target".
			if err := recordAdminActivity(do, id, actorID, models.ActivityTypeRelationAdded, "", string(relType), req.PeerEntryID); err != nil {
				response.SystemError(c, err)
				return
			}
			if err := recordAdminActivity(do, req.PeerEntryID, actorID, models.ActivityTypeRelationAdded, "", string(relType), id); err != nil {
				response.SystemError(c, err)
				return
			}
		}
		writeEntryDetailResponse(c, do, id)
	}
}
