// Package middlewares contains the Gin middleware factories for Mixdive.
//
// Two flavors live here: business-rule gates (RequireSetupCompleted,
// RequirePortalReadAccess) and the cookie-backed auth pipeline
// (AttachUser + RequireAdmin) that protects the Console.
package middlewares

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// RequirePortalReadAccessMiddleware gates Portal read endpoints.
// Authenticated visitors pass; everyone else gets 401. Portal access is
// always authenticated — there is no anonymous read path.
func RequirePortalReadAccessMiddleware(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentUser(c) != nil {
			c.Next()
			return
		}
		response.UnauthorizedErrorWithMessage(c, "Authentication required.")
	}
}

// RequirePortalWriteAccessMiddleware gates Portal write endpoints (entry
// submission + voting). When custom auth is on the visitor must be
// authenticated; otherwise writes stay anonymous (v0.1 behavior).
func RequirePortalWriteAccessMiddleware(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentUser(c) != nil {
			c.Next()
			return
		}
		s, err := do.GetSettings()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if s != nil && s.Portal.CustomAuthEnabled {
			response.UnauthorizedErrorWithMessage(c, "Authentication required.")
			return
		}
		c.Next()
	}
}
