package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/pkg/aianalyzer"
)

// GetAIQueueHandler returns a point-in-time snapshot of the AI
// analysis queue: how many entries are waiting per analyzer, how many
// are currently in-flight, and the worker's polling cadence.
//
// Numbers are cluster-wide truth (counted in Mongo), so on a
// multi-instance Cloud Run deployment every instance returns the same
// values.
//
//	@ID			console-get-ai-queue
//	@Summary	Get AI queue stats (admin)
//	@Tags		Console
//	@Produce	json
//	@Success	200	{object}	aianalyzer.Stats
//	@Router		/api/console/ai/queue [get]
func GetAIQueueHandler(worker *aianalyzer.Worker) gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Success(c, worker.Stats())
	}
}
