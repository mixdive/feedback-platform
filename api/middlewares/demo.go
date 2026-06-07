package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/models"
)

// DemoUserMiddleware auto-authenticates every request as the supplied
// user. It replaces the cookie-backed AttachUserMiddleware when the
// server boots in DEMO mode: there is no Mongo and therefore no session
// store, so instead of resolving a cookie we stash a fixed synthetic
// admin on the context. Downstream gates (RequireConsoleAccess,
// RequireUser, RequirePortalReadAccess) then all pass, making both the
// Console and the Portal fully browsable with zero login friction.
func DemoUserMiddleware(u *models.User) gin.HandlerFunc {
	return func(c *gin.Context) {
		if u != nil {
			c.Set(userContextKey, u)
		}
		c.Next()
	}
}

// DemoReadOnlyMiddleware enforces the read-only guarantee of DEMO mode by
// rejecting every mutating request with a 403 before it reaches a
// handler. Safe (idempotent) methods pass through. This is the primary
// guard; the demo store's write methods returning ErrReadOnly is the
// belt-and-suspenders backstop.
func DemoReadOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
		default:
			response.ForbiddenErrorWithMessage(c, "This is a read-only demo — changes are disabled.")
		}
	}
}
