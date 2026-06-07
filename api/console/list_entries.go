// Package console contains the admin HTTP handlers exposed under
// /api/console/*: settings management and entry CRUD.
package console

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// entryResponse is the console-facing entry projection. Portal keeps
// its own copy because the shapes are expected to diverge soon (admin
// fields, status changes, etc.).
//
// Creator is omitted for anonymously submitted records (UserID == "") and
// for records whose author no longer exists. IsVoted reports whether the
// admin viewing the entry has voted on it — admins can vote from the
// Console under the same one-vote-per-user rule that applies on the Portal.
//
// AITopicIDs and EntryTypeAppliedByAI carry per-assignment AI
// provenance so the Console can render an AI badge next to chips
// that an analyzer applied (vs. ones an admin picked manually).
// AITopicIDs is a subset of the Topics array above. Portal responses
// never include these fields — AI provenance is an internal admin
// signal.
type entryResponse struct {
	ID                   string                  `json:"id"`
	Title                string                  `json:"title"`
	Description          string                  `json:"description,omitempty"`
	VoteCount            int                     `json:"voteCount"`
	IsVoted              bool                    `json:"isVoted"`
	CommentCount         int                     `json:"commentCount"`
	Source               string                  `json:"source"`
	IsInternal           bool                    `json:"isInternal"`
	Creator              *api.EntryCreator       `json:"creator,omitempty"`
	EntryType            *api.EntryTypeResponse  `json:"entryType,omitempty"`
	EntryTypeAppliedByAI bool                    `json:"entryTypeAppliedByAI"`
	// Suggested entry type (populated when the AI analyzer ran),
	// for the "Apply suggestion" badge on the Console detail sidebar.
	SuggestedEntryType       *api.EntryTypeResponse  `json:"suggestedEntryType,omitempty"`
	SuggestedEntryTypeReason string                  `json:"suggestedEntryTypeReason,omitempty"`
	Status                   api.EntryStatusResponse `json:"status"`
	Topics                   []EntryTopicResponse    `json:"topics"`
	AITopicIDs               []string                `json:"aiTopicIds"`
	Relations                []EntryRelationResponse `json:"relations"`
	Release                  *api.ReleaseResponse    `json:"release,omitempty"`
	GitHubIssue              *api.GitHubIssueResponse `json:"githubIssue,omitempty"`
	CreatedAt                string                  `json:"createdAt"`
	UpdatedAt                string                  `json:"updatedAt"`
} //@name Entry

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

