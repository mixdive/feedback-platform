package console

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/pkg/aianalyzer"
	"github.com/mixdive/feedback-platform/pkg/storage"
)

// updateAISettingsRequest is the body for PATCH /api/console/settings/ai.
//
// Sparse update: only fields present on the wire are applied. Pointers
// here exist purely so the JSON binder can distinguish "not provided"
// from "explicitly empty".
//
// APIKey is a write-only field — the GET endpoint never returns the
// cleartext key (it returns hasApiKey + apiKeyPreview instead). To
// clear the stored key, send an empty string.
type updateAISettingsRequest struct {
	Enabled *bool   `json:"enabled,omitempty"`
	APIKey  *string `json:"apiKey,omitempty"`
	Model   *string `json:"model,omitempty"`
} //@name consoleUpdateAISettingsRequest

// UpdateAISettingsHandler patches the AI sub-block on the singleton
// settings document. After a successful write it calls worker.Refresh()
// so the in-memory snapshot picks up the new configuration on the next
// poll tick — no server restart required.
//
// Validation rule: AI cannot be enabled without an API key set. The
// post-patch state of both fields is computed from the current
// settings + the incoming patch so the admin can flip the toggle on
// without re-sending an unchanged key.
//
//	@ID			console-update-ai-settings
//	@Summary	Update AI settings (admin)
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		request	body		updateAISettingsRequest	true	"Patch"
//	@Success	200		{object}	settingsResponse
//	@Failure	400		{object}	response.ApiError
//	@Router		/api/console/settings/ai [patch]
func UpdateAISettingsHandler(do dataoperations.Store, worker *aianalyzer.Worker, store *storage.Holder) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req updateAISettingsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		s, err := do.GetSettings()
		if err != nil || s == nil {
			response.SystemError(c, err)
			return
		}

		// Compute post-patch state for the validation rule.
		nextEnabled := s.AI.Enabled
		if req.Enabled != nil {
			nextEnabled = *req.Enabled
		}
		nextAPIKey := s.AI.APIKey
		if req.APIKey != nil {
			nextAPIKey = strings.TrimSpace(*req.APIKey)
		}
		if nextEnabled && nextAPIKey == "" {
			response.BadRequestWithMessage(c, "An API key is required to enable AI analysis.")
			return
		}

		set := map[string]any{}
		if req.Enabled != nil {
			set["ai.enabled"] = *req.Enabled
		}
		if req.APIKey != nil {
			set["ai.apikey"] = nextAPIKey
		}
		if req.Model != nil {
			set["ai.model"] = strings.TrimSpace(*req.Model)
		}
		if len(set) > 0 {
			if err := do.UpdateSettings(set); err != nil {
				response.SystemError(c, err)
				return
			}
		}
		// Refresh the worker's cached snapshot so the next tick sees
		// the new config. Cheap (one Mongo doc read).
		worker.Refresh()

		updated, err := do.GetSettings()
		if err != nil || updated == nil {
			response.SystemError(c, err)
			return
		}
		response.Success(c, newSettingsResponse(updated, true, store))
	}
}
