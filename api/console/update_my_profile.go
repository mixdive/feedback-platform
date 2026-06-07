package console

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// updateMyProfileRequest carries the editable fields on the My Profile page.
// Pointers let the binder distinguish "not provided" from "explicitly cleared"
// so the UI can drop the avatar (empty string) without also clearing the name.
type updateMyProfileRequest struct {
	Name     *string `json:"name,omitempty"`
	ImageURL *string `json:"imageUrl,omitempty"`
} //@name consoleUpdateMyProfileRequest

// UpdateMyProfileHandler updates the signed-in user's own display name and
// avatar URL. The change lands on the Email account when one exists,
// otherwise on the Custom account — whichever account currently powers the
// user's identity.
//
//	@ID			console-update-my-profile
//	@Summary	Update my profile
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		request	body		updateMyProfileRequest	true	"Patch"
//	@Success	200		{object}	myProfileResponse
//	@Failure	401		{object}	response.ApiError
//	@Router		/api/console/me [patch]
func UpdateMyProfileHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := middlewares.CurrentUser(c)
		if u == nil {
			response.UnauthorizedErrorWithMessage(c, "Authentication required.")
			return
		}
		var req updateMyProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}

		acctType, acct, ok := primaryAccountForEdit(u)
		if !ok {
			response.BadRequestWithMessage(c, "No editable account on this user.")
			return
		}

		if req.Name != nil {
			acct.Name = strings.TrimSpace(*req.Name)
		}
		if req.ImageURL != nil {
			acct.ImageURL = strings.TrimSpace(*req.ImageURL)
		}

		u.SetAccount(acctType, acct)
		u.UpdatedAt = time.Now().UTC()
		if err := do.UpdateUser(u); err != nil {
			response.SystemError(c, err)
			return
		}
		response.Success(c, newMyProfileResponse(u))
	}
}

// primaryAccountForEdit picks which account on the user receives display-field
// edits. Email account wins when present (the bootstrap admin path); the
// Custom account is the fallback for users who arrived via the portal auth
// provider and were later granted a Console role.
func primaryAccountForEdit(u *models.User) (models.UserAccountType, models.UserAccount, bool) {
	if a, ok := u.EmailAccount(); ok {
		return models.UserAccountTypeEmail, a, true
	}
	if a, ok := u.CustomAccount(); ok {
		return models.UserAccountTypeCustom, a, true
	}
	return "", models.UserAccount{}, false
}
