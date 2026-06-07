package console

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// updateUserRolesRequest is the body for PATCH /api/console/user/:id/roles.
//
// Set-based, not additive: the caller sends the desired final state of the
// user's roles array. An empty array is a valid value — it strips every role
// from the user, which removes them from the Team listing and turns them
// back into a portal-only normal user.
type updateUserRolesRequest struct {
	Roles []string `json:"roles" validate:"required" example:"admin,editor"`
} //@name consoleUpdateUserRolesRequest

// UpdateUserRolesHandler replaces a user's role array. Admin-only. Refuses
// the change when it would leave the deployment without any active admin.
//
//	@ID			console-update-user-roles
//	@Summary	Update a user's roles
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string					true	"User ID"
//	@Param		request	body		updateUserRolesRequest	true	"Desired roles"
//	@Success	200		{object}	administratorResponse
//	@Failure	400		{object}	response.ApiError
//	@Failure	404		{object}	response.ApiError
//	@Failure	409		{object}	response.ApiError
//	@Router		/api/console/user/{id}/roles [patch]
func UpdateUserRolesHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.BadRequestWithMessage(c, "User id is required.")
			return
		}
		var req updateUserRolesRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}

		nextRoles, err := normalizeRoles(req.Roles)
		if err != nil {
			response.BadRequestWithMessage(c, err.Error())
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

		// Last-admin guard: CountActiveAdmins includes the target, so the
		// "==1 and target loses admin" branch is the exact condition that
		// would lock the Console out. The check is skipped when the target
		// either isn't currently a counted admin or keeps the role.
		wasActiveAdmin := target.IsAdmin() && !target.IsBlocked
		willBeAdmin := containsRole(nextRoles, models.RoleAdmin)
		if wasActiveAdmin && !willBeAdmin {
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

		target.Roles = nextRoles
		target.UpdatedAt = time.Now().UTC()
		if err := do.UpdateUser(target); err != nil {
			response.SystemError(c, err)
			return
		}
		response.Success(c, userToAdministratorResponse(*target))
	}
}

// normalizeRoles validates and de-duplicates the requested role list. Returns
// a clean []models.UserRole or an error naming the first invalid value.
func normalizeRoles(in []string) ([]models.UserRole, error) {
	out := make([]models.UserRole, 0, len(in))
	seen := map[models.UserRole]struct{}{}
	for _, raw := range in {
		role := models.UserRole(raw)
		switch role {
		case models.RoleAdmin, models.RoleEditor:
			if _, dup := seen[role]; dup {
				continue
			}
			seen[role] = struct{}{}
			out = append(out, role)
		default:
			return nil, errors.New("Unknown role: " + raw)
		}
	}
	return out, nil
}

func containsRole(roles []models.UserRole, want models.UserRole) bool {
	for _, r := range roles {
		if r == want {
			return true
		}
	}
	return false
}
