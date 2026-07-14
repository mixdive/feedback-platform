package api

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// userPayload is the wire shape returned by /api/me, /api/login, and the
// portal custom-auth callback. The fields are populated from whichever
// account the caller authenticated through, with Email taking precedence
// when a user has both (the bootstrap admin who later signs into the
// portal too).
type userPayload struct {
	ID       string   `json:"id"`
	Email    string   `json:"email,omitempty"`
	Name     string   `json:"name,omitempty"`
	Username string   `json:"username,omitempty"`
	ImageURL string   `json:"imageUrl,omitempty"`
	Roles    []string `json:"roles"`
} //@name UserInfo

func newUserPayload(u *models.User) userPayload {
	roles := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, string(r))
	}
	out := userPayload{ID: u.ID, Roles: roles}
	if a, ok := u.EmailAccount(); ok {
		out.Email = a.Email
		out.Name = firstNonEmpty(out.Name, a.Name)
		out.Username = firstNonEmpty(out.Username, a.Username)
		out.ImageURL = firstNonEmpty(out.ImageURL, a.ImageURL)
	}
	if a, ok := u.CustomAccount(); ok {
		out.Email = firstNonEmpty(out.Email, a.Email)
		out.Name = firstNonEmpty(out.Name, a.Name)
		out.Username = firstNonEmpty(out.Username, a.Username)
		out.ImageURL = firstNonEmpty(out.ImageURL, a.ImageURL)
	}
	if a, ok := u.GoogleAccount(); ok {
		out.Email = firstNonEmpty(out.Email, a.Email)
		out.Name = firstNonEmpty(out.Name, a.Name)
		out.Username = firstNonEmpty(out.Username, a.Username)
		out.ImageURL = firstNonEmpty(out.ImageURL, a.ImageURL)
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// meResponse is what /api/me returns to authenticated callers.
type meResponse struct {
	User userPayload `json:"user"`
} //@name MeResult

// MeHandler reports the currently authenticated user. Returns 401 when no
// session is attached — the Console uses the 401 to redirect to /login;
// the Portal uses it to fall back to the anonymous (or login-button) view.
//
//	@ID			auth-me
//	@Summary	Current user
//	@Description	Returns information about the user behind the session cookie.
//	@Tags		Auth
//	@Produce	json
//	@Success	200	{object}	meResponse
//	@Failure	401	{object}	response.ApiError
//	@Router		/api/me [get]
func MeHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := middlewares.CurrentUser(c)
		if u == nil {
			response.UnauthorizedErrorWithMessage(c, "Authentication required.")
			return
		}
		response.Success(c, meResponse{User: newUserPayload(u)})
	}
}
