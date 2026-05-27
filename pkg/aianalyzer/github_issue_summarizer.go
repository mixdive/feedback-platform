package aianalyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/mixdive/feedback-platform/models"
)

// GitHubIssueSummarizerName is used in logs / error envelopes.
const GitHubIssueSummarizerName = "github-issue-summarizer"

// githubIssueToolName is the forced-tool-call name. Mirrors the rest
// of the analyzer family: we want JSON-shaped output, not free text.
const githubIssueToolName = "draft_github_issue"

// githubIssueDescriptionTruncateLimit caps how much of the entry
// description we send to the model. Mixdive entries can run long
// (especially bug repros with logs pasted in); 8 KB is plenty for a
// summary task and keeps the prompt cheap.
const githubIssueDescriptionTruncateLimit = 8000

// GitHubIssueDraft is the AI's draft of a GitHub issue. Title is a
// concise, GitHub-style sentence ("Add dark mode to the dashboard");
// Body is markdown structured to fit GitHub's issue conventions for
// the source entry type (Problem/Proposed solution for feature
// requests, Steps to reproduce/Expected/Actual for bugs).
type GitHubIssueDraft struct {
	Title string
	Body  string
}

// githubIssueToolResult is the typed shape the model emits inside the
// tool_use block.
type githubIssueToolResult struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// SummarizeGitHubIssue asks Claude to produce a GitHub-issue-shaped
// title + body from a Mixdive entry. The caller is responsible for
// gating on entry type (only feature-request and bug enter here in
// v0.1); the prompt branches on the type so the body structure
// matches GitHub conventions for that kind of issue.
//
// Synchronous: invoked from a request handler when the admin clicks
// "Create GitHub issue". On AI-disabled deployments or empty key,
// returns a zero-value draft and ok=false so the caller can fall
// back to a verbatim title/description (or refuse the click).
func SummarizeGitHubIssue(ctx context.Context, snap Snapshot, entry models.Entry) (GitHubIssueDraft, bool, error) {
	if !snap.Enabled || snap.APIKey == "" {
		return GitHubIssueDraft{}, false, nil
	}
	if strings.TrimSpace(entry.Title) == "" {
		return GitHubIssueDraft{}, false, fmt.Errorf("entry title is empty")
	}
	model := snap.Model
	if model == "" {
		model = DefaultModel
	}

	desc := entry.Description
	if len(desc) > githubIssueDescriptionTruncateLimit {
		desc = desc[:githubIssueDescriptionTruncateLimit]
	}

	client := anthropic.NewClient(option.WithAPIKey(snap.APIKey))

	userPrompt := fmt.Sprintf(
		"Mixdive entry to publish as a GitHub issue.\n\nType: %s\n\nTitle: %s\n\nDescription:\n%s",
		describeEntryType(entry.EntryType),
		entry.Title,
		desc,
	)

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 1500,
		System: []anthropic.TextBlockParam{
			{Text: buildGitHubIssueSystemPrompt(entry.EntryType)},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
		Tools: []anthropic.ToolUnionParam{{
			OfTool: &anthropic.ToolParam{
				Name:        githubIssueToolName,
				Description: anthropic.String("Record the GitHub issue title and markdown body to open from this Mixdive entry."),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: buildGitHubIssueToolSchema(),
					Required:   []string{"title", "body"},
				},
			},
		}},
		ToolChoice: anthropic.ToolChoiceParamOfTool(githubIssueToolName),
	})
	if err != nil {
		return GitHubIssueDraft{}, false, err
	}

	for _, block := range msg.Content {
		u := block.AsToolUse()
		if u.Name != githubIssueToolName {
			continue
		}
		var r githubIssueToolResult
		if err := json.Unmarshal(u.Input, &r); err != nil {
			return GitHubIssueDraft{}, false, fmt.Errorf("decode tool_use input: %w", err)
		}
		title := strings.TrimSpace(r.Title)
		body := strings.TrimRight(r.Body, " \t\r\n")
		if title == "" {
			// Fall back to the entry title rather than fail the user
			// click — they clicked publish, not "wait for AI".
			title = entry.Title
		}
		return GitHubIssueDraft{Title: title, Body: body}, true, nil
	}
	return GitHubIssueDraft{}, false, fmt.Errorf("model did not emit tool_use for %s", githubIssueToolName)
}

// describeEntryType renders the entry's type in human prose so the
// system prompt can branch without leaking the raw kebab-case enum.
func describeEntryType(t models.EntryType) string {
	switch t {
	case models.EntryTypeFeatureRequest:
		return "Feature request"
	case models.EntryTypeBug:
		return "Bug report"
	case models.EntryTypeSupport:
		return "Support request"
	default:
		return "Feedback"
	}
}

// buildGitHubIssueSystemPrompt switches the expected body structure
// based on the source entry type. Feature requests use a
// Problem/Proposed solution shape; bugs use a Steps-to-reproduce
// shape; anything else gets a generic Summary/Details body. The
// model is told to preserve the customer's voice — this is feedback,
// not marketing copy.
func buildGitHubIssueSystemPrompt(t models.EntryType) string {
	var bodyShape string
	switch t {
	case models.EntryTypeBug:
		bodyShape = `## Summary
One or two sentences describing the bug from the customer's perspective.

## Steps to reproduce
Numbered list extracted from the entry. If the entry doesn't give explicit steps, infer the most likely sequence and prefix the list with "(inferred from customer report)".

## Expected behavior
What the customer expected to happen.

## Actual behavior
What actually happened. Include any error messages or screenshots referenced in the description.

## Customer context
Any environment details (browser, OS, account tier, version) the customer mentioned. Skip the section entirely if none were provided.`
	case models.EntryTypeFeatureRequest:
		bodyShape = `## Problem
What is the customer trying to do, and what's blocking or annoying them today?

## Proposed solution
The behavior the customer is asking for, written as the desired user experience.

## Context
Why this matters to the customer — frequency, impact, workarounds they've tried. Include only what the entry actually says.`
	default:
		bodyShape = `## Summary
One concise paragraph describing what the customer is asking for.

## Details
Anything else from the entry that an engineer triaging this issue should know.`
	}

	return "You are drafting a GitHub issue on behalf of a product team, summarizing a single customer feedback entry collected in Mixdive. Produce a concise, GitHub-style title (under 90 characters, no trailing period) and a markdown body that follows this exact section structure:\n\n" +
		bodyShape +
		"\n\nRules:\n- Preserve the customer's intent and key facts; do not invent details.\n- Drop boilerplate the customer copy-pasted from a template.\n- Stay neutral and engineering-focused; this is not marketing copy.\n- Do not include the entry ID, vote count, status, or any Mixdive-internal metadata in the body — the caller appends its own footer.\n- Markdown only; no HTML."
}

func buildGitHubIssueToolSchema() map[string]any {
	return map[string]any{
		"title": map[string]any{
			"type":        "string",
			"description": "Concise GitHub-style issue title summarizing the customer's feedback. Under 90 characters, no trailing period.",
		},
		"body": map[string]any{
			"type":        "string",
			"description": "Markdown issue body following the section structure described in the system prompt. Do not include a footer or Mixdive metadata.",
		},
	}
}
