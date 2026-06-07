package console

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/storage"
)

// portalSettingsPayload is the admin-facing projection of PortalSettings.
// JWTPrivateKey is admin-only — non-admins (e.g. editors) see the rest of
// the page but the key is omitted from the wire entirely so the secret
// never reaches their browser.
type portalSettingsPayload struct {
	CustomAuthEnabled    bool   `json:"customAuthEnabled"`
	AuthURL              string `json:"authUrl"`
	CustomAuthButtonText string `json:"customAuthButtonText"`
	JWTPrivateKey        string `json:"jwtPrivateKey,omitempty"`
	GoogleAuthEnabled    bool   `json:"googleAuthEnabled"`
	GoogleClientID       string `json:"googleClientId"`
} //@name PortalSettings

// aiSettingsPayload is the admin-facing projection of AISettings. The
// Anthropic API key is NEVER sent on the wire in cleartext — only a
// boolean (HasAPIKey) and a masked preview ("•••• abcd") are returned.
// The PATCH endpoint accepts a new key via a separate field; that's
// the only direction the secret ever travels.
type aiSettingsPayload struct {
	Enabled          bool   `json:"enabled"`
	HasAPIKey        bool   `json:"hasApiKey"`
	APIKeyPreview    string `json:"apiKeyPreview,omitempty"`
	Model            string `json:"model"`
	LastAnalyzedAt   string `json:"lastAnalyzedAt,omitempty"`
	LastErrorAt      string `json:"lastErrorAt,omitempty"`
	LastErrorMessage string `json:"lastErrorMessage,omitempty"`
} //@name AISettings

// feedbackSettingsPayload is the admin-facing projection of
// FeedbackSettings. Exposes the per-user vote quota, the per-user
// feature-request quota, and the per-(entry-type, language)
// description templates. Outer key is EntryType ("feature-request",
// "bug", "support", "other"); inner key is a BCP-47 language code
// ("en", "tr"); inner value is the markdown template ("" = no
// template for that pair).
//
// DefaultEntryTypeTemplates is the bundled starter set, returned on
// every response so the Console "Reset to default" button can restore
// a template without a second round trip. Static — never reflects
// admin edits.
// supportRequestPayload is the admin-facing projection of
// SupportRequestSettings. Mirrors the wire shape exactly — the
// Portal config endpoint exposes the same pair under different field
// names for end-user-facing rendering.
type supportRequestPayload struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
} //@name SupportRequestSettings

type feedbackSettingsPayload struct {
	MaxVotesPerUser           int                          `json:"maxVotesPerUser"`
	MaxFeatureRequestsPerUser int                          `json:"maxFeatureRequestsPerUser"`
	EntryTypeTemplates        map[string]map[string]string `json:"entryTypeTemplates"`
	DefaultEntryTypeTemplates map[string]map[string]string `json:"defaultEntryTypeTemplates"`
	SupportRequest            supportRequestPayload        `json:"supportRequest"`
} //@name FeedbackSettings

// githubIntegrationPayload is the admin-facing projection of
// GitHubIntegration. The cleartext token is NEVER returned — only
// HasToken plus a short suffix preview ("…abcd"). The PATCH endpoint
// accepts a new token via a separate field; that's the only direction
// the secret travels.
//
// Active is the derived "fully wired and not kill-switched" flag,
// computed server-side so the UI doesn't have to repeat the rule.
type githubIntegrationPayload struct {
	Disabled     bool   `json:"disabled"`
	Owner        string `json:"owner,omitempty"`
	Repo         string `json:"repo,omitempty"`
	HasToken     bool   `json:"hasToken"`
	TokenPreview string `json:"tokenPreview,omitempty"`
	ConnectedAt  string `json:"connectedAt,omitempty"`
	ConnectedBy  string `json:"connectedBy,omitempty"`
	Active       bool   `json:"active"`
} //@name GitHubIntegration

// integrationsSettingsPayload groups every third-party integration's
// projection. Adding a new integration (Slack, Linear, …) lands a
// sibling field here.
type integrationsSettingsPayload struct {
	GitHub githubIntegrationPayload `json:"github"`
} //@name IntegrationsSettings

// uploadSettingsPayload is the admin-facing projection of UploadSettings.
// LastError surfaces the most recent backend-construction failure (e.g.
// GCS client init, bucket auth) so the admin can correct it from the
// same page they configured it on. Empty when the active backend is
// healthy or uploads are disabled.
type uploadSettingsPayload struct {
	Enabled   bool   `json:"enabled"`
	Backend   string `json:"backend,omitempty"`
	GCSBucket string `json:"gcsBucket,omitempty"`
	LastError string `json:"lastError,omitempty"`
} //@name UploadSettings

// settingsResponse is the admin-facing projection of the singleton settings
// document. Reused by UpdateSettingsHandler in the same package.
type settingsResponse struct {
	ProjectName  string                      `json:"projectName"`
	LogoURL      string                      `json:"logoUrl,omitempty"`
	PrimaryColor string                      `json:"primaryColor,omitempty"`
	Portal       portalSettingsPayload       `json:"portal"`
	AI           aiSettingsPayload           `json:"ai"`
	Feedback     feedbackSettingsPayload     `json:"feedback"`
	Uploads      uploadSettingsPayload       `json:"uploads"`
	Integrations integrationsSettingsPayload `json:"integrations"`
} //@name Settings

