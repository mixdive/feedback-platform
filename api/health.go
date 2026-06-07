package api

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// healthResponse is the body returned by /health.
type healthResponse struct {
	Status string `json:"status" example:"ok"`
} //@name Health

// HealthHandler reports both process liveness and Mongo reachability in one
// endpoint. Returns 200 when the server is up AND can talk to Mongo, 503
// otherwise.
//
//	@ID			system-health
//	@Summary	Health probe
//	@Description	Returns 200 when the server is running and Mongo is reachable.
//	@Tags		Health
//	@Produce	json
//	@Success	200	{object}	healthResponse
//	@Failure	503	{object}	response.ApiError
//	@Router		/health [get]
func HealthHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := do.GetSettings(); err != nil {
			response.UnderMaintenance(c)
			return
		}
		response.Success(c, healthResponse{Status: "ok"})
	}
}
