package console

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// userResponse is the row shape for the Console "Users" page. Covers
// both administrative users (any role) and normal users (no role —
// e.g. portal visitors signed in via the custom auth provider).
//
// Email/Name/Username/ImageURL are pulled from the user's most
// authoritative attached account: the Email account when present
// (admins from setup), otherwise the Custom account (portal
// visitors). Sensitive fields (PasswordHash, raw account list) never
// leave the server. Accounts is the set of attached account-type keys
// so the Console can tell at a glance which providers a user uses.
//
// EntryCount/FeatureRequestCount/BugCount/SupportCount/OtherCount are
// the per-user aggregate columns the Users page renders. OtherCount is
// the catch-all (everything not feature-request, bug, or support —
// including the untyped value), so Total = FR + Bug + Support + Other
// always holds.
type userResponse struct {
	ID                  string   `json:"id"`
	Email               string   `json:"email"`
	Name                string   `json:"name,omitempty"`
	Username            string   `json:"username,omitempty"`
	ImageURL            string   `json:"imageUrl,omitempty"`
	Roles               []string `json:"roles"`
	Accounts            []string `json:"accounts"`
	IsBlocked           bool     `json:"isBlocked"`
	CreatedAt           string   `json:"createdAt"`
	EntryCount          int      `json:"entryCount"`
	FeatureRequestCount int      `json:"featureRequestCount"`
	BugCount            int      `json:"bugCount"`
	SupportCount        int      `json:"supportCount"`
	OtherCount          int      `json:"otherCount"`
} //@name User

type userListResponse struct {
	Data []userResponse `json:"data"`
	Meta listMeta       `json:"meta"`
} //@name UserList

func userToResponse(u models.User) userResponse {
	primary := pickPrimaryAccount(u)
	roles := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, string(r))
	}
	accounts := make([]string, 0, len(u.Accounts))
	// Iterate in a fixed order so identical users produce identical
	// payloads regardless of map iteration order.
	for _, t := range []models.UserAccountType{
		models.UserAccountTypeEmail,
		models.UserAccountTypeCustom,
		models.UserAccountTypeGoogle,
	} {
		if _, ok := u.Accounts[t]; ok {
			accounts = append(accounts, string(t))
		}
	}
	return userResponse{
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

// pickPrimaryAccount selects the account whose display fields back the
// row. Email is preferred (covers admins from the setup wizard);
// Custom comes next (covers portal visitors); Google last. Returns the
// zero account when the user has no attached accounts at all.
func pickPrimaryAccount(u models.User) models.UserAccount {
	for _, t := range []models.UserAccountType{
		models.UserAccountTypeEmail,
		models.UserAccountTypeCustom,
		models.UserAccountTypeGoogle,
	} {
		if a, ok := u.Accounts[t]; ok {
			return a
		}
	}
	return models.UserAccount{}
}

// listUsersQuery is the bound query string for GET /api/console/user.
//
// Sort options:
//
//   - new/old/email — legacy modes (createdat desc/asc, email asc).
//   - alpha — name → username → email fallback, case-insensitive.
//   - created — createdat; direction picks newest- or oldest-first.
//   - entries/feature-requests/bugs/others — count columns; default
//     direction is desc so the busiest users sort to the top.
//
// Direction is asc|desc; when omitted, each sort key picks the
// idiomatic default (alpha=asc, every count + created=desc).
type listUsersQuery struct {
	Search    string `form:"search"                                                                                       example:"alice"`
	Role      string `form:"role"      enums:"all,admin,editor,none"                                                      example:"admin"`
	Status    string `form:"status"    enums:"all,active,blocked"                                                         example:"active"`
	Sort      string `form:"sort"      enums:"new,old,email,alpha,created,entries,feature-requests,bugs,support,others"           example:"alpha"`
	Direction string `form:"direction" enums:"asc,desc"                                                                   example:"asc"`
	Page      int    `form:"page"                                                                                         example:"1"`
	Limit     int    `form:"limit"                                                                                        example:"25"`
}

// ListUsersHandler returns a paginated list of every user in the
// system — administrative and normal alike — with search, role,
// status and sort controls. Powers the Console "Users" page.
//
// The handler fetches every matching user (no Mongo pagination),
// merges in per-user entry counts via a single aggregation, then
// sorts and slices in Go. Pilot volume is small enough that loading
// the full set is cheaper than emitting a heavier $lookup pipeline,
// and it makes count-based sort options trivially correct.
//
//	@ID			console-list-users
//	@Summary	List users
//	@Tags		Console
//	@Produce	json
//	@Param		request	query		listUsersQuery	false	"Filters"
//	@Success	200		{object}	userListResponse
//	@Failure	500		{object}	response.ApiError
//	@Router		/api/console/user [get]
func ListUsersHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request listUsersQuery
		if err := c.ShouldBindQuery(&request); err != nil {
			response.ValidationError(c, err)
			return
		}
		filter := dataoperations.UserListFilter{
			Search: request.Search,
			Role:   request.Role,
			Status: request.Status,
		}
		page := request.Page
		if page <= 0 {
			page = 1
		}
		limit := request.Limit
		if limit <= 0 || limit > 100 {
			limit = 25
		}

		users, err := do.ListAllUsers(filter)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		counts, err := do.CountEntriesByUserAndEntryType()
		if err != nil {
			response.SystemError(c, err)
			return
		}

		rows := make([]userResponse, 0, len(users))
		for _, u := range users {
			row := userToResponse(u)
			c := counts[u.ID]
			row.EntryCount = c.Total
			row.FeatureRequestCount = c.FeatureRequest
			row.BugCount = c.Bug
			row.SupportCount = c.Support
			row.OtherCount = c.Other
			rows = append(rows, row)
		}
		sortUserRows(rows, request.Sort, request.Direction)

		total := int64(len(rows))
		start := (page - 1) * limit
		if start < 0 {
			start = 0
		}
		if start > len(rows) {
			start = len(rows)
		}
		end := start + limit
		if end > len(rows) {
			end = len(rows)
		}
		pageRows := rows[start:end]

		response.Success(c, userListResponse{
			Data: pageRows,
			Meta: listMeta{
				Total:   total,
				Page:    page,
				Limit:   limit,
				HasMore: int64(end) < total,
			},
		})
	}
}

