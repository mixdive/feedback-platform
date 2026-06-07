package middlewares

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// userContextKey is the gin context key under which the resolved user is
// stashed by AttachUserMiddleware.
const userContextKey = "user"

// AttachUserMiddleware reads the session cookie, resolves it via
// DataOperations, and stashes the user on the gin context. It never aborts
// — RequireUser/RequireAdmin do that. This split lets the same middleware
// run on both anonymous-allowed and auth-only routes.
func AttachUserMiddleware(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ReadSessionToken(c)
		if token == "" {
			c.Next()
			return
		}
		user, err := do.ResolveSession(token)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if user != nil {
			c.Set(userContextKey, user)
		}
		c.Next()
	}
}

// CurrentUser returns the user attached by AttachUserMiddleware, or nil.
func CurrentUser(c *gin.Context) *models.User {
	v, ok := c.Get(userContextKey)
	if !ok {
		return nil
	}
	u, _ := v.(*models.User)
	return u
}

// RequireAdminMiddleware gates routes that require an authenticated admin.
// 401 if no session, 403 if the resolved user lacks the admin role. Used for
// endpoints that touch the administrator roster itself; everything else in
// the Console runs under RequireConsoleAccessMiddleware.
func RequireAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil {
			response.UnauthorizedErrorWithMessage(c, "Authentication required.")
			return
		}
		if !u.IsAdmin() {
			response.ForbiddenErrorWithMessage(c, "Admin role required.")
			return
		}
		c.Next()
	}
}

// RequireConsoleAccessMiddleware gates the bulk of the Console — settings,
// feedback, anything that isn't admin-roster management. Admins and editors
// both pass; anonymous and role-less users get the same 401/403 split as
// RequireAdmin.
func RequireConsoleAccessMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil {
			response.UnauthorizedErrorWithMessage(c, "Authentication required.")
			return
		}
		if !u.HasConsoleAccess() {
			response.ForbiddenErrorWithMessage(c, "Console access required.")
			return
		}
		c.Next()
	}
}

// RequireUserMiddleware gates routes that need *any* authenticated user
// (no role check). Voting is the primary v0.1 caller — the per-user vote
// uniqueness rule means anonymous voting is meaningless even when the
// portal otherwise allows anonymous writes.
func RequireUserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentUser(c) == nil {
			response.UnauthorizedErrorWithMessage(c, "Authentication required.")
			return
		}
		c.Next()
	}
}
