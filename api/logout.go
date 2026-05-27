package api

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// LogoutHandler invalidates the caller's session cookie. Idempotent: hitting
// /api/logout without a session is still a 204.
//
//	@ID			auth-logout
//	@Summary	Log out
//	@Description	Deletes the active session and clears the cookie. Idempotent.
//	@Tags		Auth
//	@Success	204
//	@Router		/api/logout [post]
func LogoutHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := middlewares.ReadSessionToken(c)
		if token != "" {
			if err := do.DeleteSessionByToken(token); err != nil {
				response.SystemError(c, err)
				return
			}
		}
		middlewares.ClearSessionCookie(c)
		response.NoContent(c)
	}
}
