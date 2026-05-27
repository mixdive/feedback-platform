package aianalyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// RelationAnalyzerName is the registry key + queue job name for the
// relation analyzer.
const RelationAnalyzerName = "relation"

// relationDescriptionTruncateLimit caps the new-entry description text
// sent to the LLM. Mirrors the tag/topic analyzer bound.
const relationDescriptionTruncateLimit = 4000

// candidateDescriptionTruncateLimit caps the per-candidate description
// blurb included in the prompt. Pilot entries are short, but a single
// pathological description could otherwise dominate the token budget.
const candidateDescriptionTruncateLimit = 600

// relationToolName is the forced-tool-call name. Must match the Tools
// entry below and the ToolChoiceParamOfTool argument so the model is
// constrained to emit exactly this tool's JSON schema.
const relationToolName = "suggest_relations"

// relationSystemPromptHeader is the static lead-in for the system
// message. The dynamic part — the candidate list — is appended at
// call time.
const relationSystemPromptHeader = "You decide whether a newly-created feedback entry is related to any existing entries in a feedback portal. Two kinds of links are recognised:\n\n- duplicate: the new entry describes the same problem, request or topic as the existing one. Use sparingly — only when the two entries clearly overlap.\n- related: the entries touch the same area or concept but are not the same. Use as a catch-all for any non-duplicate similarity.\n\nReturn only entries that meaningfully connect. Do not link entries that are merely about the same product. If no existing entry connects, return an empty list. Never invent entry IDs that are not in the candidate list below.\n\nCandidate entries:\n"

// relationToolResult is the typed shape of the JSON the model emits
// inside its tool_use block.
type relationToolResult struct {
	Relations []relationToolPair `json:"relations"`
}

// relationToolPair is one (entryId, type, reason) item from the model.
// Type is validated client-side; an unknown or missing type falls back
// to EntryRelationTypeRelated rather than dropping the match. Reason
// is the per-link justification surfaced on the Console badge tooltip.
type relationToolPair struct {
	EntryID string `json:"entryId"`
	Type    string `json:"type"`
	Reason  string `json:"reason"`
}

// RelationAnalyzer asks Claude to compare a freshly-created entry
// against every existing entry and emit a list of (peerEntryID, type)
// pairs the two are linked by. Authoritative — the result is written
// directly to Entry.Relations and mirrored onto every peer.
//
// Trigger condition is one-shot at creation: the worker only claims
// entries where Relations is empty AND the analysis has never run (or
// previously failed). An admin clearing all relations on an existing
// entry does NOT re-trigger the analyzer.
//
// Candidate pool in v0.1 is the entire entries collection minus the
// new entry. Pilot dataset is small enough to enumerate inline. When
// vector search lands in v0.2 this becomes a similarity-narrowed
// subset.
type RelationAnalyzer struct {
	snapshot func() Snapshot
}

// NewRelationAnalyzer constructs a RelationAnalyzer backed by the
// given snapshot accessor.
func NewRelationAnalyzer(snapshot func() Snapshot) *RelationAnalyzer {
	return &RelationAnalyzer{snapshot: snapshot}
}

func (a *RelationAnalyzer) Name() string { return RelationAnalyzerName }

func (a *RelationAnalyzer) ClaimNext(do *dataoperations.DataOperations, claimTTL time.Duration) (*models.Entry, error) {
	return do.ClaimNextPendingForRelationAnalysis(claimTTL)
}

func (a *RelationAnalyzer) PendingCount(do *dataoperations.DataOperations, claimTTL time.Duration) (int, error) {
	return do.CountEntriesPendingRelationAnalysis(claimTTL)
}

func (a *RelationAnalyzer) InFlightCount(do *dataoperations.DataOperations, claimTTL time.Duration) (int, error) {
	return do.CountEntriesInFlightRelationAnalysis(claimTTL)
}

