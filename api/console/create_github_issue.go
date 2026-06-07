package console

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/aianalyzer"
	"github.com/mixdive/feedback-platform/pkg/github"
)

// createGitHubIssueResponse is what the handler returns on success.
// Same shape as a full entry response — the React app uses the
// updated entry to flip the button to "View on GitHub" without a
// second round trip.
//
// We could return only the GitHubIssue sub-doc, but list-page caches
// hold full entries and the smaller payload would force a refetch.
// Returning the entry keeps the cache-update pattern in lockstep with
// PATCH /api/console/entry/:id.

// CreateGitHubIssueHandler publishes a feature-request or bug entry as
// a GitHub issue under the repo configured in
// Settings.Integrations.GitHub. AI generates the issue title and body
// from the entry; the handler appends a footer linking back to the
// Mixdive entry detail page and a machine-readable HTML comment
// carrying the entry ID for future bidirectional sync.
//
// Validation gates (return 400 with an actionable message):
//   - Entry exists
//   - Entry already has a linked issue (Number > 0) → 409
//   - EntryType is feature-request or bug
//   - GitHub integration is active (not Disabled, owner/repo/token set)
//   - AI is enabled with a key set (the summarizer needs both)
//
// Failure modes (500 with the upstream message):
//   - Anthropic call fails → no GitHub side effect, admin retries
//   - GitHub call fails → no entry mutation, admin retries
//
// Records a single ActivityTypeGitHubIssueCreated row with TargetID =
// the issue URL so the timeline renders a clickable line without
// re-reading the entry.
//
//	@ID			console-create-github-issue
//	@Summary	Publish entry as GitHub issue (admin)
//	@Tags		Console
//	@Produce	json
//	@Param		id	path		string	true	"entry id"
//	@Success	200	{object}	entryResponse
//	@Failure	400	{object}	response.ApiError
//	@Failure	404	{object}	response.ApiError
//	@Failure	409	{object}	response.ApiError
//	@Router		/api/console/entry/{id}/github-issue [post]
func CreateGitHubIssueHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		existing, err := do.FindEntryByID(id)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		if existing == nil {
			response.NotFoundWithMessage(c, "Entry not found.")
			return
		}
		if existing.GitHubIssue.Number > 0 {
			response.ConflictWithMessage(c, "This entry is already linked to a GitHub issue.")
			return
		}
		if !isGitHubEligible(existing.EntryType) {
			response.BadRequestWithMessage(c, "Only feature requests and bugs can be published to GitHub.")
			return
		}

		s, err := do.GetSettings()
		if err != nil || s == nil {
			response.SystemError(c, err)
			return
		}
		if !models.IsGitHubIntegrationActive(s.Integrations.GitHub) {
			response.BadRequestWithMessage(c, "GitHub integration is not configured. Connect it in Settings → Integrations.")
			return
		}
		if !s.AI.Enabled || s.AI.APIKey == "" {
			response.BadRequestWithMessage(c, "AI is required to summarize the entry. Enable AI in Settings → AI.")
			return
		}

		snap := aianalyzer.Snapshot{
			Enabled: s.AI.Enabled,
			APIKey:  s.AI.APIKey,
			Model:   s.AI.Model,
		}
		draft, ok, err := aianalyzer.SummarizeGitHubIssue(c.Request.Context(), snap, *existing)
		if err != nil {
			response.ErrorWithStatusCodeAndMessage(c, 500, "Couldn't summarize the entry: "+err.Error())
			return
		}
		if !ok {
			response.BadRequestWithMessage(c, "AI is required to summarize the entry. Enable AI in Settings → AI.")
			return
		}

		body := appendMixdiveFooter(draft.Body, mixdiveEntryURL(c, id), id)

		issue, err := github.CreateIssue(c.Request.Context(), github.CreateIssueParams{
			Token: s.Integrations.GitHub.Token,
			Owner: s.Integrations.GitHub.Owner,
			Repo:  s.Integrations.GitHub.Repo,
			Title: draft.Title,
			Body:  body,
		})
		if err != nil {
			response.ErrorWithStatusCodeAndMessage(c, 500, "Couldn't create the GitHub issue: "+err.Error())
			return
		}

		actorID := ""
		if u := middlewares.CurrentUser(c); u != nil {
			actorID = u.ID
		}
		link := models.GitHubIssue{
			URL:       issue.HTMLURL,
			Owner:     s.Integrations.GitHub.Owner,
			Repo:      s.Integrations.GitHub.Repo,
			Number:    issue.Number,
			CreatedAt: time.Now().UTC(),
			CreatedBy: actorID,
		}
		if err := do.SetEntryGitHubIssue(id, link); err != nil {
			response.SystemError(c, err)
			return
		}
		if err := recordAdminActivity(do, id, actorID, models.ActivityTypeGitHubIssueCreated, "", "", issue.HTMLURL); err != nil {
			response.SystemError(c, err)
			return
		}

		updated, err := do.FindEntryByID(id)
		if err != nil || updated == nil {
			response.SystemError(c, err)
			return
		}
		creators, err := api.LoadEntryCreators(do, []string{updated.UserID})
		if err != nil {
			response.SystemError(c, err)
			return
		}
		topics, err := LoadEntryTopics(do, updated.TopicIDs)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		releases, err := api.LoadReleases(do, []string{updated.ReleaseID})
		if err != nil {
			response.SystemError(c, err)
			return
		}
		isVoted := false
		if u := middlewares.CurrentUser(c); u != nil {
			v, err := do.FindVote(u.ID, id)
			if err != nil {
				response.SystemError(c, err)
				return
			}
			isVoted = v != nil
		}
		relations, err := resolveRelationsForEntry(do, *updated)
		if err != nil {
			response.SystemError(c, err)
			return
		}
		response.Success(c, entryToResponse(*updated, creators, topics, releases, isVoted, relations))
	}
}

// isGitHubEligible gates which entry types can be published. Locked to
// feature requests and bugs — support/other entries don't belong in an
// engineering issue tracker.
func isGitHubEligible(t models.EntryType) bool {
	switch t {
	case models.EntryTypeFeatureRequest, models.EntryTypeBug:
		return true
	}
	return false
}

// mixdiveEntryURL builds the back-link URL to the Console entry detail
// page from the inbound request. We derive scheme + host instead of
// hardcoding a base URL because Mixdive is self-hostable and has no
// notion of its own canonical URL — the request is the source of truth.
func mixdiveEntryURL(c *gin.Context, entryID string) string {
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := c.Request.Host
	return fmt.Sprintf("%s://%s/console/entry/%s", scheme, host, entryID)
}

// appendMixdiveFooter glues the back-reference to the AI-drafted body.
// The visible footer gives a clickable Mixdive link; the HTML comment
// carries the raw entry ID for any future "sync GitHub state back into
// Mixdive" worker — it's invisible in rendered markdown but greppable
// from the API.
func appendMixdiveFooter(body, entryURL, entryID string) string {
	body = strings.TrimRight(body, " \t\r\n")
	return body + fmt.Sprintf("\n\n---\n\n_Originally submitted on Mixdive: [view entry →](%s)_\n\n<!-- mixdive-entry-id: %s -->\n", entryURL, entryID)
}
