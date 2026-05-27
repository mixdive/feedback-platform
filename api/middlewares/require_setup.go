package middlewares

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// RequireSetupCompletedMiddleware short-circuits with 503 setup_required if
// first-run setup hasn't been completed. Apply to every API group except
// /api/setup itself; the frontend uses the resulting 503 to redirect to the
// setup form.
func RequireSetupCompletedMiddleware(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := do.GetSettings()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if s == nil || !s.SetupCompleted {
			response.SetupRequired(c)
			return
		}
		c.Next()
	}
}