// Process runs the LLM call against the already-claimed entry,
// auto-applies the resolved relation pairs (mirroring onto every peer),
// and persists the analysis record. Called by the Worker after a
// successful ClaimNext.
//
// Order matters: SetEntryRelationsFromAnalyzer first so the relations
// land before SetEntryRelationAnalysis flips status to done. If the
// second write crashes, the entry has correct relations and a
// non-final analysis status — the next tick won't re-claim it.
func (a *RelationAnalyzer) Process(ctx context.Context, do *dataoperations.DataOperations, e *models.Entry) error {
	snap := a.snapshot()
	if !snap.Enabled || snap.APIKey == "" {
		return do.SetEntryRelationAnalysis(e.ID, models.EntryRelationAnalysis{})
	}
	model := snap.Model
	if model == "" {
		model = DefaultModel
	}

	candidates, err := do.ListEntriesForRelationAnalysis(e.ID)
	if err != nil {
		_ = do.FailEntryRelationAnalysis(e.ID, err.Error())
		return err
	}
	if len(candidates) == 0 {
		// First entry in the system. Nothing to compare against —
		// stamp done with no suggestions so the entry stops cycling
		// through the queue.
		return do.SetEntryRelationAnalysis(e.ID, models.EntryRelationAnalysis{
			Status:     models.AnalysisStatusDone,
			Model:      model,
			AnalyzedAt: time.Now().UTC(),
		})
	}

	result, err := callClaudeForRelations(ctx, snap.APIKey, model, *e, candidates)
	if err != nil {
		_ = do.FailEntryRelationAnalysis(e.ID, err.Error())
		return err
	}

	suggestedRels, reasons := resolveRelationResults(candidates, result)

	if err := do.SetEntryRelationsFromAnalyzer(e.ID, suggestedRels); err != nil {
		_ = do.FailEntryRelationAnalysis(e.ID, err.Error())
		return err
	}
	// Mirror activity rows: relation-added on both the source and the
	// peer entry so each timeline shows the link. ToValue carries the
	// relation type so the Console can render "Related to X" vs
	// "Duplicate of X" without a second lookup.
	for _, r := range suggestedRels {
		src := models.NewActivity()
		src.EntryID = e.ID
		src.Type = models.ActivityTypeRelationAdded
		src.Source = models.ActivitySourceAI
		src.ToValue = string(r.Type)
		src.TargetID = r.EntryID
		_ = do.InsertActivity(src)

		peer := models.NewActivity()
		peer.EntryID = r.EntryID
		peer.Type = models.ActivityTypeRelationAdded
		peer.Source = models.ActivitySourceAI
		peer.ToValue = string(r.Type)
		peer.TargetID = e.ID
		_ = do.InsertActivity(peer)
	}

	return do.SetEntryRelationAnalysis(e.ID, models.EntryRelationAnalysis{
		Status:                   models.AnalysisStatusDone,
		Model:                    model,
		AnalyzedAt:               time.Now().UTC(),
		SuggestedRelations:       suggestedRels,
		SuggestedRelationReasons: reasons,
	})
}

// resolveRelationResults validates and dedupes the model's output.
// Pairs whose entryId is not in the candidate set are silently
// dropped (defends against the model hallucinating IDs even though
// the prompt enumerates them); duplicates by entryId are collapsed —
// the FIRST pair's reason wins.
//
// Type validation falls back to EntryRelationTypeRelated when the
// model returns an unknown or empty value. This is the chosen
// no-fit-fallback per the v0.1 design — keep the match, don't drop
// it.
//
// Returns both the cleaned relation list and the reasons map keyed
// by peer entry ID, ready for persistence on the analysis sub-struct.
func resolveRelationResults(candidates []models.Entry, result relationToolResult) ([]models.EntryRelation, map[string]string) {
	known := make(map[string]struct{}, len(candidates))
	for _, c := range candidates {
		known[c.ID] = struct{}{}
	}
	seen := map[string]struct{}{}
	out := make([]models.EntryRelation, 0, len(result.Relations))
	reasons := map[string]string{}
	for _, p := range result.Relations {
		id := strings.TrimSpace(p.EntryID)
		if id == "" {
			continue
		}
		if _, ok := known[id]; !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		relType := models.EntryRelationType(strings.TrimSpace(p.Type))
		if !models.IsValidEntryRelationType(relType) {
			relType = models.EntryRelationTypeRelated
		}
		seen[id] = struct{}{}
		out = append(out, models.EntryRelation{EntryID: id, Type: relType})
		reasons[id] = strings.TrimSpace(p.Reason)
	}
	return out, reasons
}

