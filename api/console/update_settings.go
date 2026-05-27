package console

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// updatePortalRequest is the nested patch payload for PortalSettings.
// JWTPrivateKey is intentionally absent — it is generated at setup and
// cannot be rotated through the API in v0.1.
type updatePortalRequest struct {
	CustomAuthEnabled    *bool   `json:"customAuthEnabled,omitempty"`
	AuthURL              *string `json:"authUrl,omitempty"`
	CustomAuthButtonText *string `json:"customAuthButtonText,omitempty"`
} //@name consoleUpdatePortalSettings

// updateSupportRequestRequest is the nested patch payload for the
// Portal "New Support Request" button. Enabled+URL share the same
// "enabled requires a usable URL" invariant as the custom-auth pair.
type updateSupportRequestRequest struct {
	Enabled *bool   `json:"enabled,omitempty"`
	URL     *string `json:"url,omitempty"`
} //@name consoleUpdateSupportRequestSettings

// updateFeedbackRequest is the nested patch payload for FeedbackSettings.
//
// EntryTypeTemplates, when present, replaces the entire templates map.
// The Console UI always submits the full set of current templates, so a
// PATCH that omits the field leaves templates untouched and a PATCH
// that includes it is authoritative for every key.
type updateFeedbackRequest struct {
	MaxVotesPerUser           *int                         `json:"maxVotesPerUser,omitempty"`
	MaxFeatureRequestsPerUser *int                         `json:"maxFeatureRequestsPerUser,omitempty"`
	EntryTypeTemplates        *map[string]string           `json:"entryTypeTemplates,omitempty"`
	SupportRequest            *updateSupportRequestRequest `json:"supportRequest,omitempty"`
} //@name consoleUpdateFeedbackSettings

// updateSettingsRequest is the body for PATCH /api/console/settings.
//
// Sparse update: only fields present on the wire are applied. Pointers are
// retained here purely so the JSON binder can distinguish "not provided"
// from "explicitly empty".
type updateSettingsRequest struct {
	ProjectName  *string                `json:"projectName,omitempty"`
	LogoURL      *string                `json:"logoUrl,omitempty"`
	PrimaryColor *string                `json:"primaryColor,omitempty"`
	Portal       *updatePortalRequest   `json:"portal,omitempty"`
	Feedback     *updateFeedbackRequest `json:"feedback,omitempty"`
} //@name consoleUpdateSettingsRequest

