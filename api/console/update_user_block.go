package console

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// updateUserBlockRequest toggles the IsBlocked flag on a user. A blocked
// user keeps their roles but cannot log in or hold a session — useful when
// an admin needs to suspend access without losing the role assignment.
type updateUserBlockRequest struct {
	IsBlocked bool `json:"isBlocked" example:"true"`
} //@name consoleUpdateUserBlockRequest

// UpdateUserBlockHandler sets the user's block flag. Admin-only. Refuses to
// block the caller themselves (foot-gun: locks them out of the Console even
// if other admins exist) and refuses to block the last active admin.
//
//	@ID			console-update-user-block
//	@Summary	Block or unblock a user
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"User ID"
//	@Param		request	body		updateUserBlockRequest	true	"Block flag"
//	@Success	200		{object}	administratorResponse
//	@Failure	400		{object}	response.ApiError
//	@Failure	404		{object}	response.ApiError
//	@Failure	409		{object}	response.ApiError
//	@Router		/api/console/user/{id}/block [patch]
func UpdateUserBlockHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.BadRequestWithMessage(c, "User id is required.")
			return
		}
		var req updateUserBlockRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}

		caller := middlewares.CurrentUser(c)
		if caller != nil && caller.ID == id && req.IsBlocked {
			response.ConflictWithMessage(c, "You cannot block your own account.")
			return
		}

		target, err := do.FindUserByID(id)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if target == nil || target.IsDeleted {
			response.NotFoundWithMessage(c, "User not found.")
			return
		}

		if target.IsBlocked == req.IsBlocked {
			response.Success(c, userToAdministratorResponse(*target))
			return
		}

		// Last-admin guard mirrors the role-update path: blocking the only
		// remaining active admin would lock everyone out of the Console.
		if req.IsBlocked && target.IsAdmin() && !target.IsBlocked {
			active, err := do.CountActiveAdmins()
			if err != nil {
				response.SystemError(c, err)
				return
			}
			if active <= 1 {
				response.ConflictWithMessage(c, "At least one active administrator must remain.")
				return
			}
		}

		target.IsBlocked = req.IsBlocked
		target.UpdatedAt = time.Now().UTC()
		if err := do.UpdateUser(target); err != nil {
			response.SystemError(c, err)
			return
		}
		response.Success(c, userToAdministratorResponse(*target))
	}
}
