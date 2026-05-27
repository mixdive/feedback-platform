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

// EntryTypeAnalyzerName is the registry key + queue job name for the
// entry-type analyzer. Stable string — referenced from the Worker's
// stats payload (PendingByName / InFlightByName are keyed by analyzer
// name) and from any future endpoint that needs to address this
// analyzer.
const EntryTypeAnalyzerName = "entry-type"

// descriptionTruncateLimit caps the description text sent to the LLM.
// 4000 chars is well beyond a typical feedback entry; bounding it
// prevents pathological inputs from running up token spend.
const descriptionTruncateLimit = 4000

// entryTypeToolName is the forced-tool-call name. Must match the Tools
// entry below and the ToolChoiceParamOfTool argument so the model is
// constrained to emit exactly this tool's JSON schema.
const entryTypeToolName = "suggest_entry_type"

// entryTypeSystemPromptHeader is the static lead-in for the system
// message. The dynamic part — the actual list of available entry
// types with their descriptions — is appended at call time.
const entryTypeSystemPromptHeader = "You classify user feedback for a feedback portal. Given the title and description of one entry, decide which single entry type it best fits.\n\nAvailable entry types:\n"

// entryTypeToolResult is the typed shape of the JSON the model emits
// inside its tool_use block. Reason is a single sentence the model
// writes alongside its pick — surfaced on the Console badge tooltip
// so admins can see why the AI made the call.
type entryTypeToolResult struct {
	EntryType string `json:"entryType"`
	Reason    string `json:"reason"`
}

// EntryTypeAnalyzer asks Claude to map an entry's title + description
// to one of the hardcoded EntryType enum values ("feature-request",
// "bug", "support", "other"). Result is advisory — the analyzer
// writes the suggested enum value to
// Entry.EntryTypeAnalysis.SuggestedEntryTypeID, never to
// Entry.EntryType. The Console UI surfaces the suggestion with an
// Apply button.
type EntryTypeAnalyzer struct {
	snapshot func() Snapshot
}

// entryTypeChoice carries the metadata the analyzer sends to the LLM
// for each available entry type — value (the enum literal we want
// back), a human-readable title, and a one-line description.
type entryTypeChoice struct {
	Value       models.EntryType
	Title       string
	Description string
}

// availableEntryTypes is the static list the analyzer offers to the
// LLM. Title and description are kept here (not on EntryTypeResponse)
// because the wire shape doesn't include a description and the
// analyzer wants one in the prompt.
var availableEntryTypes = []entryTypeChoice{
	{Value: models.EntryTypeFeatureRequest, Title: "Feature Request", Description: "Ideas and improvements you'd like to see."},
	{Value: models.EntryTypeBug, Title: "Bug", Description: "Something isn't working the way it should."},
	{Value: models.EntryTypeSupport, Title: "Support", Description: "Help, troubleshooting, or a how-do-I question — not a defect."},
	{Value: models.EntryTypeOther, Title: "Other", Description: "Anything that doesn't fit the types above."},
}

// NewEntryTypeAnalyzer constructs an EntryTypeAnalyzer backed by the
// given snapshot accessor. The snapshot indirection means the
// analyzer does not hold a Worker reference (no import cycle, easier
// to test).
func NewEntryTypeAnalyzer(snapshot func() Snapshot) *EntryTypeAnalyzer {
	return &EntryTypeAnalyzer{snapshot: snapshot}
}

func (a *EntryTypeAnalyzer) Name() string { return EntryTypeAnalyzerName }

// ClaimNext atomically claims the next entry needing entry-type
// analysis run. The Worker calls this once per tick.
func (a *EntryTypeAnalyzer) ClaimNext(do *dataoperations.DataOperations, claimTTL time.Duration) (*models.Entry, error) {
	return do.ClaimNextPendingForEntryTypeAnalysis(claimTTL)
}

// PendingCount delegates to the matching DB query so the Worker's
// stats surface includes a per-analyzer line.
func (a *EntryTypeAnalyzer) PendingCount(do *dataoperations.DataOperations, claimTTL time.Duration) (int, error) {
	return do.CountEntriesPendingEntryTypeAnalysis(claimTTL)
}

func (a *EntryTypeAnalyzer) InFlightCount(do *dataoperations.DataOperations, claimTTL time.Duration) (int, error) {
	return do.CountEntriesInFlightEntryTypeAnalysis(claimTTL)
}

