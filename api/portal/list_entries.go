// Package portal contains the public-facing HTTP handlers exposed under
// /api/portal/*: list and read entries, submit a new entry, and vote on
// an entry. Read access is gated by RequirePortalReadAccessMiddleware
// applied at the router level. Setup completion is enforced by
// RequireSetupCompletedMiddleware.
package portal

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// entryResponse is the portal-facing entry projection. Console keeps
// its own copy because the shapes are expected to diverge soon (admin
// fields, status changes, etc.).
//
// Creator is omitted for anonymously submitted records (UserID == "") and
// for records whose author no longer exists. IsVoted reports whether the
// caller (when authenticated) has voted on this entry — always false
// for anonymous viewers, since per-user voting needs identity.
type entryResponse struct {
	ID           string                  `json:"id"`
	Title        string                  `json:"title"`
	Description  string                  `json:"description,omitempty"`
	VoteCount    int                     `json:"voteCount"`
	IsVoted      bool                    `json:"isVoted"`
	CommentCount int                     `json:"commentCount"`
	Creator      *api.EntryCreator       `json:"creator,omitempty"`
	EntryType    *api.EntryTypeResponse  `json:"entryType,omitempty"`
	Status       api.EntryStatusResponse `json:"status"`
	Release      *api.ReleaseResponse    `json:"release,omitempty"`
	CreatedAt    string                  `json:"createdAt"`
	UpdatedAt    string                  `json:"updatedAt"`
} //@name Entry

// buildEntryTypePtr returns a pointer to the wire shape when
// e.EntryType resolves; otherwise nil so the field is omitted.
func buildEntryTypePtr(e models.Entry) *api.EntryTypeResponse {
	et, ok := api.BuildEntryType(e.EntryType)
	if !ok {
		return nil
	}
	return &et
}

// listMeta is the pagination block returned with every list endpoint.
type listMeta struct {
	Total   int64 `json:"total"`
	Page    int   `json:"page"`
	Limit   int   `json:"limit"`
	HasMore bool  `json:"hasMore"`
} //@name ListMeta

// entryListResponse is the envelope for paginated entry lists.
type entryListResponse struct {
	Data []entryResponse `json:"data"`
	Meta listMeta        `json:"meta"`
} //@name EntryList

// iso renders a UTC ISO-8601 string. Returns "" for the zero time so it
// can be omitted from response payloads with `omitempty`.
func iso(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

func entryToResponse(
	e models.Entry,
	creators map[string]api.EntryCreator,
	releases map[string]api.ReleaseResponse,
	isVoted bool,
) entryResponse {
	r := entryResponse{
		ID:           e.ID,
		Title:        e.Title,
		Description:  e.Description,
		VoteCount:    e.VoteCount,
		IsVoted:      isVoted,
		CommentCount: e.CommentCount,
		Status:       api.BuildEntryStatus(e.Status),
		EntryType: buildEntryTypePtr(e),
		CreatedAt:    iso(e.CreatedAt),
		UpdatedAt:    iso(e.UpdatedAt),
	}
	if e.UserID != "" {
		if c, ok := creators[e.UserID]; ok {
			r.Creator = &c
		}
	}
	if e.ReleaseID != "" {
		if rel, ok := releases[e.ReleaseID]; ok {
			r.Release = &rel
		}
	}
	return r
}

// listEntriesQuery is the bound query string for GET /api/portal/entry.
// One struct, one @Param line, one BindQuery call — every optional filter
// the handler accepts lives here.
//
// Mine flips the list to "entries authored by me" and requires an
// authenticated visitor (401 otherwise). Either way, internal records
// are stripped — the Portal never surfaces them, regardless of viewer
// or authorship; admins see internal records only in the Console.
//
// EntryType narrows to a single feedback type (feature-request, bug,
// support, other). Empty means "no restriction". OpenOnly hides closed
// (completed/cancelled) entries — the Portal "All/Feature Requests/
// Bugs" tabs set this; "Mine" leaves it false so authors see their own
// closed records.
type listEntriesQuery struct {
	Search    string `form:"search"                                                  example:"dark mode"`
	Sort      string `form:"sort"      enums:"top,new"                                example:"top"`
	Page      int    `form:"page"                                                    example:"1"`
	Limit     int    `form:"limit"                                                   example:"25"`
	Mine      bool   `form:"mine"                                                    example:"false"`
	EntryType string `form:"entryType" enums:"feature-request,bug,support,other"      example:"feature-request"`
	OpenOnly  bool   `form:"openOnly"                                                example:"true"`
}

// ListEntriesHandler returns a paginated list of entries.
//
//	@ID			portal-list-entries
//	@Summary	List entries
//	@Tags		Portal
//	@Produce	json
//	@Param		request	query		listEntriesQuery	false	"Filters"
//	@Success	200		{object}	entryListResponse
//	@Failure	401		{object}	response.ApiError
//	@Router		/api/portal/entry [get]
func ListEntriesHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request listEntriesQuery
		if err := c.BindQuery(&request); err != nil {
			response.ValidationError(c, err)
			return
		}
		viewer := middlewares.CurrentUser(c)
		hidden := false
		filter := dataoperations.EntryListFilter{
			Search:     request.Search,
			Sort:       request.Sort,
			Page:       request.Page,
			Limit:      request.Limit,
			IsInternal: &hidden,
			OpenOnly:   request.OpenOnly,
		}
		if request.EntryType != "" {
			ft, err := resolveEntryType(request.EntryType)
			if err != nil {
				response.BadRequestWithMessage(c, err.Error())
				return
			}
			filter.EntryType = ft
		}
		if request.Mine {
			if viewer == nil {
				response.UnauthorizedErrorWithMessage(c, "Authentication required.")
				return
			}
			filter.OwnerID = viewer.ID
		}
		if filter.Page <= 0 {
			filter.Page = 1
		}
		if filter.Limit <= 0 || filter.Limit > 100 {
			filter.Limit = 25
		}
		records, err := do.ListEntries(filter)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		total, err := do.CountEntries(filter)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		ids := make([]string, 0, len(records))
		entryIDs := make([]string, 0, len(records))
		releaseIDs := make([]string, 0, len(records))
		for _, e := range records {
			ids = append(ids, e.UserID)
			entryIDs = append(entryIDs, e.ID)
			releaseIDs = append(releaseIDs, e.ReleaseID)
		}
		creators, err := api.LoadEntryCreators(do, ids)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		releases, err := api.LoadReleases(do, releaseIDs)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		voterID := ""
		if viewer != nil {
			voterID = viewer.ID
		}
		voted, err := do.VotedEntryIDs(voterID, entryIDs)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		out := make([]entryResponse, 0, len(records))
		for _, e := range records {
			out = append(out, entryToResponse(e, creators, releases, voted[e.ID]))
		}
		response.Success(c, entryListResponse{
			Data: out,
			Meta: listMeta{
				Total:   total,
				Page:    filter.Page,
				Limit:   filter.Limit,
				HasMore: int64(filter.Page*filter.Limit) < total,
			},
		})
	}
}
