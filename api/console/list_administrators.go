package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// administratorResponse is the row shape returned by the Administrators
// settings page. Display fields (Email, Name, Username, ImageURL) come from
// the user's primary account — Email account when present, otherwise the
// Custom account. Sensitive fields (PasswordHash, Keys) never leave the
// server.
type administratorResponse struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name,omitempty"`
	Username  string   `json:"username,omitempty"`
	ImageURL  string   `json:"imageUrl,omitempty"`
	Roles     []string `json:"roles"`
	Accounts  []string `json:"accounts"`
	IsBlocked bool     `json:"isBlocked"`
	CreatedAt string   `json:"createdAt"`
} //@name Administrator

type administratorListResponse struct {
	Data []administratorResponse `json:"data"`
} //@name AdministratorList

func userToAdministratorResponse(u models.User) administratorResponse {
	primary := pickPrimaryAccount(u)
	roles := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, string(r))
	}
	// Iterate the known account types in fixed order so the wire payload is
	// deterministic regardless of map iteration order.
	accounts := make([]string, 0, len(u.Accounts))
	for _, t := range []models.UserAccountType{
		models.UserAccountTypeEmail,
		models.UserAccountTypeCustom,
		models.UserAccountTypeGoogle,
	} {
		if _, ok := u.Accounts[t]; ok {
			accounts = append(accounts, string(t))
		}
	}
	return administratorResponse{
		ID:        u.ID,
		Email:     primary.Email,
		Name:      primary.Name,
		Username:  primary.Username,
		ImageURL:  primary.ImageURL,
		Roles:     roles,
		Accounts:  accounts,
		IsBlocked: u.IsBlocked,
		CreatedAt: iso(u.CreatedAt),
	}
}

// ListAdministratorsHandler returns every user that carries at least one
// role. Powers the Console > Settings > Administrators page.
//
//	@ID			console-list-administrators
//	@Summary	List administrators
//	@Tags		Console
//	@Produce	json
//	@Success	200	{object}	administratorListResponse
//	@Failure	500	{object}	response.ApiError
//	@Router		/api/console/administrator [get]
func ListAdministratorsHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := do.ListUsersWithRoles()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		out := make([]administratorResponse, 0, len(users))
		for _, u := range users {
			out = append(out, userToAdministratorResponse(u))
		}
		response.Success(c, administratorListResponse{Data: out})
	}
}
