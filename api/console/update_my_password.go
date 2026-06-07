package console

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// updateMyPasswordRequest carries the credentials for POST /api/console/me/password.
// CurrentPassword is required so a hijacked session cannot silently rotate
// the password — the rotator must still know the live secret.
type updateMyPasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required,min=1"`
	NewPassword     string `json:"newPassword"     binding:"required,min=10"`
} //@name consoleUpdateMyPasswordRequest

// UpdateMyPasswordHandler rotates the signed-in user's argon2id-hashed
// password. Refuses for users who don't have an Email account (custom-auth
// only).
//
//	@ID			console-update-my-password
//	@Summary	Update my password
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		request	body	updateMyPasswordRequest	true	"Credentials"
//	@Success	204
//	@Failure	400	{object}	response.ApiError
//	@Failure	401	{object}	response.ApiError
//	@Router		/api/console/me/password [post]
func UpdateMyPasswordHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := middlewares.CurrentUser(c)
		if u == nil {
			response.UnauthorizedErrorWithMessage(c, "Authentication required.")
			return
		}
		var req updateMyPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}

		emailAccount, ok := u.EmailAccount()
		if !ok || emailAccount.Password == "" {
			response.BadRequestWithMessage(c, "This account does not use a password.")
			return
		}

		match, err := api.VerifyPassword(req.CurrentPassword, emailAccount.Password)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if !match {
			response.UnauthorizedErrorWithMessage(c, "Current password is incorrect.")
			return
		}

		newHash, err := api.HashPassword(req.NewPassword)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		emailAccount.Password = newHash
		u.SetAccount(models.UserAccountTypeEmail, emailAccount)
		u.UpdatedAt = time.Now().UTC()
		if err := do.UpdateUser(u); err != nil {
			response.SystemError(c, err)
			return
		}
		response.NoContent(c)
	}
}
