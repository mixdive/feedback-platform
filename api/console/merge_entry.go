package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// mergeEntryRequest is the body for POST
// /api/console/entry/{id}/merge. TargetEntryID is the entry the
// caller is merging the path entry INTO — votes flow from path entry
// (source) to TargetEntryID (target), the source is moved to the
// cancel-target status, and a merged-into activity is appended on the
// source recording the merge.
type mergeEntryRequest struct {
	TargetEntryID string `json:"targetEntryId" binding:"required"`
} //@name consoleMergeEntryRequest

// MergeEntryHandler folds the path entry into TargetEntryID. Steps,
// run sequentially without a Mongo transaction (matches the rest of
// the code's eventual-consistency approach):
//
//  1. Adds a `duplicate` relation between source and target. Symmetric
//     pair lands on both entries via AddEntryRelation.
//  2. Migrates every Vote on source to target. Votes from users who
//     already voted on target are deleted as duplicates; the rest are
//     retargeted. Both entries' VoteCount is recomputed from the votes
//     collection so drift can't survive the merge.
//  3. Sets the source's status to EntryStatusCancelTarget so the merge
//     audit trail reads as "abandoned, not shipped".
//  4. Appends a merged-into activity to the source authored by the
//     calling admin/editor. TargetID points at the survivor; ToValue
//     carries the survivor's title so the Console can still render
//     a readable line if the target gets deleted later.
//
// Idempotent enough for repeat clicks — relation upsert is idempotent,
// vote migration leaves zero votes behind, status assignment is a
// straight write. A second merge call will append a second merged-into
// activity, which we accept as a small audit trail rather than a bug.
//
// Permission: covered by RequireConsoleAccessMiddleware on the route
// group — both admins and editors can merge.
//
//	@ID			console-merge-entry
//	@Summary	Merge entry into another (admin/editor)
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"source entry id"
//	@Param		request	body		mergeEntryRequest	true	"Merge target"
//	@Success	200		{object}	entryResponse
//	@Failure	400		{object}	response.ApiError
//	@Failure	404		{object}	response.ApiError
//	@Failure	500		{object}	response.ApiError
//	@Router		/api/console/entry/{id}/merge [post]
func MergeEntryHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		srcID := c.Param("id")
		var req mergeEntryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		if req.TargetEntryID == srcID {
			response.BadRequestWithMessage(c, "An entry cannot be merged into itself.")
			return
		}
		src, err := do.FindEntryByID(srcID)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if src == nil {
			response.NotFoundWithMessage(c, "Entry not found.")
			return
		}
		dst, err := do.FindEntryByID(req.TargetEntryID)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if dst == nil {
			response.BadRequestWithMessage(c, "Target entry not found.")
			return
		}

		if err := do.AddEntryRelation(srcID, req.TargetEntryID, models.EntryRelationTypeDuplicate); err != nil {
			response.SystemError(c, err)
			return
		}

		if _, err := do.MigrateVotesToEntry(srcID, req.TargetEntryID); err != nil {
			response.SystemError(c, err)
			return
		}
		srcCount, err := do.CountVotesForEntry(srcID)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if err := do.SetEntryVoteCount(srcID, srcCount); err != nil {
			response.SystemError(c, err)
			return
		}
		dstCount, err := do.CountVotesForEntry(req.TargetEntryID)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if err := do.SetEntryVoteCount(req.TargetEntryID, dstCount); err != nil {
			response.SystemError(c, err)
			return
		}

		if err := do.SetEntryStatus(srcID, models.EntryStatusCancelTarget); err != nil {
			response.SystemError(c, err)
			return
		}

		actorID := ""
		if u := middlewares.CurrentUser(c); u != nil {
			actorID = u.ID
		}
		// One merged-into activity captures the whole merge as a single
		// timeline event. The implicit relation-add and status flip are
		// side effects of the merge and intentionally not double-logged
		// — Console renders this row as the dominant signal.
		if err := recordAdminActivity(do, srcID, actorID, models.ActivityTypeMergedInto, "", dst.Title, req.TargetEntryID); err != nil {
			response.SystemError(c, err)
			return
		}

		writeEntryDetailResponse(c, do, srcID)
	}
}