// buildRelationSystemPrompt assembles the system message. Each
// candidate is rendered on one line with its ID, title, and a
// truncated description blurb. The model's tool output references
// entries by these exact IDs.
func buildRelationSystemPrompt(candidates []models.Entry) string {
	var b strings.Builder
	b.WriteString(relationSystemPromptHeader)
	for _, c := range candidates {
		b.WriteString("- id=")
		b.WriteString(c.ID)
		b.WriteString(" | ")
		b.WriteString(c.Title)
		desc := strings.TrimSpace(c.Description)
		if len(desc) > candidateDescriptionTruncateLimit {
			desc = desc[:candidateDescriptionTruncateLimit]
		}
		if desc != "" {
			b.WriteString(" — ")
			b.WriteString(desc)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// buildRelationToolSchema produces the per-call tool input schema. The
// type field is constrained to the kebab-case enum; entryId is left as
// a free string so the prompt's candidate list can carry an arbitrary
// number of IDs without ballooning the JSON Schema. Hallucinated IDs
// are filtered out by resolveRelationResults.
func buildRelationToolSchema() map[string]any {
	return map[string]any{
		"relations": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"entryId": map[string]any{
						"type":        "string",
						"description": "ID of the existing entry that connects to the new one. Must be one of the IDs listed in the system prompt verbatim.",
					},
					"type": map[string]any{
						"type":        "string",
						"enum":        []string{string(models.EntryRelationTypeDuplicate), string(models.EntryRelationTypeRelated)},
						"description": "How the entries connect. Use 'duplicate' only when the two entries clearly describe the same problem or request; otherwise 'related'.",
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "Single short sentence (max 200 chars) explaining why these two entries are linked — surfaced verbatim on the Console badge tooltip. Reference the concrete overlap (same screen, same workflow, same root cause) rather than restating titles.",
					},
				},
				"required": []string{"entryId", "type", "reason"},
			},
			"uniqueItems": true,
			"description": "List of (entryId, type) pairs naming every existing entry the new entry meaningfully connects to. Empty array when none apply.",
		},
	}
}

// callClaudeForRelations issues one Messages.New request with a
// forced tool call. Returns the parsed result; caller validates and
// resolves to EntryRelation pairs via resolveRelationResults.
func callClaudeForRelations(ctx context.Context, apiKey, model string, entry models.Entry, candidates []models.Entry) (relationToolResult, error) {
	description := entry.Description
	if len(description) > relationDescriptionTruncateLimit {
		description = description[:relationDescriptionTruncateLimit]
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	userPrompt := fmt.Sprintf("New entry to analyse:\n\nTitle: %s\n\nDescription: %s", entry.Title, description)

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: buildRelationSystemPrompt(candidates)},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
		Tools: []anthropic.ToolUnionParam{{
			OfTool: &anthropic.ToolParam{
				Name:        relationToolName,
				Description: anthropic.String("Record the list of existing entries that connect to the new one, with the kind of connection."),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: buildRelationToolSchema(),
					Required:   []string{"relations"},
				},
			},
		}},
		ToolChoice: anthropic.ToolChoiceParamOfTool(relationToolName),
	})
	if err != nil {
		return relationToolResult{}, err
	}
	for _, block := range msg.Content {
		u := block.AsToolUse()
		if u.Name != relationToolName {
			continue
		}
		var r relationToolResult
		if err := json.Unmarshal(u.Input, &r); err != nil {
			return relationToolResult{}, fmt.Errorf("decode tool_use input: %w", err)
		}
		return r, nil
	}
	return relationToolResult{}, fmt.Errorf("model did not emit tool_use for %s", relationToolName)
}
