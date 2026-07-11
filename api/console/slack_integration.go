package console

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/middlewares"
	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/slack"
)

// Slack connects via the admin's own Slack App using OAuth
// (incoming-webhook scope). The webhook that ultimately delivers
// messages is minted by Slack when the admin picks a channel, so there
// is no webhook field to paste — the admin supplies their App's Client
// ID + Secret once, then the browser round-trips through Slack's channel
// picker. See models.SlackIntegration for the field layout.

// slackStateCookieName holds the one-time CSRF state that ties an
// /authorize kickoff to its /callback. Path-scoped to the Slack routes
// and SameSite=Lax so it survives the top-level GET redirect back from
// slack.com.
const slackStateCookieName = "mixdive_slack_oauth"

// slackSettingsPath is the Console page the callback bounces the browser
// back to, carrying a ?slack=connected|error(&message=) query the page
// turns into a toast.
const slackSettingsPath = "/console/settings/integrations"

// slackRedirectURI builds the absolute OAuth callback URL from the
// request host. It must be registered verbatim on the Slack App and must
// match byte-for-byte between the authorize and exchange calls (both
// derive it from the same deployment host). Honors X-Forwarded-Proto so
// it reads https behind a TLS-terminating proxy (Cloud Run).
func slackRedirectURI(c *gin.Context) string {
	host := c.Request.Host
	if host == "" {
		return ""
	}
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + host + "/api/console/integrations/slack/callback"
}

// redirectToSlackSettings bounces the browser back to the Console
// Integrations page with a status the React page renders as a toast.
func redirectToSlackSettings(c *gin.Context, status, msg string) {
	q := url.Values{}
	q.Set("slack", status)
	if msg != "" {
		q.Set("message", msg)
	}
	c.Redirect(http.StatusFound, slackSettingsPath+"?"+q.Encode())
}

// slackModeFromQuery reads the popup flag the "Add to Slack" button sets
// when it opens the flow in a popup window rather than the full page.
func slackModeFromQuery(c *gin.Context) string {
	if c.Query("popup") == "1" {
		return "popup"
	}
	return "page"
}

// slackStateMode recovers the mode that slackModeFromQuery encoded into
// the OAuth state ("<random>.<mode>"). base64url has no '.', so the
// suffix is unambiguous; anything unexpected falls back to "page".
func slackStateMode(state string) string {
	if i := strings.LastIndex(state, "."); i >= 0 && state[i+1:] == "popup" {
		return "popup"
	}
	return "page"
}