// formatTimeOmitZero renders a UTC RFC3339 timestamp, returning ""
// for the zero value so JSON's omitempty drops the field entirely.
func formatTimeOmitZero(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// maskAPIKey produces a "•••• abcd" preview of the stored API key
// suitable for display in the Console without revealing the secret.
// Empty input returns empty (the whole field is omitted on the wire).
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return strings.Repeat("•", len(key))
	}
	return "•••• " + key[len(key)-4:]
}

func newSettingsResponse(s *models.Settings, includeSecrets bool, store *storage.Holder) settingsResponse {
	key := ""
	if includeSecrets {
		key = s.Portal.JWTPrivateKey
	}
	maxVotes := s.Feedback.MaxVotesPerUser
	if maxVotes <= 0 {
		maxVotes = models.DefaultMaxVotesPerUser
	}
	maxFR := s.Feedback.MaxFeatureRequestsPerUser
	if maxFR <= 0 {
		maxFR = models.DefaultMaxFeatureRequestsPerUser
	}
	// Legacy deployments (or fresh ones where the backfill pass has
	// not yet landed) come back with a nil map. Surface defaults so the
	// Console editor renders the bundled starter templates the admin
	// can immediately accept or edit.
	templates := s.Feedback.EntryTypeTemplatesByLang
	if templates == nil {
		templates = models.DefaultEntryTypeTemplates()
	}
	return settingsResponse{
		ProjectName:  s.ProjectName,
		LogoURL:      s.LogoURL,
		PrimaryColor: s.PrimaryColor,
		Portal: portalSettingsPayload{
			CustomAuthEnabled:    s.Portal.CustomAuthEnabled,
			AuthURL:              s.Portal.AuthURL,
			CustomAuthButtonText: s.Portal.CustomAuthButtonText,
			JWTPrivateKey:        key,
			GoogleAuthEnabled:    s.Portal.GoogleAuthEnabled,
			GoogleClientID:       s.Portal.GoogleClientID,
		},
		AI: aiSettingsPayload{
			Enabled:          s.AI.Enabled,
			HasAPIKey:        s.AI.APIKey != "",
			APIKeyPreview:    maskAPIKey(s.AI.APIKey),
			Model:            s.AI.Model,
			LastAnalyzedAt:   formatTimeOmitZero(s.AI.LastAnalyzedAt),
			LastErrorAt:      formatTimeOmitZero(s.AI.LastErrorAt),
			LastErrorMessage: s.AI.LastErrorMessage,
		},
		Feedback: feedbackSettingsPayload{
			MaxVotesPerUser:           maxVotes,
			MaxFeatureRequestsPerUser: maxFR,
			EntryTypeTemplates:        templates,
			DefaultEntryTypeTemplates: models.DefaultEntryTypeTemplates(),
			SupportRequest: supportRequestPayload{
				Enabled: s.Feedback.SupportRequest.Enabled,
				URL:     s.Feedback.SupportRequest.URL,
			},
		},
		Uploads: uploadSettingsPayload{
			Enabled:   s.Uploads.Enabled,
			Backend:   string(s.Uploads.Backend),
			GCSBucket: s.Uploads.GCSBucket,
			LastError: store.LastBuildError(),
		},
		Integrations: buildIntegrationsPayload(s.Integrations),
	}
}

// buildIntegrationsPayload projects the embedded IntegrationsSettings
// into wire shape. Lives next to newSettingsResponse so the per-integration
// masking rules (token preview, "active" derivation) stay co-located
// with the rest of the settings projection.
func buildIntegrationsPayload(in models.IntegrationsSettings) integrationsSettingsPayload {
	return integrationsSettingsPayload{
		GitHub: githubIntegrationPayload{
			Disabled:     in.GitHub.Disabled,
			Owner:        in.GitHub.Owner,
			Repo:         in.GitHub.Repo,
			HasToken:     in.GitHub.Token != "",
			TokenPreview: maskAPIKey(in.GitHub.Token),
			ConnectedAt:  formatTimeOmitZero(in.GitHub.ConnectedAt),
			ConnectedBy:  in.GitHub.ConnectedBy,
			Active:       models.IsGitHubIntegrationActive(in.GitHub),
		},
	}
}

// GetSettingsHandler returns the runtime settings document.
//
//	@ID			console-get-settings
//	@Summary	Get settings
//	@Tags		Console
//	@Produce	json
//	@Success	200	{object}	settingsResponse
//	@Router		/api/console/settings [get]
func GetSettingsHandler(do dataoperations.Store, store *storage.Holder) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := do.GetSettings()
		if err != nil || s == nil {
			response.SystemError(c, err)
			return
		}
		u := middlewares.CurrentUser(c)
		response.Success(c, newSettingsResponse(s, u != nil && u.IsAdmin(), store))
	}
}
