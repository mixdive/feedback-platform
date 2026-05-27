package console

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/pkg/github"
)

// integrationsResponse is the envelope returned by GET
// /api/console/integrations. Wraps the same per-integration payloads
// the main settings GET serves up so the React app reuses the same
// types regardless of which endpoint it called.
type integrationsResponse struct {
	GitHub githubIntegrationPayload `json:"github"`
} //@name IntegrationsResponse

// GetIntegrationsHandler returns the current state of every configured
// integration. Admin+editor — read-only consumers (entry detail page
// needs to know whether the GitHub button should appear) include
// editors.
//
//	@ID			console-get-integrations
//	@Summary	List integrations
//	@Tags		Console
//	@Produce	json
//	@Success	200	{object}	integrationsResponse
//	@Router		/api/console/integrations [get]
func GetIntegrationsHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := do.GetSettings()
		if err != nil || s == nil {
			response.SystemError(c, err)
			return
		}
		payload := buildIntegrationsPayload(s.Integrations)
		response.Success(c, integrationsResponse{GitHub: payload.GitHub})
	}
}

// updateGitHubIntegrationRequest is the body for PUT
// /api/console/integrations/github. All three fields are required so
// the handler can verify the (owner, repo, token) triple against
// GitHub before persisting. To rotate just the token, the admin
// re-sends the same owner/repo with a new token — the verify call
// catches typos in the unchanged fields too.
type updateGitHubIntegrationRequest struct {
	Owner string `json:"owner" binding:"required" example:"acme-corp"`
	Repo  string `json:"repo"  binding:"required" example:"webapp"`
	Token string `json:"token" binding:"required" example:"ghp_..."`
} //@name consoleUpdateGitHubIntegrationRequest

// UpdateGitHubIntegrationHandler persists or rotates the GitHub
// integration. Before writing, calls GitHub's GET /repos/{owner}/{repo}
// with the supplied token — a 401 means the PAT is invalid or lacks
// scopes; a 404 means the owner/repo path is wrong. Either way the
// admin sees the upstream message verbatim and the settings doc is
// untouched.
//
// Disabled is intentionally not touched here — the admin may have the
// kill switch on while rotating credentials, and a successful PUT
// shouldn't silently flip it off.
//
//	@ID			console-update-github-integration
//	@Summary	Connect or update the GitHub integration (admin)
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		request	body		updateGitHubIntegrationRequest	true	"GitHub credentials"
//	@Success	200		{object}	integrationsResponse
//	@Failure	400		{object}	response.ApiError
//	@Router		/api/console/integrations/github [put]
func UpdateGitHubIntegrationHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req updateGitHubIntegrationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		owner := strings.TrimSpace(req.Owner)
		repo := strings.TrimSpace(req.Repo)
		token := strings.TrimSpace(req.Token)
		if owner == "" || repo == "" || token == "" {
			response.BadRequestWithMessage(c, "Owner, repo, and token are all required.")
			return
		}
		// Defend against "owner/repo" pasted as a single field. The
		// GitHub URL the admin most likely copied has the slash; we
		// want each piece in its own input.
		if strings.Contains(owner, "/") || strings.Contains(repo, "/") {
			response.BadRequestWithMessage(c, "Owner and repo must be supplied as separate fields, not a single owner/repo string.")
			return
		}

		if err := github.VerifyRepoAccess(c.Request.Context(), token, owner, repo); err != nil {
			response.BadRequestWithMessage(c, err.Error())
			return
		}

		actorID := ""
		if u := middlewares.CurrentUser(c); u != nil {
			actorID = u.ID
		}
		patch := map[string]any{
			"integrations.github.owner":       owner,
			"integrations.github.repo":        repo,
			"integrations.github.token":       token,
			"integrations.github.connectedat": time.Now().UTC(),
			"integrations.github.connectedby": actorID,
		}
		if err := do.UpdateSettings(patch); err != nil {
			response.SystemError(c, err)
			return
		}
		updated, err := do.GetSettings()
		if err != nil || updated == nil {
			response.SystemError(c, err)
			return
		}
		payload := buildIntegrationsPayload(updated.Integrations)
		response.Success(c, integrationsResponse{GitHub: payload.GitHub})
	}
}

// DeleteGitHubIntegrationHandler clears every field on
// Settings.Integrations.GitHub — owner, repo, token, and the audit
// pair. Disabled is also reset since "not connected" reads the same
// regardless. Linked entries keep their Entry.GitHubIssue intact: the
// upstream issues still exist on GitHub, and we don't want to strip
// the back-link just because the credentials rotated.
//
//	@ID			console-delete-github-integration
//	@Summary	Disconnect the GitHub integration (admin)
//	@Tags		Console
//	@Produce	json
//	@Success	200	{object}	integrationsResponse
//	@Router		/api/console/integrations/github [delete]
func DeleteGitHubIntegrationHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		patch := map[string]any{
			"integrations.github.owner":       "",
			"integrations.github.repo":        "",
			"integrations.github.token":       "",
			"integrations.github.connectedat": time.Time{},
			"integrations.github.connectedby": "",
			"integrations.github.disabled":    false,
		}
		if err := do.UpdateSettings(patch); err != nil {
			response.SystemError(c, err)
			return
		}
		updated, err := do.GetSettings()
		if err != nil || updated == nil {
			response.SystemError(c, err)
			return
		}
		payload := buildIntegrationsPayload(updated.Integrations)
		response.Success(c, integrationsResponse{GitHub: payload.GitHub})
	}
}