// finishSlack ends the OAuth flow. In page mode it redirects back to the
// settings page (the page reads ?slack= and toasts). In popup mode it
// returns a tiny HTML page that posts the result to the opener window
// and closes itself, so the settings page never navigates away.
func finishSlack(c *gin.Context, mode, status, msg string) {
	if mode != "popup" {
		redirectToSlackSettings(c, status, msg)
		return
	}
	// encoding/json escapes <, >, & to \u00xx, so the payload is safe to
	// inline inside the <script> without a </script> break-out.
	payload, _ := json.Marshal(map[string]string{
		"source":  "mixdive-slack",
		"status":  status,
		"message": msg,
	})
	html := `<!doctype html><html><head><meta charset="utf-8"><title>Slack</title></head>` +
		`<body style="font:14px system-ui,sans-serif;color:#52525b;padding:24px">You can close this window.` +
		`<script>(function(){try{if(window.opener){window.opener.postMessage(` + string(payload) +
		`,window.location.origin);}}catch(e){}window.close();})();</script></body></html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func setSlackStateCookie(c *gin.Context, state string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     slackStateCookieName,
		Value:    state,
		Path:     "/api/console/integrations/slack",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   c.Request.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func readSlackStateCookie(c *gin.Context) string {
	ck, err := c.Request.Cookie(slackStateCookieName)
	if err != nil {
		return ""
	}
	return ck.Value
}

func clearSlackStateCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     slackStateCookieName,
		Value:    "",
		Path:     "/api/console/integrations/slack",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.Request.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func randomSlackState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// updateSlackIntegrationRequest is the body for PUT
// /api/console/integrations/slack. It saves the admin's Slack App
// credentials and the per-event toggles — NOT a webhook (that comes from
// the OAuth flow). ClientSecret is optional once stored: sending it blank
// keeps the existing secret so the admin can flip toggles or rotate only
// the Client ID without re-pasting it. Blank fields are never cleared —
// full removal is Disconnect.
type updateSlackIntegrationRequest struct {
	ClientID        string `json:"clientId"     example:"1234567890.1234567890"`
	ClientSecret    string `json:"clientSecret" example:"abc123..."`
	NotifyOnEntry   bool   `json:"notifyOnEntry"`
	NotifyOnComment bool   `json:"notifyOnComment"`
	NotifyOnVote    bool   `json:"notifyOnVote"`
} //@name consoleUpdateSlackIntegrationRequest

// UpdateSlackIntegrationHandler saves the admin's Slack App credentials
// and the per-event notification toggles. Credentials can't be verified
// without running the full OAuth flow, so this just persists them; the
// real check happens when the admin clicks "Add to Slack" and the code
// exchange succeeds or fails. Blank clientId/clientSecret are left
// untouched so a toggle-only save (or a Client-ID-only rotation) doesn't
// wipe the stored secret.
//
//	@ID			console-update-slack-integration
//	@Summary	Save Slack App credentials and event toggles (admin)
//	@Tags		Console
//	@Accept		json
//	@Produce	json
//	@Param		request	body		updateSlackIntegrationRequest	true	"Slack App credentials and event toggles"
//	@Success	200		{object}	integrationsResponse
//	@Failure	400		{object}	response.ApiError
//	@Router		/api/console/integrations/slack [put]
func UpdateSlackIntegrationHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req updateSlackIntegrationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ValidationError(c, err)
			return
		}
		patch := map[string]any{
			"integrations.slack.notifyonentry":   req.NotifyOnEntry,
			"integrations.slack.notifyoncomment": req.NotifyOnComment,
			"integrations.slack.notifyonvote":    req.NotifyOnVote,
		}
		if cid := strings.TrimSpace(req.ClientID); cid != "" {
			patch["integrations.slack.clientid"] = cid
		}
		if cs := strings.TrimSpace(req.ClientSecret); cs != "" {
			patch["integrations.slack.clientsecret"] = cs
		}
		if err := do.UpdateSettings(patch); err != nil {
			response.SystemError(c, err)
			return
		}
		respondIntegrations(c, do)
	}
}

// SlackAuthorizeHandler kicks off the OAuth flow: it mints a CSRF state,
// drops it in a short-lived cookie, and 302-redirects the admin's
// browser to Slack's authorize page (where they pick a workspace +
// channel). This is a browser navigation, so every outcome is a redirect
// — a missing-credentials error bounces back to the settings page rather
// than returning JSON.
//
//	@ID			console-slack-authorize
//	@Summary	Start the Slack OAuth flow (admin)
//	@Tags		Console
//	@Success	302
//	@Router		/api/console/integrations/slack/authorize [get]
func SlackAuthorizeHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		mode := slackModeFromQuery(c)
		s, err := do.GetSettings()
		if err != nil || s == nil {
			response.SystemError(c, err)
			return
		}
		slk := s.Integrations.Slack
		if !models.IsSlackAppConfigured(slk) {
			finishSlack(c, mode, "error", "Enter your Slack App Client ID and Client Secret first.")
			return
		}
		rnd, err := randomSlackState()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		// Carry popup/page mode inside the state so the callback (which
		// only gets code + state back from Slack) knows how to finish.
		state := rnd + "." + mode
		setSlackStateCookie(c, state)
		c.Redirect(http.StatusFound, slack.AuthorizeURL(slk.ClientID, slackRedirectURI(c), state))
	}
}

// SlackCallbackHandler is where Slack sends the admin back after they
// approve (or deny). It verifies the CSRF state against the cookie,
// exchanges the code for an Incoming Webhook, stores the connection, and
// redirects to the settings page with a toast. Every branch redirects —
// the admin never sees raw JSON.
//
//	@ID			console-slack-callback
//	@Summary	Slack OAuth callback (admin)
//	@Tags		Console
//	@Param		code	query	string	false	"OAuth code"
//	@Param		state	query	string	false	"CSRF state"
//	@Success	302
//	@Router		/api/console/integrations/slack/callback [get]
func SlackCallbackHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// The state cookie is single-use regardless of outcome.
		state := c.Query("state")
		cookieState := readSlackStateCookie(c)
		clearSlackStateCookie(c)
		mode := slackStateMode(state)

		if e := c.Query("error"); e != "" {
			finishSlack(c, mode, "error", "Slack authorization was cancelled.")
			return
		}
		if state == "" || cookieState == "" || state != cookieState {
			finishSlack(c, mode, "error", "Slack authorization could not be verified. Please try again.")
			return
		}
		code := c.Query("code")
		if code == "" {
			finishSlack(c, mode, "error", "Slack did not return an authorization code.")
			return
		}

		s, err := do.GetSettings()
		if err != nil || s == nil {
			response.SystemError(c, err)
			return
		}
		slk := s.Integrations.Slack
		if !models.IsSlackAppConfigured(slk) {
			finishSlack(c, mode, "error", "Slack App credentials are no longer configured.")
			return
		}

		res, err := slack.ExchangeCode(c.Request.Context(), slk.ClientID, slk.ClientSecret, code, slackRedirectURI(c))
		if err != nil {
			finishSlack(c, mode, "error", err.Error())
			return
		}

		actorID := ""
		if u := middlewares.CurrentUser(c); u != nil {
			actorID = u.ID
		}
		patch := map[string]any{
			"integrations.slack.webhookurl":       res.WebhookURL,
			"integrations.slack.channelname":      res.ChannelName,
			"integrations.slack.teamname":         res.TeamName,
			"integrations.slack.accesstoken":      res.AccessToken,
			"integrations.slack.connectedat":      time.Now().UTC(),
			"integrations.slack.connectedby":      actorID,
			"integrations.slack.lasterrorat":      time.Time{},
			"integrations.slack.lasterrormessage": "",
		}
		if err := do.UpdateSettings(patch); err != nil {
			response.SystemError(c, err)
			return
		}
		finishSlack(c, mode, "connected", "")
	}
}

// DisconnectSlackIntegrationHandler fully removes the Slack integration:
// it revokes the bot token upstream (best-effort) so a stale token can't
// keep posting, then clears every Slack field — App credentials,
// connection, toggles, audit, and last-error. "Change the channel" is a
// separate flow (re-run authorize); this is the "I'm done with Slack"
// button.
//
//	@ID			console-delete-slack-integration
//	@Summary	Disconnect and remove the Slack integration (admin)
//	@Tags		Console
//	@Produce	json
//	@Success	200	{object}	integrationsResponse
//	@Router		/api/console/integrations/slack [delete]
func DisconnectSlackIntegrationHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if s, err := do.GetSettings(); err == nil && s != nil && s.Integrations.Slack.AccessToken != "" {
			// Best-effort — a revoke failure must not block the local clear.
			_ = slack.Revoke(c.Request.Context(), s.Integrations.Slack.AccessToken)
		}
		patch := map[string]any{
			"integrations.slack.clientid":         "",
			"integrations.slack.clientsecret":     "",
			"integrations.slack.webhookurl":       "",
			"integrations.slack.channelname":      "",
			"integrations.slack.teamname":         "",
			"integrations.slack.accesstoken":      "",
			"integrations.slack.notifyonentry":    false,
			"integrations.slack.notifyoncomment":  false,
			"integrations.slack.notifyonvote":     false,
			"integrations.slack.disabled":         false,
			"integrations.slack.connectedat":      time.Time{},
			"integrations.slack.connectedby":      "",
			"integrations.slack.lasterrorat":      time.Time{},
			"integrations.slack.lasterrormessage": "",
		}
		if err := do.UpdateSettings(patch); err != nil {
			response.SystemError(c, err)
			return
		}
		respondIntegrations(c, do)
	}
}
