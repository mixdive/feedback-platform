package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// myProfileResponse is the wire shape returned for the signed-in admin's
// own profile page. HasPasswordAccount tells the UI whether to render the
// "Change password" form — only Email-backed accounts can rotate a stored
// hash; Custom-auth-only users have no local credential.
type myProfileResponse struct {
	ID                string `json:"id"`
	Email             string `json:"email,omitempty"`
	Name              string `json:"name,omitempty"`
	Username          string `json:"username,omitempty"`
	ImageURL          string `json:"imageUrl,omitempty"`
	HasPasswordAccount bool  `json:"hasPasswordAccount"`
} //@name MyProfile

func newMyProfileResponse(u *models.User) myProfileResponse {
	out := myProfileResponse{ID: u.ID}
	if a, ok := u.EmailAccount(); ok {
		out.Email = a.Email
		out.Name = a.Name
		out.Username = a.Username
		out.ImageURL = a.ImageURL
		out.HasPasswordAccount = a.Password != ""
	}
	if out.Name == "" || out.Username == "" || out.ImageURL == "" || out.Email == "" {
		if a, ok := u.CustomAccount(); ok {
			if out.Email == "" {
				out.Email = a.Email
			}
			if out.Name == "" {
				out.Name = a.Name
			}
			if out.Username == "" {
				out.Username = a.Username
			}
			if out.ImageURL == "" {
				out.ImageURL = a.ImageURL
			}
		}
	}
	return out
}

// GetMyProfileHandler returns the signed-in user's own profile, including
// whether they can change their password. Powers Console > Settings > My
// Profile.
//
//	@ID			console-get-my-profile
//	@Summary	Get my profile
//	@Tags		Console
//	@Produce	json
//	@Success	200	{object}	myProfileResponse
//	@Failure	401	{object}	response.ApiError
//	@Router		/api/console/me [get]
func GetMyProfileHandler(_ dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := middlewares.CurrentUser(c)
		if u == nil {
			response.UnauthorizedErrorWithMessage(c, "Authentication required.")
			return
		}
		response.Success(c, newMyProfileResponse(u))
	}
}
