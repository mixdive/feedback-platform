package console

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/storage"
)

// updatePortalRequest is the nested patch payload for PortalSettings.
// JWTPrivateKey is intentionally absent — it is generated at setup and
// cannot be rotated through the API in v0.1. GoogleClientID is accepted
// (it is not a secret) but GoogleAuthEnabled=true requires it to be set.
type updatePortalRequest struct {
	CustomAuthEnabled    *bool   `json:"customAuthEnabled,omitempty"`
	AuthURL              *string `json:"authUrl,omitempty"`
	CustomAuthButtonText *string `json:"customAuthButtonText,omitempty"`
	GoogleAuthEnabled    *bool   `json:"googleAuthEnabled,omitempty"`
	GoogleClientID       *string `json:"googleClientId,omitempty"`
} //@name consoleUpdatePortalSettings

// updateSupportRequestRequest is the nested patch payload for the
// Portal "New Support Request" button. Enabled+URL share the same
// "enabled requires a usable URL" invariant as the custom-auth pair.
type updateSupportRequestRequest struct {
	Enabled *bool   `json:"enabled,omitempty"`
	URL     *string `json:"url,omitempty"`
} //@name consoleUpdateSupportRequestSettings

// updateUploadsRequest is the nested patch payload for UploadSettings.
// All fields are pointers so the binder can distinguish "not provided"
// from "explicitly empty" — the admin can flip Enabled off without
// re-sending the Backend / GCSBucket fields.
type updateUploadsRequest struct {
	Enabled   *bool   `json:"enabled,omitempty"`
	Backend   *string `json:"backend,omitempty"`
	GCSBucket *string `json:"gcsBucket,omitempty"`
} //@name consoleUpdateUploadsSettings

// updateFeedbackRequest is the nested patch payload for FeedbackSettings.
//
// EntryTypeTemplates, when present, replaces the entire templates map.
// The Console UI always submits the full set of current templates
// (across every entry type and every language), so a PATCH that
// omits the field leaves templates untouched and a PATCH that
// includes it is authoritative for every (entry-type, language)
// pair.
type updateFeedbackRequest struct {
	EntryTypeTemplates *map[string]map[string]string `json:"entryTypeTemplates,omitempty"`
	SupportRequest     *updateSupportRequestRequest  `json:"supportRequest,omitempty"`
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
	Uploads      *updateUploadsRequest  `json:"uploads,omitempty"`
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
func UpdateSettingsHandler(do dataoperations.Store, store *storage.Holder) gin.HandlerFunc {
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

			// Google auth mirrors the custom-auth invariant: enabling
			// requires a usable Client ID. Pull the current value so the
			// admin can flip the toggle without re-sending an unchanged ID.
			nextGoogleEnabled := s.Portal.GoogleAuthEnabled
			if req.Portal.GoogleAuthEnabled != nil {
				nextGoogleEnabled = *req.Portal.GoogleAuthEnabled
			}
			nextGoogleClientID := s.Portal.GoogleClientID
			if req.Portal.GoogleClientID != nil {
				nextGoogleClientID = strings.TrimSpace(*req.Portal.GoogleClientID)
			}
			if nextGoogleEnabled && nextGoogleClientID == "" {
				response.BadRequestWithMessage(c, "Google Client ID is required when Google authentication is enabled.")
				return
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
			if req.Portal.GoogleAuthEnabled != nil {
				set["portal.googleauthenabled"] = *req.Portal.GoogleAuthEnabled
			}
			if req.Portal.GoogleClientID != nil {
				set["portal.googleclientid"] = nextGoogleClientID
			}
		}
		if req.Feedback != nil {
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
				// Validate keys and normalize values. Outer keys are
				// EntryType values, inner keys are language codes; we
				// do NOT reject unknown values on either axis —
				// forward-compatibility for future entry types and
				// languages means an admin's existing data is never
				// dropped on an upgrade. Empty keys are dropped
				// defensively; trailing whitespace is trimmed off
				// every leaf to keep the document tidy.
				in := *req.Feedback.EntryTypeTemplates
				out := make(map[string]map[string]string, len(in))
				for entryType, byLang := range in {
					if entryType == "" {
						continue
					}
					inner := make(map[string]string, len(byLang))
					for lang, body := range byLang {
						if lang == "" {
							continue
						}
						inner[lang] = strings.TrimRight(body, " \t\r\n")
					}
					out[entryType] = inner
				}
				set["feedback.entrytypetemplatesbylang"] = out
			}
		}
		if req.Uploads != nil {
			// Compute the post-patch state of the three upload fields so
			// validation runs against the merged values (admin can flip a
			// single field without re-sending the rest).
			nextEnabled := s.Uploads.Enabled
			if req.Uploads.Enabled != nil {
				nextEnabled = *req.Uploads.Enabled
			}
			nextBackend := s.Uploads.Backend
			if req.Uploads.Backend != nil {
				nextBackend = models.UploadBackend(strings.TrimSpace(*req.Uploads.Backend))
			}
			nextBucket := s.Uploads.GCSBucket
			if req.Uploads.GCSBucket != nil {
				nextBucket = strings.TrimSpace(*req.Uploads.GCSBucket)
			}
			if nextEnabled {
				switch nextBackend {
				case models.UploadBackendLocal:
				case models.UploadBackendGCS:
					if nextBucket == "" {
						response.BadRequestWithMessage(c, "GCS bucket is required when backend is gcs.")
						return
					}
				case "":
					response.BadRequestWithMessage(c, "Upload backend is required when uploads are enabled.")
					return
				default:
					response.BadRequestWithMessage(c, "Upload backend must be either \"local\" or \"gcs\".")
					return
				}
			}
			if req.Uploads.Enabled != nil {
				set["uploads.enabled"] = *req.Uploads.Enabled
			}
			if req.Uploads.Backend != nil {
				set["uploads.backend"] = string(nextBackend)
			}
			if req.Uploads.GCSBucket != nil {
				set["uploads.gcsbucket"] = nextBucket
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
		// Hot-swap the active storage backend if uploads-related fields
		// were touched. Build errors (e.g. GCS auth) are not fatal — the
		// holder retains the previous backend and surfaces the message
		// via LastBuildError on the response.
		if req.Uploads != nil {
			_ = store.Reload(updated.Uploads)
		}
		response.Success(c, newSettingsResponse(updated, true, store))
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
