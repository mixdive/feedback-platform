package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// RequireUploadsEnabledMiddleware short-circuits with 403 when the
// admin has turned uploads off on the Settings page. Applied to the
// two POST /files routes (one per surface) so the handler itself does
// not have to know about the feature flag. Reads / deletes are
// intentionally NOT gated — links to previously-uploaded blobs keep
// working until the admin deletes them explicitly.
func RequireUploadsEnabledMiddleware(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := do.GetSettings()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if s == nil || !s.Uploads.Enabled {
			response.ErrorWithStatusCodeAndMessage(c, http.StatusForbidden, "File uploads are disabled.")
			return
		}
		c.Next()
	}
}