// sortUserRows orders rows in place by sortKey + direction. Direction
// "" defers to each key's idiomatic default (alpha=asc, count or
// created=desc, legacy keys carry their own direction).
func sortUserRows(rows []userResponse, sortKey, direction string) {
	var less func(a, b userResponse) bool
	defaultDesc := false
	switch sortKey {
	case "alpha":
		less = func(a, b userResponse) bool { return userAlphaKey(a) < userAlphaKey(b) }
	case "old":
		less = func(a, b userResponse) bool { return a.CreatedAt < b.CreatedAt }
	case "email":
		less = func(a, b userResponse) bool { return strings.ToLower(a.Email) < strings.ToLower(b.Email) }
	case "created":
		less = func(a, b userResponse) bool { return a.CreatedAt < b.CreatedAt }
		defaultDesc = true
	case "entries":
		less = func(a, b userResponse) bool { return a.EntryCount < b.EntryCount }
		defaultDesc = true
	case "feature-requests":
		less = func(a, b userResponse) bool { return a.FeatureRequestCount < b.FeatureRequestCount }
		defaultDesc = true
	case "bugs":
		less = func(a, b userResponse) bool { return a.BugCount < b.BugCount }
		defaultDesc = true
	case "support":
		less = func(a, b userResponse) bool { return a.SupportCount < b.SupportCount }
		defaultDesc = true
	case "others":
		less = func(a, b userResponse) bool { return a.OtherCount < b.OtherCount }
		defaultDesc = true
	default: // "" or "new"
		less = func(a, b userResponse) bool { return a.CreatedAt < b.CreatedAt }
		defaultDesc = true
	}
	desc := defaultDesc
	switch direction {
	case "asc":
		desc = false
	case "desc":
		desc = true
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if desc {
			return less(rows[j], rows[i])
		}
		return less(rows[i], rows[j])
	})
}

// userAlphaKey is the case-insensitive label used by the "alpha" sort:
// name → username → email fallback. Mirrors the entry-author dropdown
// ordering so a user appears in the same spot across surfaces.
func userAlphaKey(u userResponse) string {
	switch {
	case u.Name != "":
		return strings.ToLower(u.Name)
	case u.Username != "":
		return strings.ToLower(u.Username)
	default:
		return strings.ToLower(u.Email)
	}
}
