package portal

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// loginGoogleRequest carries the ID token minted by Google Identity Services
// in the visitor's browser. The Portal frontend forwards the `credential`
// field it receives in the GIS sign-in callback verbatim.
type loginGoogleRequest struct {
	IDToken string `json:"idToken" binding:"required,min=1"`
} //@name portalLoginGoogleRequest

// loginGoogleResponse confirms a successful login and echoes user info so the
// frontend can render the avatar/name without a follow-up /me call. Same
// shape as the custom-auth response — the Portal reuses one Redux handler.
type loginGoogleResponse struct {
	User userPayload `json:"user"`
} //@name portalLoginGoogleResponse

// LoginGoogleHandler verifies a Google Identity Services ID token against
// Google's public keys (audience = the admin-configured Client ID),
// provisions or updates a Google-account user keyed by the token's `sub`,
// and issues a session cookie.
//
// No client secret is involved: the GIS ID-token flow obtains a signed JWT
// in the browser and the server only verifies it. GoogleClientID is the
// expected audience and is not a secret.
//
//	@ID			portal-login-google
//	@Summary	Portal Google login
//	@Description	Exchanges a Google Identity Services ID token for a Mixdive session cookie.
//	@Tags		Portal
//	@Accept		json
//	@Produce	json
//	@Param		request	body		loginGoogleRequest	true	"Google ID token"
//	@Success	200		{object}	loginGoogleResponse
//	@Failure	400		{object}	response.ApiError
//	@Failure	401		{object}	response.ApiError
//	@Failure	503		{object}	response.ApiError
//	@Router		/api/portal/auth/google [post]
func LoginGoogleHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := do.GetSettings()
		if err != nil || s == nil {
			response.SystemError(c, err)
			return
		}
		if !s.Portal.GoogleAuthEnabled || s.Portal.GoogleClientID == "" {
			response.ErrorWithStatusCodeAndMessage(c, 503, "Google authentication is not enabled.")
			return
		}
		var req loginGoogleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}

		// idtoken.Validate checks the HS256/RS256 signature against Google's
		// published certs (cached package-side) AND that the audience equals
		// the configured Client ID. A nil error means the token is genuine
		// and intended for this deployment.
		payload, err := idtoken.Validate(c.Request.Context(), req.IDToken, s.Portal.GoogleClientID)
		if err != nil {
			response.UnauthorizedErrorWithMessage(c, "Invalid Google sign-in token.")
			return
		}

		googleID := strings.TrimSpace(payload.Subject)
		if googleID == "" {
			response.UnauthorizedErrorWithMessage(c, "Google token missing subject.")
			return
		}
		email := payloadClaimString(payload, "email")
		emailVerified := payloadClaimBool(payload, "email_verified")
		name := payloadClaimString(payload, "name")
		photoURL := payloadClaimString(payload, "picture")

		user, err := do.FindUserByKey(googleID)
		if err != nil {
			response.SystemError(c, err)
			return
		}

		// Account linking: when no user owns this Google identity yet but the
		// token carries a VERIFIED email that already belongs to an existing
		// user (e.g. the bootstrap admin's email account), attach the Google
		// account to that user instead of creating a duplicate. Gated on
		// email_verified so an unverified address can never be used to take
		// over an existing account.
		if user == nil && email != "" && emailVerified {
			byEmail, err := do.FindUserByKey(email)
			if err != nil {
				response.SystemError(c, err)
				return
			}
			user = byEmail
		}

		now := time.Now().UTC()
		if user == nil {
			user = models.NewUser()
			user.SetAccount(models.UserAccountTypeGoogle, models.UserAccount{
				ID:              googleID,
				Name:            name,
				Email:           email,
				ImageURL:        photoURL,
				IsEmailVerified: emailVerified,
				CreatedAt:       now,
			})
			if err := do.InsertUser(user); err != nil {
				response.SystemError(c, err)
				return
			}
		} else {
			if user.IsBlocked || user.IsDeleted {
				response.ForbiddenErrorWithMessage(c, "Account is not allowed to sign in.")
				return
			}
			existing, _ := user.GoogleAccount()
			if existing.CreatedAt.IsZero() {
				existing.CreatedAt = now
			}
			existing.ID = googleID
			if email != "" {
				existing.Email = email
				existing.IsEmailVerified = emailVerified
			}
			if name != "" {
				existing.Name = name
			}
			if photoURL != "" {
				existing.ImageURL = photoURL
			}
			user.SetAccount(models.UserAccountTypeGoogle, existing)
			if err := do.UpdateUser(user); err != nil {
				response.SystemError(c, err)
				return
			}
		}

		sess, err := do.CreateSession(user.ID)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		middlewares.SetSessionCookie(c, sess.Token)
		response.Success(c, loginGoogleResponse{User: newPortalUserPayload(user)})
	}
}

// payloadClaimString reads a trimmed string claim from a verified Google ID
// token payload, returning "" when the claim is absent or not a string.
func payloadClaimString(p *idtoken.Payload, key string) string {
	v, ok := p.Claims[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

// payloadClaimBool reads a boolean claim (email_verified) from a verified
// Google ID token payload, returning false when absent or non-boolean.
func payloadClaimBool(p *idtoken.Payload, key string) bool {
	v, ok := p.Claims[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}