// UpdateSettingsHandler patches the runtime settings document. Nested
// fields under Portal are addressed via dot-paths so a single request
// touching only one portal flag does not have to round-trip the others.
//
//	@ID			console-update-settings
//	@Summary	Update settings
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		request	body		updateSettingsRequest	true	"Patch"
//	@Success	200		{object}	settingsResponse
//	@Router		/api/console/settings [patch]
func UpdateSettingsHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req updateSettingsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		s, err := do.GetSettings()
		if err != nil || s == nil {
			response.SystemError(c, err)
			return
		}
		set := map[string]any{}
		if req.ProjectName != nil {
			set["projectname"] = *req.ProjectName
		}
		if req.LogoURL != nil {
			set["logourl"] = *req.LogoURL
		}
		if req.PrimaryColor != nil {
			set["primarycolor"] = *req.PrimaryColor
		}
		if req.Portal != nil {
			// Compute the post-patch state of the two custom-auth fields so
			// we can enforce: customAuthEnabled = true requires a usable
			// AuthURL. Pulling the current values from settings means the
			// admin can flip the toggle without re-sending an unchanged URL.
			nextEnabled := s.Portal.CustomAuthEnabled
			if req.Portal.CustomAuthEnabled != nil {
				nextEnabled = *req.Portal.CustomAuthEnabled
			}
			nextAuthURL := s.Portal.AuthURL
			if req.Portal.AuthURL != nil {
				nextAuthURL = strings.TrimSpace(*req.Portal.AuthURL)
			}
			if nextEnabled {
				if nextAuthURL == "" {
					response.BadRequestWithMessage(c, "Auth URL is required when custom authentication is enabled.")
					return
				}
				if !isValidAbsoluteURL(nextAuthURL) {
					response.BadRequestWithMessage(c, "Auth URL must be a valid absolute URL.")
					return
				}
			}

			if req.Portal.CustomAuthEnabled != nil {
				set["portal.customauthenabled"] = *req.Portal.CustomAuthEnabled
			}
			if req.Portal.AuthURL != nil {
				set["portal.authurl"] = nextAuthURL
			}
			if req.Portal.CustomAuthButtonText != nil {
				set["portal.customauthbuttontext"] = strings.TrimSpace(*req.Portal.CustomAuthButtonText)
			}
		}
		if req.Feedback != nil {
			if req.Feedback.MaxVotesPerUser != nil {
				v := *req.Feedback.MaxVotesPerUser
				if v < 1 {
					response.BadRequestWithMessage(c, "Max votes per user must be at least 1.")
					return
				}
				set["feedback.maxvotesperuser"] = v
			}
			if req.Feedback.MaxFeatureRequestsPerUser != nil {
				v := *req.Feedback.MaxFeatureRequestsPerUser
				if v < 1 {
					response.BadRequestWithMessage(c, "Max feature requests per user must be at least 1.")
					return
				}
				set["feedback.maxfeaturerequestsperuser"] = v
			}
			if req.Feedback.SupportRequest != nil {
				// Mirrors the custom-auth invariant: enabling the
				// button requires a usable absolute URL. Pull current
				// values so the admin can flip the toggle without
				// re-sending an unchanged URL.
				nextEnabled := s.Feedback.SupportRequest.Enabled
				if req.Feedback.SupportRequest.Enabled != nil {
					nextEnabled = *req.Feedback.SupportRequest.Enabled
				}
				nextURL := s.Feedback.SupportRequest.URL
				if req.Feedback.SupportRequest.URL != nil {
					nextURL = strings.TrimSpace(*req.Feedback.SupportRequest.URL)
				}
				if nextEnabled {
					if nextURL == "" {
						response.BadRequestWithMessage(c, "Support request URL is required when the button is enabled.")
						return
					}
					if !isValidAbsoluteURL(nextURL) {
						response.BadRequestWithMessage(c, "Support request URL must be a valid absolute URL.")
						return
					}
				}
				if req.Feedback.SupportRequest.Enabled != nil {
					set["feedback.supportrequest.enabled"] = *req.Feedback.SupportRequest.Enabled
				}
				if req.Feedback.SupportRequest.URL != nil {
					set["feedback.supportrequest.url"] = nextURL
				}
			}
			if req.Feedback.EntryTypeTemplates != nil {
				// Validate keys are known EntryType values and normalize
				// the map (trim trailing whitespace; preserve internal
				// markdown verbatim). We do NOT reject unknown keys —
				// future entry types should be forward-compatible — but
				// we drop empty keys defensively.
				in := *req.Feedback.EntryTypeTemplates
				out := make(map[string]string, len(in))
				for k, v := range in {
					if k == "" {
						continue
					}
					out[k] = strings.TrimRight(v, " \t\r\n")
				}
				set["feedback.entrytypetemplates"] = out
			}
		}
		if err := do.UpdateSettings(set); err != nil {
			response.SystemError(c, err)
			return
		}
		updated, err := do.GetSettings()
		if err != nil || updated == nil {
			response.SystemError(c, err)
			return
		}
		response.Success(c, newSettingsResponse(updated, true))
	}
}

// isValidAbsoluteURL accepts only http(s) absolute URLs with a host. The
// custom auth URL must be reachable by the visitor's browser, so relative
// or javascript: URLs are rejected outright.
func isValidAbsoluteURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}