// Process runs the LLM call against the already-claimed entry and
// persists the final state. Called by the Worker after a successful
// ClaimNext. The success path writes a fresh EntryTypeAnalysis (which
// has zero ClaimedAt — releases the claim); the failure path stamps
// status=failed and clears ClaimedAt so the next tick can retry.
//
// The entry-type list is hardcoded (see availableEntryTypes), so
// there's no Mongo lookup and no admin-managed schema to drift.
func (a *EntryTypeAnalyzer) Process(ctx context.Context, do *dataoperations.DataOperations, e *models.Entry) error {
	snap := a.snapshot()
	if !snap.Enabled || snap.APIKey == "" {
		// Race: AI was flipped off between the worker's snapshot read
		// and ours. Release the claim by writing an empty analysis
		// (ClaimedAt zero, status default ""), so the entry is left
		// alone until AI is re-enabled.
		return do.SetEntryTypeAnalysis(e.ID, models.EntryTypeAnalysis{})
	}
	model := snap.Model
	if model == "" {
		model = DefaultModel
	}

	result, err := callClaudeForEntryType(ctx, snap.APIKey, model, e.Title, e.Description)
	if err != nil {
		_ = do.FailEntryTypeAnalysis(e.ID, err.Error())
		return err
	}

	suggested := resolveEntryTypeValue(result.EntryType)
	reason := strings.TrimSpace(result.Reason)
	if suggested == "" {
		reason = ""
	}

	return do.SetEntryTypeAnalysis(e.ID, models.EntryTypeAnalysis{
		Status:                   models.AnalysisStatusDone,
		Model:                    model,
		AnalyzedAt:               time.Now().UTC(),
		SuggestedEntryTypeID:     string(suggested),
		SuggestedEntryTypeReason: reason,
	})
}

// buildEntryTypeToolSchema produces the per-call tool input schema.
// The enum lists the hardcoded EntryType titles — frozen across runs
// because the set itself is frozen.
func buildEntryTypeToolSchema() map[string]any {
	titles := make([]string, len(availableEntryTypes))
	for i, c := range availableEntryTypes {
		titles[i] = c.Title
	}
	return map[string]any{
		"entryType": map[string]any{
			"type":        "string",
			"enum":        titles,
			"description": "The single best-fit entry type title for this feedback. Must match one of the listed enum values verbatim.",
		},
		"reason": map[string]any{
			"type":        "string",
			"description": "A single short sentence (max 160 chars) explaining why this entry type fits. Surfaced verbatim on the Console badge tooltip — keep it crisp, evidence-based, and free of restating the title.",
		},
	}
}

// buildEntryTypeSystemPrompt assembles the system message. The static
// header explains the task; the dynamic body lists each available
// type's title + description.
func buildEntryTypeSystemPrompt() string {
	var b strings.Builder
	b.WriteString(entryTypeSystemPromptHeader)
	for _, c := range availableEntryTypes {
		b.WriteString("- ")
		b.WriteString(c.Title)
		if c.Description != "" {
			b.WriteString(" — ")
			b.WriteString(c.Description)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// callClaudeForEntryType issues one Messages.New request with a
// forced tool call. Returns the parsed tool result (entry-type title
// + a one-line justification).
func callClaudeForEntryType(ctx context.Context, apiKey, model, title, description string) (entryTypeToolResult, error) {
	if len(description) > descriptionTruncateLimit {
		description = description[:descriptionTruncateLimit]
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	userPrompt := fmt.Sprintf("Title: %s\n\nDescription: %s", title, description)

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 256,
		System: []anthropic.TextBlockParam{
			{Text: buildEntryTypeSystemPrompt()},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
		Tools: []anthropic.ToolUnionParam{{
			OfTool: &anthropic.ToolParam{
				Name:        entryTypeToolName,
				Description: anthropic.String("Record the suggested entry type for this feedback."),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: buildEntryTypeToolSchema(),
					Required:   []string{"entryType", "reason"},
				},
			},
		}},
		ToolChoice: anthropic.ToolChoiceParamOfTool(entryTypeToolName),
	})
	if err != nil {
		return entryTypeToolResult{}, err
	}
	for _, block := range msg.Content {
		u := block.AsToolUse()
		if u.Name != entryTypeToolName {
			continue
		}
		var r entryTypeToolResult
		if err := json.Unmarshal(u.Input, &r); err != nil {
			return entryTypeToolResult{}, fmt.Errorf("decode tool_use input: %w", err)
		}
		return r, nil
	}
	return entryTypeToolResult{}, fmt.Errorf("model did not emit tool_use for %s", entryTypeToolName)
}

// resolveEntryTypeValue maps a Title returned by the model back to
// its EntryType enum value (case-insensitive, whitespace-trimmed).
// Returns "" if no match — the caller persists the empty value
// rather than guessing.
func resolveEntryTypeValue(name string) models.EntryType {
	target := strings.ToLower(strings.TrimSpace(name))
	for _, c := range availableEntryTypes {
		if strings.ToLower(c.Title) == target {
			return c.Value
		}
	}
	return ""
}
