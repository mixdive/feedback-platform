package portal

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// configResponse is the public-facing Portal bootstrap config sourced from
// the singleton Settings document. The JWT signing key is intentionally
// NOT included — it is a server-side secret used only to verify inbound
// JWTs from the auth URL.
//
// EntryTypeTemplates is the admin-managed per-entry-type markdown
// template used to pre-fill the description field on the new-entry
// form. Keys are EntryType values ("feature-request", "bug", "support",
// "other"); empty string = no template, in which case the Portal falls
// back to the per-type placeholder copy.
type configResponse struct {
	ProjectName           string            `json:"projectName"`
	LogoURL               string            `json:"logoUrl,omitempty"`
	PrimaryColor          string            `json:"primaryColor,omitempty"`
	CustomAuthEnabled     bool              `json:"customAuthEnabled"`
	AuthURL               string            `json:"authUrl,omitempty"`
	CustomAuthButtonText  string            `json:"customAuthButtonText,omitempty"`
	EntryTypeTemplates    map[string]string `json:"entryTypeTemplates"`
	SupportRequestEnabled bool              `json:"supportRequestEnabled"`
	SupportRequestURL     string            `json:"supportRequestUrl,omitempty"`
	UploadsEnabled        bool              `json:"uploadsEnabled"`
} //@name Config

// GetConfigHandler returns the Portal bootstrap config. Setup-completion is
// already enforced by middleware, so settings are guaranteed present here.
//
//	@ID			portal-get-config
//	@Summary	Portal config
//	@Description	Returns project branding and the inbound auth URL (when custom auth is on).
//	@Tags		Portal
//	@Produce	json
//	@Success	200	{object}	configResponse
//	@Router		/api/portal/config [get]
func GetConfigHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := do.GetSettings()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		// Backfill the templates map on the wire so legacy deployments
		// that came up before EnsureFeedbackDefaults landed still
		// surface the bundled starter templates on the Portal.
		templates := s.Feedback.EntryTypeTemplates
		if templates == nil {
			templates = models.DefaultEntryTypeTemplates()
		}
		response.Success(c, configResponse{
			ProjectName:           s.ProjectName,
			LogoURL:               s.LogoURL,
			PrimaryColor:          s.PrimaryColor,
			CustomAuthEnabled:     s.Portal.CustomAuthEnabled,
			AuthURL:               s.Portal.AuthURL,
			CustomAuthButtonText:  s.Portal.CustomAuthButtonText,
			EntryTypeTemplates:    templates,
			SupportRequestEnabled: s.Feedback.SupportRequest.Enabled,
			SupportRequestURL:     s.Feedback.SupportRequest.URL,
			UploadsEnabled:        s.Uploads.Enabled,
		})
	}
}
