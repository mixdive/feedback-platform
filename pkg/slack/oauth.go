package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	// authorizeEndpoint is where the admin's browser is sent to pick a
	// workspace + channel. The incoming-webhook scope is what makes Slack
	// render the channel picker and mint a webhook on approval.
	authorizeEndpoint = "https://slack.com/oauth/v2/authorize"
	// oauthScope is the only scope we request — it returns an Incoming
	// Webhook bound to the channel the admin selects, nothing more.
	oauthScope = "incoming-webhook"
)

// accessEndpoint exchanges the temporary code for the webhook URL;
// revokeEndpoint invalidates the bot token on Disconnect. Package-level
// vars (not consts) so tests can point them at an httptest server.
var (
	accessEndpoint = "https://slack.com/api/oauth.v2.access"
	revokeEndpoint = "https://slack.com/api/auth.revoke"
)

// AuthorizeURL builds the Slack OAuth authorize URL the admin's browser
// is redirected to. redirectURI must exactly match one registered on the
// Slack App and is echoed back on the code exchange. state is an opaque
// CSRF token the caller also stores in a cookie and re-checks on
// callback.
func AuthorizeURL(clientID, redirectURI, state string) string {
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("scope", oauthScope)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	return authorizeEndpoint + "?" + q.Encode()
}

// OAuthResult is the slice of Slack's oauth.v2.access response we keep.
// URL is the Incoming Webhook to POST to; ChannelName / TeamName are for
// display; AccessToken is retained only to auth.revoke on Disconnect.
type OAuthResult struct {
	WebhookURL  string
	ChannelName string
	TeamName    string
	AccessToken string
}

// ExchangeCode trades the temporary code from the OAuth callback for a
// permanent Incoming Webhook. Slack answers 200 with an {ok:false,error}
// body on failure (bad code, redirect_uri mismatch, wrong client
// secret), so we key off the `ok` field rather than the HTTP status.
func ExchangeCode(ctx context.Context, clientID, clientSecret, code, redirectURI string) (OAuthResult, error) {
	if clientID == "" || clientSecret == "" || code == "" {
		return OAuthResult{}, fmt.Errorf("slack: missing client id/secret/code")
	}
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)

	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, accessEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return OAuthResult{}, fmt.Errorf("slack: build oauth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mixdive")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return OAuthResult{}, fmt.Errorf("slack: oauth exchange: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var r struct {
		OK          bool   `json:"ok"`
		Error       string `json:"error"`
		AccessToken string `json:"access_token"`
		Team        struct {
			Name string `json:"name"`
		} `json:"team"`
		IncomingWebhook struct {
			Channel string `json:"channel"`
			URL     string `json:"url"`
		} `json:"incoming_webhook"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return OAuthResult{}, fmt.Errorf("slack: decode oauth response: %w", err)
	}
	if !r.OK {
		reason := r.Error
		if reason == "" {
			reason = "unknown_error"
		}
		return OAuthResult{}, fmt.Errorf("slack: authorization failed (%s)", reason)
	}
	if r.IncomingWebhook.URL == "" {
		return OAuthResult{}, fmt.Errorf("slack: no incoming webhook returned (was the incoming-webhook scope granted?)")
	}
	return OAuthResult{
		WebhookURL:  r.IncomingWebhook.URL,
		ChannelName: r.IncomingWebhook.Channel,
		TeamName:    r.Team.Name,
		AccessToken: r.AccessToken,
	}, nil
}

// Revoke invalidates a bot token via auth.revoke. Best-effort: called on
// Disconnect so a stale token can't keep posting, but a failure here
// (already-revoked token, network blip) must not block the local
// disconnect. Returns the error for logging; the caller ignores it.
func Revoke(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	form := url.Values{}
	form.Set("token", token)

	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, revokeEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("slack: build revoke request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mixdive")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack: revoke: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var r struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("slack: decode revoke response: %w", err)
	}
	if !r.OK {
		return fmt.Errorf("slack: revoke failed (%s)", r.Error)
	}
	return nil
}
