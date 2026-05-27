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

// SimilarityFinderName is used in logs / error envelopes.
const SimilarityFinderName = "similarity"

// similarityToolName is the forced-tool-call name. Mirrors the
// relation analyzer pattern so the model is constrained to emit the
// JSON shape we expect.
const similarityToolName = "rank_similar_entries"

// similarityDescriptionTruncateLimit caps the new-draft description
// blurb sent to the model. Mirrors the relation analyzer bound.
const similarityDescriptionTruncateLimit = 4000

// similarityCandidateDescTruncateLimit caps the per-candidate blurb
// included in the prompt.
const similarityCandidateDescTruncateLimit = 600

// SimilarityFinderMaxResults is the upper bound on matches the model
// is allowed to return. Five keeps the create-entry similarity panel
// scannable; the LLM is instructed to skip weak matches rather than
// pad to this number.
const SimilarityFinderMaxResults = 5

// similaritySystemPromptHeader is the static lead-in for the system
// message. The candidate list is appended at call time.
const similaritySystemPromptHeader = "You help a user submitting NEW feedback notice when an existing entry already covers the same idea. Given the user's draft (title and optional description) and a list of existing entries, return only the entries that describe the same feature, problem, or request. Two entries can use very different words for the same idea — focus on intent, not phrasing.\n\nReturn at most " + similarityFinderMaxResultsString + " entries, ordered most relevant first. Skip weak matches rather than pad. If nothing matches, return an empty list. Never invent entry IDs that are not in the candidate list below.\n\nCandidate entries:\n"

// similarityFinderMaxResultsString is the string form of
// SimilarityFinderMaxResults so it can sit inside a const string.
const similarityFinderMaxResultsString = "5"

// SimilarityCandidate is the minimal shape the finder needs from each
// existing entry. Built by the caller from models.Entry.
type SimilarityCandidate struct {
	ID          string
	Title       string
	Description string
}

// SimilarityMatch is one ranked match the finder returned.
type SimilarityMatch struct {
	EntryID string
	Reason  string
}

// similarityToolResult is the typed shape of the JSON the model emits
// inside its tool_use block.
type similarityToolResult struct {
	Matches []similarityToolMatch `json:"matches"`
}

type similarityToolMatch struct {
	EntryID string `json:"entryId"`
	Reason  string `json:"reason"`
}

// FindSimilarEntries asks Claude which of the supplied candidates
// describe the same idea as the draft (title + description). Returns
// an ordered list of matches; empty slice means "no similar entries".
//
// Synchronous: this is invoked from a request handler, not a worker
// tick. Cost and latency are bounded by SimilarityFinderMaxResults
// and the candidate-description truncation. Pilot dataset can fit
// comfortably in one prompt; when entries grow into the thousands,
// the caller pre-narrows via text search before passing candidates
// in here.
func FindSimilarEntries(ctx context.Context, snap Snapshot, draftTitle, draftDescription string, candidates []SimilarityCandidate) ([]SimilarityMatch, error) {
	if !snap.Enabled || snap.APIKey == "" {
		return nil, nil
	}
	if strings.TrimSpace(draftTitle) == "" {
		return nil, nil
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	model := snap.Model
	if model == "" {
		model = DefaultModel
	}

	desc := draftDescription
	if len(desc) > similarityDescriptionTruncateLimit {
		desc = desc[:similarityDescriptionTruncateLimit]
	}
	client := anthropic.NewClient(option.WithAPIKey(snap.APIKey))

	userPrompt := fmt.Sprintf("New draft to compare against the candidates:\n\nTitle: %s\n\nDescription: %s", draftTitle, desc)

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: buildSimilaritySystemPrompt(candidates)},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
		Tools: []anthropic.ToolUnionParam{{
			OfTool: &anthropic.ToolParam{
				Name:        similarityToolName,
				Description: anthropic.String("Record the existing entries that describe the same idea as the new draft, ordered most relevant first."),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: buildSimilarityToolSchema(),
					Required:   []string{"matches"},
				},
			},
		}},
		ToolChoice: anthropic.ToolChoiceParamOfTool(similarityToolName),
	})
	if err != nil {
		return nil, err
	}
	for _, block := range msg.Content {
		u := block.AsToolUse()
		if u.Name != similarityToolName {
			continue
		}
		var r similarityToolResult
		if err := json.Unmarshal(u.Input, &r); err != nil {
			return nil, fmt.Errorf("decode tool_use input: %w", err)
		}
		return resolveSimilarityResults(candidates, r), nil
	}
	return nil, fmt.Errorf("model did not emit tool_use for %s", similarityToolName)
}

// resolveSimilarityResults validates and dedupes the model's output.
// Hallucinated IDs (not in the candidate set) are dropped; duplicates
// by entryId are collapsed; the slice is capped at
// SimilarityFinderMaxResults.
func resolveSimilarityResults(candidates []SimilarityCandidate, result similarityToolResult) []SimilarityMatch {
	known := make(map[string]struct{}, len(candidates))
	for _, c := range candidates {
		known[c.ID] = struct{}{}
	}
	seen := map[string]struct{}{}
	out := make([]SimilarityMatch, 0, len(result.Matches))
	for _, m := range result.Matches {
		id := strings.TrimSpace(m.EntryID)
		if id == "" {
			continue
		}
		if _, ok := known[id]; !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, SimilarityMatch{EntryID: id, Reason: strings.TrimSpace(m.Reason)})
		if len(out) >= SimilarityFinderMaxResults {
			break
		}
	}
	return out
}

func buildSimilaritySystemPrompt(candidates []SimilarityCandidate) string {
	var b strings.Builder
	b.WriteString(similaritySystemPromptHeader)
	for _, c := range candidates {
		b.WriteString("- id=")
		b.WriteString(c.ID)
		b.WriteString(" | ")
		b.WriteString(c.Title)
		desc := strings.TrimSpace(c.Description)
		if len(desc) > similarityCandidateDescTruncateLimit {
			desc = desc[:similarityCandidateDescTruncateLimit]
		}
		if desc != "" {
			b.WriteString(" — ")
			b.WriteString(desc)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func buildSimilarityToolSchema() map[string]any {
	return map[string]any{
		"matches": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"entryId": map[string]any{
						"type":        "string",
						"description": "ID of an existing entry that describes the same idea as the new draft. Must be one of the IDs listed in the system prompt verbatim.",
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "One short sentence explaining what the new draft and this existing entry have in common.",
					},
				},
				"required": []string{"entryId", "reason"},
			},
			"description": "Existing entries that describe the same idea, ordered most relevant first. Empty array when none apply.",
		},
	}
}

// CandidatesFromEntries adapts a slice of models.Entry into the
// per-call SimilarityCandidate shape the finder consumes. Caller is
// responsible for filtering visibility (e.g. dropping internal
// entries on the portal surface) before passing in.
func CandidatesFromEntries(entries []models.Entry) []SimilarityCandidate {
	out := make([]SimilarityCandidate, 0, len(entries))
	for _, e := range entries {
		out = append(out, SimilarityCandidate{
			ID:          e.ID,
			Title:       e.Title,
			Description: e.Description,
		})
	}
	return out
}
