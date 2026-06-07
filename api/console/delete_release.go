package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// DeleteReleaseHandler removes a release and cascade-clears its ID
// from every entry that referenced it. Admin-only.
//
// The cascade runs BEFORE the release delete so a transient failure
// leaves the system in a consistent state: release still exists,
// every entry still references it. The reverse order would leave
// dangling release IDs across the entries collection that no admin
// tool surfaces.
//
//	@ID			console-delete-release
//	@Summary	Delete release (admin)
//	@Tags		Console
//	@Param		id	path	string	true	"release id"
//	@Success	204
//	@Failure	404	{object}	response.ApiError
//	@Router		/api/console/release/{id} [delete]
func DeleteReleaseHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		existing, err := findReleaseOrNotFound(do, id)
		if err != nil {
			response.BadRequestWithMessage(c, err.Error())
			return
		}
		if existing == nil {
			response.NotFoundWithMessage(c, "Release not found.")
			return
		}
		if err := do.RemoveReleaseIDFromAllEntries(id); err != nil {
			response.SystemError(c, err)
			return
		}
		if err := do.DeleteRelease(id); err != nil {
			response.SystemError(c, err)
			return
		}
		response.NoContent(c)
	}
}
