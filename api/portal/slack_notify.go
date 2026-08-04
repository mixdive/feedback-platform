package portal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api"
	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/slack"
)

// slackNotifyTimeout bounds a single fire-and-forget webhook post. The
// notifier runs in its own goroutine detached from the portal request,
// so this cap is the only thing stopping it lingering on a hung
// connection.
const slackNotifyTimeout = 15 * time.Second

// slackEventEnabled selects the per-event toggle a given notification
// depends on — e.g. func(s SlackIntegration) bool { return s.NotifyOnEntry }.
type slackEventEnabled func(models.SlackIntegration) bool

// notifySlack posts text to the configured Slack webhook in the
// background when the integration is active and the given per-event
// toggle is on. The settings read, the gating check, the POST, and
// error recording ALL happen inside the goroutine, so the portal request
// that triggered it pays zero added latency and can never fail because
// Slack is slow or misconfigured.
//
// Best-effort by design: a delivery failure is stamped onto the settings
// doc (LastErrorAt/Message) so the admin sees it on the Integrations
// page, but we deliberately don't retry — a missed portal notification
// isn't worth a durable queue. Callers must have already excluded
// internal entries; Slack mirrors public portal activity only.
func notifySlack(do dataoperations.Store, wantsEvent slackEventEnabled, text string) {
	go func() {
		s, err := do.GetSettings()
		if err != nil || s == nil {
			return
		}
		slk := s.Integrations.Slack
		if !models.IsSlackIntegrationActive(slk) || !wantsEvent(slk) {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), slackNotifyTimeout)
		defer cancel()
		if err := slack.PostMessage(ctx, slk.WebhookURL, text); err != nil {
			_ = do.UpdateSettings(map[string]any{
				"integrations.slack.lasterrorat":      time.Now().UTC(),
				"integrations.slack.lasterrormessage": err.Error(),
			})
		}
	}()
}

// consoleEntryURL builds an absolute link to an entry's Console detail
// page from the incoming request's host/scheme. Single binary: the host
// serving the Portal API is the same host serving the Console SPA, so
// c.Request.Host is correct. Honors X-Forwarded-Proto so links generated
// behind a TLS-terminating proxy (Cloud Run) use https. Returns "" when
// the host is unknown so the caller omits the link rather than emit a
// broken relative URL.
func consoleEntryURL(c *gin.Context, entryID string) string {
	host := c.Request.Host
	if host == "" {
		return ""
	}
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/console/entry/%s", scheme, host, entryID)
}

// slackEntryRef renders an entry as a Slack mrkdwn link to its Console
// detail page (falling back to the bare title when no absolute URL can
// be built). The title is escaped for Slack's three reserved characters.
func slackEntryRef(c *gin.Context, entryID, title string) string {
	t := slackEscape(title)
	if url := consoleEntryURL(c, entryID); url != "" {
		return fmt.Sprintf("<%s|%s>", url, t)
	}
	return t
}

// slackAuthorName resolves a display name for a message actor, following
// the same precedence as the Console and Portal (username → name →
// "Anonymous"; see web/{console,portal}/src/utils/user-display.ts). A nil
// user (anonymous portal submission) also reads as "Anonymous".
func slackAuthorName(u *models.User) string {
	if u == nil {
		return "Anonymous"
	}
	ec := api.BuildEntryCreator(*u)
	if ec.Username != "" {
		return slackEscape(ec.Username)
	}
	if ec.Name != "" {
		return slackEscape(ec.Name)
	}
	return "Anonymous"
}

// slackEntryTypeLabel maps the kebab-case EntryType enum to a
// human-readable label for message copy. Untyped entries read as the
// generic "Feedback".
func slackEntryTypeLabel(t models.EntryType) string {
	switch t {
	case models.EntryTypeFeatureRequest:
		return "Feature request"
	case models.EntryTypeBug:
		return "Bug"
	case models.EntryTypeSupport:
		return "Support"
	case models.EntryTypeOther:
		return "Other"
	default:
		return "Feedback"
	}
}

// slackEscape escapes the three characters Slack reserves in message
// text (& < >). Applied to any user-supplied string interpolated into a
// message so a title like "a < b" can't corrupt the mrkdwn.
func slackEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// slackExcerpt collapses whitespace and truncates to a rune-safe length
// for a one-line comment preview.
func slackExcerpt(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= max {
		return slackEscape(s)
	}
	return slackEscape(strings.TrimSpace(string(r[:max]))) + "…"
}