// filterAIIDs returns the subset of aiIDs that still appear in
// currentIDs. Defends the wire payload against any drift between the
// AI-provenance subset and the main array (e.g. an old document that
// missed the cascade pull) — the Console UI should never show an AI
// badge for an ID that isn't actually on the entry. Returns an empty
// (non-nil) slice so JSON serialization emits [] rather than null.
func filterAIIDs(aiIDs, currentIDs []string) []string {
	out := []string{}
	if len(aiIDs) == 0 || len(currentIDs) == 0 {
		return out
	}
	current := make(map[string]struct{}, len(currentIDs))
	for _, id := range currentIDs {
		current[id] = struct{}{}
	}
	for _, id := range aiIDs {
		if _, ok := current[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

// entryTypeAppliedByAI derives whether the entry's current
// feedback type originated from the AI suggestion. There is no stored
// source field — the user explicitly chose the derivation strategy.
// The check requires three things: a non-empty EntryType, a
// EntryTypeAnalysis that ran (or was disabled after a successful
// run) and produced a suggestion, and the suggestion matching the
// current feedback type.
//
// Edge: an admin manually picking the same type the analyzer would
// have suggested counts as AI-applied here. Accepted tradeoff for
// avoiding a stored flag.
func entryTypeAppliedByAI(e models.Entry) bool {
	if e.EntryType == "" {
		return false
	}
	if e.EntryTypeAnalysis.SuggestedEntryTypeID == "" {
		return false
	}
	if string(e.EntryType) != e.EntryTypeAnalysis.SuggestedEntryTypeID {
		return false
	}
	switch e.EntryTypeAnalysis.Status {
	case models.AnalysisStatusDone, models.AnalysisStatusDisabled:
		return true
	}
	return false
}

// entryToResponse builds the wire payload. The users map resolves
// creators by UserID. Missing users (deleted, never existed) yield an
// absent field on the wire.
//
// relations carries the resolved Relations tab payload for the detail
// view. Pass nil from list-style callers — relations are not surfaced
// on list cells (per project convention, list cells are read-only and
// don't show per-row relation chips).
func entryToResponse(
	e models.Entry,
	users map[string]api.EntryCreator,
	topics map[string]EntryTopicResponse,
	releases map[string]api.ReleaseResponse,
	isVoted bool,
	relations []EntryRelationResponse,
) entryResponse {
	if relations == nil {
		relations = []EntryRelationResponse{}
	}
	r := entryResponse{
		ID:                   e.ID,
		Title:                e.Title,
		Description:          e.Description,
		VoteCount:            e.VoteCount,
		IsVoted:              isVoted,
		CommentCount:         e.CommentCount,
		Source:               string(e.Source),
		IsInternal:           e.IsInternal,
		Status:               api.BuildEntryStatus(e.Status),
		Topics:               projectEntryTopics(e.TopicIDs, topics),
		AITopicIDs:           filterAIIDs(e.AITopicIDs, e.TopicIDs),
		Relations:            relations,
		EntryTypeAppliedByAI: entryTypeAppliedByAI(e),
		CreatedAt:            iso(e.CreatedAt),
		UpdatedAt:            iso(e.UpdatedAt),
	}
	if e.ReleaseID != "" {
		if rel, ok := releases[e.ReleaseID]; ok {
			r.Release = &rel
		}
	}
	if e.UserID != "" {
		if c, ok := users[e.UserID]; ok {
			r.Creator = &c
		}
	}
	if et, ok := api.BuildEntryType(e.EntryType); ok {
		r.EntryType = &et
	}
	if suggested, ok := api.BuildEntryType(models.EntryType(e.EntryTypeAnalysis.SuggestedEntryTypeID)); ok {
		r.SuggestedEntryType = &suggested
		r.SuggestedEntryTypeReason = e.EntryTypeAnalysis.SuggestedEntryTypeReason
	}
	if gh, ok := api.BuildGitHubIssue(e.GitHubIssue); ok {
		r.GitHubIssue = &gh
	}
	return r
}

// listEntriesQuery is the bound query string for GET
// /api/console/entry. One struct, one @Param line, one BindQuery call —
// every optional filter the handler accepts lives here.
//
// Status filters on the hardcoded EntryStatus enum value
// ("new","evaluation","in-progress","completed","cancelled"). Empty
// string means "no restriction".
type listEntriesQuery struct {
	Search       string `form:"search"                       example:"dark mode"`
	Sort         string `form:"sort"        enums:"top,new"  example:"new"`
	Page         int    `form:"page"                         example:"1"`
	Limit        int    `form:"limit"                        example:"25"`
	EntryType string `form:"entryType" enums:"feature-request,bug,support,other" example:"bug"`
	Status       string `form:"status"       enums:"new,evaluation,in-progress,completed,cancelled" example:"new"`
	AuthorID     string `form:"authorId"`
	TopicID      string `form:"topicId"`
}

// ListEntriesHandler returns a paginated entry list (admin view).
//
//	@ID			console-list-entries
//	@Summary	List entries (admin)
//	@Tags		Console
//	@Produce	json
//	@Param		request	query		listEntriesQuery	false	"Filters"
//	@Success	200		{object}	entryListResponse
//	@Router		/api/console/entry [get]
func ListEntriesHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request listEntriesQuery
		if err := c.BindQuery(&request); err != nil {
			response.ValidationError(c, err)
			return
		}
		filter := dataoperations.EntryListFilter{
			Search:  request.Search,
			Sort:    request.Sort,
			Page:    request.Page,
			Limit:   request.Limit,
			OwnerID: request.AuthorID,
			TopicID: request.TopicID,
		}
		if request.EntryType != "" {
			ft, err := resolveEntryType(request.EntryType)
			if err != nil {
				response.BadRequestWithMessage(c, err.Error())
				return
			}
			filter.EntryType = ft
		}
		if request.Status != "" {
			s, err := resolveEntryStatus(request.Status)
			if err != nil {
				response.BadRequestWithMessage(c, err.Error())
				return
			}
			filter.Status = s
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
		userIDs := make([]string, 0, len(records))
		entryIDs := make([]string, 0, len(records))
		topicIDs := make([]string, 0, len(records))
		for _, e := range records {
			userIDs = append(userIDs, e.UserID)
			entryIDs = append(entryIDs, e.ID)
			topicIDs = append(topicIDs, e.TopicIDs...)
		}
		creators, err := api.LoadEntryCreators(do, userIDs)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		topics, err := LoadEntryTopics(do, topicIDs)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		releaseIDs := make([]string, 0, len(records))
		for _, e := range records {
			releaseIDs = append(releaseIDs, e.ReleaseID)
		}
		releases, err := api.LoadReleases(do, releaseIDs)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		voterID := ""
		if u := middlewares.CurrentUser(c); u != nil {
			voterID = u.ID
		}
		voted, err := do.VotedEntryIDs(voterID, entryIDs)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		out := make([]entryResponse, 0, len(records))
		for _, e := range records {
			out = append(out, entryToResponse(e, creators, topics, releases, voted[e.ID], nil))
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
