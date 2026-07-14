package portal

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// loginCustomRequest carries the JWT minted by the admin-configured external
// auth provider. The portal frontend forwards whatever it received in the
// `?token=` callback query parameter.
type loginCustomRequest struct {
	Token string `json:"token" binding:"required,min=1"`
} //@name portalLoginCustomRequest

// userPayload mirrors the shape served by /api/me — kept in lockstep with
// api/me.go so the portal can reuse the same response handler in Redux.
type userPayload struct {
	ID       string   `json:"id"`
	Email    string   `json:"email,omitempty"`
	Name     string   `json:"name,omitempty"`
	Username string   `json:"username,omitempty"`
	ImageURL string   `json:"imageUrl,omitempty"`
	Roles    []string `json:"roles"`
} //@name PortalUserInfo

// loginCustomResponse confirms a successful login and echoes user info so
// the frontend can render the avatar/name without a follow-up /me call.
type loginCustomResponse struct {
	User userPayload `json:"user"`
} //@name portalLoginCustomResponse

// LoginCustomHandler verifies the inbound JWT against the portal-side
// shared secret (PortalSettings.JWTPrivateKey, HS256), provisions or
// updates a Custom-account user, and issues a session cookie.
//
// The JWT may carry: `sub`/`userId` (required, treated as the external
// user ID), `username`, and `photoUrl`.
//
//	@ID			portal-login-custom
//	@Summary	Portal custom login
//	@Description	Exchanges a JWT minted by the admin's external auth provider for a Mixdive session cookie.
//	@Tags		Portal
//	@Accept		json
//	@Produce	json
//	@Param		request	body		loginCustomRequest	true	"JWT from the external provider"
//	@Success	200		{object}	loginCustomResponse
//	@Failure	400		{object}	response.ApiError
//	@Failure	401		{object}	response.ApiError
//	@Failure	503		{object}	response.ApiError
//	@Router		/api/portal/auth/custom [post]
func LoginCustomHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := do.GetSettings()
		if err != nil || s == nil {
			response.SystemError(c, err)
			return
		}
		if !s.Portal.CustomAuthEnabled || s.Portal.AuthURL == "" || s.Portal.JWTPrivateKey == "" {
			response.ErrorWithStatusCodeAndMessage(c, 503, "Custom authentication is not enabled.")
			return
		}
		var req loginCustomRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}

		claims, err := verifyCustomAuthToken(req.Token, s.Portal.JWTPrivateKey)
		if err != nil {
			response.UnauthorizedErrorWithMessage(c, "Invalid authentication token.")
			return
		}

		userID := claimString(claims, "userId", "sub")
		if userID == "" {
			response.UnauthorizedErrorWithMessage(c, "Token missing user identifier.")
			return
		}
		username := claimString(claims, "username", "name")
		photoURL := claimString(claims, "photoUrl", "picture")
		email := claimString(claims, "email")

		user, err := do.FindUserByKey(userID)
		if err != nil {
			response.SystemError(c, err)
			return
		}

		// Account linking: when this external userId isn't known yet but the
		// JWT carries an email that already belongs to an existing user
		// (e.g. the bootstrap admin, or a prior Google sign-in), attach the
		// Custom account to that user instead of creating a duplicate. The
		// email is trusted because the token was signed by the admin's own
		// auth system — the same provider that owns the userId.
		if user == nil && email != "" {
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
			user.SetAccount(models.UserAccountTypeCustom, models.UserAccount{
				ID:              userID,
				Name:            username,
				Username:        username,
				ImageURL:        photoURL,
				Email:           email,
				IsEmailVerified: email != "",
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
			existing, _ := user.CustomAccount()
			if existing.CreatedAt.IsZero() {
				existing.CreatedAt = now
			}
			existing.ID = userID
			if username != "" {
				existing.Name = username
				existing.Username = username
			}
			if photoURL != "" {
				existing.ImageURL = photoURL
			}
			if email != "" {
				existing.Email = email
				existing.IsEmailVerified = true
			}
			user.SetAccount(models.UserAccountTypeCustom, existing)
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
		response.Success(c, loginCustomResponse{User: newPortalUserPayload(user)})
	}
}

func newPortalUserPayload(u *models.User) userPayload {
	roles := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, string(r))
	}
	out := userPayload{ID: u.ID, Roles: roles}
	if a, ok := u.EmailAccount(); ok {
		out.Email = a.Email
		out.Name = pickNonEmpty(out.Name, a.Name)
		out.Username = pickNonEmpty(out.Username, a.Username)
		out.ImageURL = pickNonEmpty(out.ImageURL, a.ImageURL)
	}
	if a, ok := u.CustomAccount(); ok {
		out.Email = pickNonEmpty(out.Email, a.Email)
		out.Name = pickNonEmpty(out.Name, a.Name)
		out.Username = pickNonEmpty(out.Username, a.Username)
		out.ImageURL = pickNonEmpty(out.ImageURL, a.ImageURL)
	}
	if a, ok := u.GoogleAccount(); ok {
		out.Email = pickNonEmpty(out.Email, a.Email)
		out.Name = pickNonEmpty(out.Name, a.Name)
		out.Username = pickNonEmpty(out.Username, a.Username)
		out.ImageURL = pickNonEmpty(out.ImageURL, a.ImageURL)
	}
	return out
}

func pickNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// verifyCustomAuthToken parses and validates the inbound HS256 JWT against
// the portal's shared secret. The secret on disk is base64-RawURLEncoded
// (see models.NewPortalSettings) — we accept both the encoded and the raw
// forms so admins copy/paste-mistakes don't lock the portal out.
func verifyCustomAuthToken(token, secret string) (jwt.MapClaims, error) {
	keys := candidateKeys(secret)
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}))
	var lastErr error
	for _, key := range keys {
		var claims jwt.MapClaims
		_, err := parser.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
			return key, nil
		})
		if err == nil {
			return claims, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no candidate key accepted token")
	}
	return nil, lastErr
}

func candidateKeys(secret string) [][]byte {
	out := [][]byte{[]byte(secret)}
	// Try base64 raw-url decoding too — that's the format we generate at
	// setup, and it's what most signers will use after copy/pasting from
	// the Console.
	for _, dec := range []func(string) ([]byte, error){
		base64RawURLDecode,
		base64StdDecode,
	} {
		if b, err := dec(secret); err == nil && len(b) > 0 {
			out = append(out, b)
		}
	}
	return out
}

func claimString(c jwt.MapClaims, keys ...string) string {
	for _, k := range keys {
		v, ok := c[k]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s != "" {
			return s
		}
	}
	return ""
}
