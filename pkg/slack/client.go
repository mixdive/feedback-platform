// Package slack is a hairline-thin wrapper around Slack Incoming
// Webhooks — the single Slack surface Mixdive posts to. v0.1 sends
// plain-text messages to one webhook URL; anything richer (Block Kit,
// interactivity, per-channel routing, slash commands) waits for a real
// customer ask.
//
// We deliberately avoid the official slack-go SDK: it brings a large
// dependency surface and we'd use < 1% of it. Direct net/http keeps the
// binary small and the failure modes legible — mirrors pkg/github.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// requestTimeout caps a single webhook POST. Slack responds in well
// under a second; the cap guards against hung TCP that would otherwise
// keep the fire-and-forget notifier goroutine alive indefinitely.
const requestTimeout = 10 * time.Second

// PostMessage posts a plain-text message to a Slack Incoming Webhook.
// The webhook URL already encodes the destination channel, so text is
// the only payload we send. Errors wrap Slack's response body so the
// caller can log a legible reason ("no_service" for a revoked webhook,
// "invalid_payload", "channel_not_found") instead of a bare status code.
//
// Used for both the verify-on-save test ping (the connect handler posts
// a confirmation message and rejects the save on error) and the runtime
// best-effort notifications fired from the portal handlers.
func PostMessage(ctx context.Context, webhookURL, text string) error {
	if strings.TrimSpace(webhookURL) == "" {
		return fmt.Errorf("slack: missing webhook url")
	}
	body, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("slack: marshal request body: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mixdive")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack: post message: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	// A Slack webhook returns 200 with the literal body "ok" on success.
	// Everything else (404 no_service, 400 invalid_payload, 403
	// action_prohibited) is a failure whose body names the reason.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack: post failed (%s): %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return nil
}
