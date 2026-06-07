package api

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// voteQuotaPayload reports the caller's per-user vote quota state.
// Unlimited=true on Console-access users (admins + editors) — they
// bypass the quota entirely. Used is meaningful only when Unlimited is
// false.
type voteQuotaPayload struct {
	Used      int  `json:"used"`
	Max       int  `json:"max"`
	Unlimited bool `json:"unlimited"`
} //@name VoteQuota

// featureRequestQuotaPayload reports the caller's per-user
// feature-request quota. Same shape as voteQuotaPayload — Used counts
// the caller's currently-open authored feature requests, Max is the
// admin-configured cap, and Unlimited=true exempts Console-access
// users entirely.
type featureRequestQuotaPayload struct {
	Used      int  `json:"used"`
	Max       int  `json:"max"`
	Unlimited bool `json:"unlimited"`
} //@name FeatureRequestQuota

// userPayload is the wire shape returned by /api/me, /api/login, and the
// portal custom-auth callback. The fields are populated from whichever
// account the caller authenticated through, with Email taking precedence
// when a user has both (the bootstrap admin who later signs into the
// portal too).
type userPayload struct {
	ID                  string                     `json:"id"`
	Email               string                     `json:"email,omitempty"`
	Name                string                     `json:"name,omitempty"`
	Username            string                     `json:"username,omitempty"`
	ImageURL            string                     `json:"imageUrl,omitempty"`
	Roles               []string                   `json:"roles"`
	VoteQuota           voteQuotaPayload           `json:"voteQuota"`
	FeatureRequestQuota featureRequestQuotaPayload `json:"featureRequestQuota"`
} //@name UserInfo

// NewUserPayloadWithQuota produces the wire payload AND computes the
// caller's vote-quota state in one call. Exposed so the portal
// custom-login handler can build the same payload without duplicating
// the quota lookup. Falls back to the default max when settings are
// missing — never blocks login on a settings read.
func NewUserPayloadWithQuota(do dataoperations.Store, u *models.User) userPayload {
	out := newUserPayload(u)
	out.VoteQuota = computeVoteQuota(do, u)
	out.FeatureRequestQuota = computeFeatureRequestQuota(do, u)
	return out
}

func computeVoteQuota(do dataoperations.Store, u *models.User) voteQuotaPayload {
	if u.HasConsoleAccess() {
		return voteQuotaPayload{Unlimited: true}
	}
	max, err := do.MaxVotesPerUser()
	if err != nil || max <= 0 {
		max = models.DefaultMaxVotesPerUser
	}
	return voteQuotaPayload{Used: u.VotesSpent, Max: max}
}

func computeFeatureRequestQuota(do dataoperations.Store, u *models.User) featureRequestQuotaPayload {
	if u.HasConsoleAccess() {
		return featureRequestQuotaPayload{Unlimited: true}
	}
	max, err := do.MaxFeatureRequestsPerUser()
	if err != nil || max <= 0 {
		max = models.DefaultMaxFeatureRequestsPerUser
	}
	return featureRequestQuotaPayload{Used: u.FeatureRequestsOpen, Max: max}
}

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
		response.Success(c, meResponse{User: NewUserPayloadWithQuota(do, u)})
	}
}
