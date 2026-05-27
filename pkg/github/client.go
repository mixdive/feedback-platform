// Package github is a hairline-thin wrapper around the parts of the
// GitHub REST API Mixdive talks to. v0.1 covers exactly one call —
// create an issue on behalf of an admin — so we hold the surface
// area to that. Anything richer (sync, comments, labels) waits for a
// real customer ask.
//
// We deliberately avoid pulling in the official go-github SDK: it
// brings a large dependency surface and we'd use < 1% of it. Direct
// net/http keeps the binary small and the failure modes legible.
package github

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

// apiBase is the GitHub REST API root. Constant so tests can swap it
// via a package-level override (none today — we'll add it when the
// first test arrives).
const apiBase = "https://api.github.com"

// requestTimeout caps a single GitHub call. 30s is generous —
// real responses land in well under a second; the cap is a guard
// against hung TCP that would otherwise block the user-facing
// Console request indefinitely.
const requestTimeout = 30 * time.Second

// Issue is the slice of GitHub's issue response we actually use.
// Everything else (labels, assignees, body, state) is ignored.
type Issue struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
}

// CreateIssueParams collects every input CreateIssue needs. Owner /
// Repo come from the singleton settings doc; Token is the PAT also
// stored there. Title + Body are the AI-summarized payload built by
// the create-issue handler.
type CreateIssueParams struct {
	Token string
	Owner string
	Repo  string
	Title string
	Body  string
}

// CreateIssue posts a new issue to {owner}/{repo}. Returns the
// created issue's number + html URL. Errors wrap the upstream status
// + body so the Console can surface a useful message ("repo not
// found", "token lacks issue scope") instead of a bare 500.
func CreateIssue(ctx context.Context, p CreateIssueParams) (Issue, error) {
	if p.Token == "" || p.Owner == "" || p.Repo == "" {
		return Issue{}, fmt.Errorf("github: missing token/owner/repo")
	}
	body, err := json.Marshal(map[string]string{
		"title": p.Title,
		"body":  p.Body,
	})
	if err != nil {
		return Issue{}, fmt.Errorf("github: marshal request body: %w", err)
	}
	url := fmt.Sprintf("%s/repos/%s/%s/issues", apiBase, p.Owner, p.Repo)

	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Issue{}, fmt.Errorf("github: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mixdive")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Issue{}, fmt.Errorf("github: post issue: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return Issue{}, fmt.Errorf("github: create issue failed (%s): %s", resp.Status, summariseError(respBody))
	}

	var issue Issue
	if err := json.Unmarshal(respBody, &issue); err != nil {
		return Issue{}, fmt.Errorf("github: decode response: %w", err)
	}
	if issue.Number == 0 || issue.HTMLURL == "" {
		return Issue{}, fmt.Errorf("github: response missing number/html_url")
	}
	return issue, nil
}

// VerifyRepoAccess GETs the repo metadata to confirm a PAT can read
// the target repo. Called from the connect-integration handler before
// the token + owner/repo are persisted, so a typo'd repo path or a
// scope-light PAT fails the Save click immediately instead of
// surfacing as a confusing "Create issue" error later. The check is
// read-only — we never write a probe issue.
func VerifyRepoAccess(ctx context.Context, token, owner, repo string) error {
	if token == "" || owner == "" || repo == "" {
		return fmt.Errorf("github: missing token/owner/repo")
	}
	url := fmt.Sprintf("%s/repos/%s/%s", apiBase, owner, repo)

	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("github: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Mixdive")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("github: get repo: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github: verify repo failed (%s): %s", resp.Status, summariseError(respBody))
	}
	return nil
}

// summariseError extracts GitHub's "message" field from an error
// response body so the caller can render it inline. Falls back to
// the raw body when the JSON doesn't parse.
func summariseError(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var e struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &e); err == nil && e.Message != "" {
		return strings.TrimSpace(e.Message)
	}
	return strings.TrimSpace(string(body))
}
