package portal

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/pkg/aianalyzer"
)

// findSimilarRequest is the body for POST /api/portal/entry/find-similar.
// Title is required (the user has at least typed something); description
// is optional (we run on title-blur, before the description is filled).
type findSimilarRequest struct {
	Title       string `json:"title"        binding:"required,min=3,max=200" example:"Add dark mode"`
	Description string `json:"description,omitempty"                         example:"A system-aware toggle would be great."`
} //@name portalFindSimilarRequest

// findSimilarMatch is one entry the AI judged similar to the draft,
// plus a one-sentence reason from the model.
type findSimilarMatch struct {
	Entry  entryResponse `json:"entry"`
	Reason string        `json:"reason,omitempty"`
} //@name FindSimilarMatch

// findSimilarResponse is the envelope. AIEnabled is false when the
// admin hasn't configured an Anthropic key — the UI then knows the
// empty result means "no AI", not "no similar entries".
type findSimilarResponse struct {
	Data      []findSimilarMatch `json:"data"`
	AIEnabled bool               `json:"aiEnabled"`
} //@name PortalFindSimilarResponse

// FindSimilarHandler runs a synchronous Claude call to find existing
// entries that describe the same idea as the user's draft. Portal
// scope: only public (non-internal) entries are surfaced as candidates.
//
//	@ID			portal-find-similar-entries
//	@Summary	Find similar existing entries (AI)
//	@Tags		Portal
//	@Accept		json
//	@Produce	json
//	@Param		request	body		findSimilarRequest	true	"Draft"
//	@Success	200		{object}	findSimilarResponse
//	@Failure	400		{object}	response.ApiError
//	@Router		/api/portal/entry/find-similar [post]
func FindSimilarHandler(do dataoperations.Store, worker *aianalyzer.Worker) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req findSimilarRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}

		snap := worker.Snapshot()
		if !snap.Enabled || snap.APIKey == "" {
			response.Success(c, findSimilarResponse{Data: []findSimilarMatch{}, AIEnabled: false})
			return
		}

		candidates, err := do.ListEntriesForSimilarity(true)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if len(candidates) == 0 {
			response.Success(c, findSimilarResponse{Data: []findSimilarMatch{}, AIEnabled: true})
			return
		}

		matches, err := aianalyzer.FindSimilarEntries(c.Request.Context(), snap, req.Title, req.Description, aianalyzer.CandidatesFromEntries(candidates))
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if len(matches) == 0 {
			response.Success(c, findSimilarResponse{Data: []findSimilarMatch{}, AIEnabled: true})
			return
		}

		byID := make(map[string]int, len(candidates))
		for i, e := range candidates {
			byID[e.ID] = i
		}
		userIDs := make([]string, 0, len(matches))
		entryIDs := make([]string, 0, len(matches))
		releaseIDs := make([]string, 0, len(matches))
		for _, m := range matches {
			idx, ok := byID[m.EntryID]
			if !ok {
				continue
			}
			e := candidates[idx]
			userIDs = append(userIDs, e.UserID)
			entryIDs = append(entryIDs, e.ID)
			releaseIDs = append(releaseIDs, e.ReleaseID)
		}
		creators, err := api.LoadEntryCreators(do, userIDs)
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
		if u := middlewares.CurrentUser(c); u != nil {
			voterID = u.ID
		}
		voted, err := do.VotedEntryIDs(voterID, entryIDs)
		if err != nil {
			response.SystemError(c, err)
			return
		}

		out := make([]findSimilarMatch, 0, len(matches))
		for _, m := range matches {
			idx, ok := byID[m.EntryID]
			if !ok {
				continue
			}
			e := candidates[idx]
			out = append(out, findSimilarMatch{
				Entry:  entryToResponse(e, creators, releases, voted[e.ID]),
				Reason: m.Reason,
			})
		}
		response.Success(c, findSimilarResponse{Data: out, AIEnabled: true})
	}
}
