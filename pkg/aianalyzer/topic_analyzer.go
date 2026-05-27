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

// TopicAnalyzerName is the registry key + queue job name for the
// topic analyzer.
const TopicAnalyzerName = "topic"

// topicDescriptionTruncateLimit caps the description text sent to
// the LLM. Mirrors the tag analyzer's bound for the same reason.
const topicDescriptionTruncateLimit = 4000

// topicToolName is the forced-tool-call name. The schema is
// permissive: the model returns a single chosen topic title, plus
// an isNew flag and an optional description/color when it wants to
// create a new topic. This keeps the contract single-tool per
// entry — the UI never sees a "no decision" outcome.
const topicToolName = "assign_topic"

// topicSystemPromptHeader is the static lead-in for the system
// message. The dynamic part — the list of existing topics — is
// appended at call time. Topics are areas of the customer product
// (sign up, feed, settings, billing, search, profile…); the model
// is allowed to invent a new topic when no listed topic clearly
// fits.
const topicSystemPromptHeader = "You assign a topic to user feedback for a feedback portal. A topic represents an area of the product the entry is about — for example sign-up, feed, settings, billing, search, profile, notifications. Given the title and description of one entry, pick the single best-fit topic.\n\nIf one of the existing topics clearly fits, return its title verbatim with isNew=false. If none of the existing topics fit, invent a short topic title (1-3 words, Title Case) describing the area and return it with isNew=true. Keep titles concise; avoid duplicating an existing title with a different casing.\n\nExisting topics:\n"

// topicToolResult is the typed shape of the JSON the model emits
// inside its tool_use block. Reason is the per-assignment
// justification surfaced on the Console badge tooltip — same shape
// as category_analyzer's reason. Description/Color carry the
// metadata only used when the model invents a brand-new topic
// (isNew=true).
type topicToolResult struct {
	Topic       string `json:"topic"`
	IsNew       bool   `json:"isNew"`
	Reason      string `json:"reason"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

// defaultAITopicColor is the color stamped on AI-created topics
// when the model does not return one.
const defaultAITopicColor = "#0EA5E9"

// TopicAnalyzer maps an entry's title + description to a single
// topic. Authoritative — the result is written directly to
// Entry.TopicIDs.
//
// Distinct from the tag analyzer: when no existing topic fits, the
// analyzer is allowed to create a brand-new EntryTopic and assign
// it. The settings list surfaces the resulting topic with an
// "AI-created" badge so admins can review and rename it.
//
// Trigger condition is one-shot at creation: the worker only
// claims entries where TopicIDs is empty AND the analysis has
// never run (or previously failed).
type TopicAnalyzer struct {
	snapshot func() Snapshot
}

// NewTopicAnalyzer constructs a TopicAnalyzer backed by the given
// snapshot accessor.
func NewTopicAnalyzer(snapshot func() Snapshot) *TopicAnalyzer {
	return &TopicAnalyzer{snapshot: snapshot}
}

func (a *TopicAnalyzer) Name() string { return TopicAnalyzerName }

// ClaimNext atomically claims the next entry needing topic
// analysis.
func (a *TopicAnalyzer) ClaimNext(do *dataoperations.DataOperations, claimTTL time.Duration) (*models.Entry, error) {
	return do.ClaimNextPendingForTopicAnalysis(claimTTL)
}

func (a *TopicAnalyzer) PendingCount(do *dataoperations.DataOperations, claimTTL time.Duration) (int, error) {
	return do.CountEntriesPendingTopicAnalysis(claimTTL)
}

func (a *TopicAnalyzer) InFlightCount(do *dataoperations.DataOperations, claimTTL time.Duration) (int, error) {
	return do.CountEntriesInFlightTopicAnalysis(claimTTL)
}

// Process runs the LLM call against the already-claimed entry,
// auto-applies the resolved topic ID to Entry.TopicIDs, and
// persists the analysis record. Called by the Worker after a
// successful ClaimNext.
//
// When the model returns isNew=true and the title isn't already
// present (case-insensitive), we insert a fresh EntryTopic with
// Source=ai and assign it. Race with a concurrent admin creating
// the same title is handled by re-checking after the model returns
// — if the title now exists, we use that record's ID and skip the
// insert.
func (a *TopicAnalyzer) Process(ctx context.Context, do *dataoperations.DataOperations, e *models.Entry) error {
	snap := a.snapshot()
	if !snap.Enabled || snap.APIKey == "" {
		return do.SetEntryTopicAnalysis(e.ID, models.EntryTopicAnalysis{})
	}
	model := snap.Model
	if model == "" {
		model = DefaultModel
	}

	topics, err := do.ListEntryTopics()
	if err != nil {
		_ = do.FailEntryTopicAnalysis(e.ID, err.Error())
		return err
	}

	result, err := callClaudeForTopic(ctx, snap.APIKey, model, e.Title, e.Description, topics)
	if err != nil {
		_ = do.FailEntryTopicAnalysis(e.ID, err.Error())
		return err
	}

	chosenID, err := resolveOrCreateTopic(do, topics, result)
	if err != nil {
		_ = do.FailEntryTopicAnalysis(e.ID, err.Error())
		return err
	}

	var assigned []string
	reasons := map[string]string{}
	if chosenID != "" {
		assigned = []string{chosenID}
		reasons[chosenID] = strings.TrimSpace(result.Reason)
	} else {
		assigned = []string{}
	}
	if err := do.SetEntryTopicsFromAnalyzer(e.ID, assigned); err != nil {
		_ = do.FailEntryTopicAnalysis(e.ID, err.Error())
		return err
	}
	// One topic-added activity per AI-assigned topic. Source=ai so the
	// Console badge it without falling back to a synthetic user.
	// Failure here is not fatal to analysis success — log via the
	// failure path on the analysis sub-doc would be misleading (the
	// topic IS assigned). We just continue.
	for _, topicID := range assigned {
		act := models.NewActivity()
		act.EntryID = e.ID
		act.Type = models.ActivityTypeTopicAdded
		act.Source = models.ActivitySourceAI
		act.TargetID = topicID
		_ = do.InsertActivity(act)
	}

	return do.SetEntryTopicAnalysis(e.ID, models.EntryTopicAnalysis{
		Status:                models.AnalysisStatusDone,
		Model:                 model,
		AnalyzedAt:            time.Now().UTC(),
		SuggestedTopicIDs:     assigned,
		SuggestedTopicReasons: reasons,
	})
}

// resolveOrCreateTopic turns the model's emitted result into a
// topic ID. If the title matches an existing topic
// (case-insensitive), that ID wins regardless of the isNew flag —
// guards against the model claiming a new topic while accidentally
// reusing an existing title. Otherwise, when isNew is true and the
// title is non-empty, insert a new topic record and return its ID.
//
// Returns "" with no error when the model's title is blank — the
// analyzer treats that as a clean "no topic" outcome rather than a
// failure, mirroring the category analyzer's behavior.
func resolveOrCreateTopic(do *dataoperations.DataOperations, existing []models.EntryTopic, result topicToolResult) (string, error) {
	title := strings.TrimSpace(result.Topic)
	if title == "" {
		return "", nil
	}
	target := strings.ToLower(title)
	for _, t := range existing {
		if strings.ToLower(t.Title) == target {
			return t.ID, nil
		}
	}
	if !result.IsNew {
		return "", nil
	}
	// Race-tolerant double-check via case-insensitive lookup right
	// before insert — an admin or another worker could have created
	// the same title between ListEntryTopics and now.
	if dup, err := do.FindEntryTopicByTitle(title); err != nil {
		return "", err
	} else if dup != nil {
		return dup.ID, nil
	}
	color := strings.TrimSpace(result.Color)
	if color == "" {
		color = defaultAITopicColor
	}
	created := models.NewEntryTopic()
	created.Title = title
	created.Description = strings.TrimSpace(result.Description)
	created.Color = color
	created.Source = models.EntryTopicSourceAI
	if err := do.InsertEntryTopic(created); err != nil {
		return "", err
	}
	return created.ID, nil
}

// buildTopicSystemPrompt assembles the system message. Static
// header explains the task; the dynamic body lists each configured
// topic's title + description.
func buildTopicSystemPrompt(topics []models.EntryTopic) string {
	var b strings.Builder
	b.WriteString(topicSystemPromptHeader)
	if len(topics) == 0 {
		b.WriteString("(none configured yet — invent a topic title that fits the entry)\n")
		return b.String()
	}
	for _, t := range topics {
		b.WriteString("- ")
		b.WriteString(t.Title)
		if t.Description != "" {
			b.WriteString(" — ")
			b.WriteString(t.Description)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// buildTopicToolSchema produces the per-call tool input schema.
// Open enum: the model may return any title string. Existing
// titles are listed in the system prompt so the model has the full
// picture without the JSON Schema constraining it; this is what
// lets the model invent a new topic when none of the listed ones
// fit.
func buildTopicToolSchema() map[string]any {
	return map[string]any{
		"topic": map[string]any{
			"type":        "string",
			"description": "The title of the topic that best fits this entry. Must be either an existing topic's title verbatim, or a brand-new short title (1-3 words, Title Case) when none of the listed topics fit.",
		},
		"isNew": map[string]any{
			"type":        "boolean",
			"description": "Whether the topic title is a new one not present in the existing list. Set false when reusing an existing title.",
		},
		"reason": map[string]any{
			"type":        "string",
			"description": "Single short sentence (max 160 chars) explaining why this topic fits the entry — surfaced verbatim on the Console badge tooltip. Do not restate the title.",
		},
		"description": map[string]any{
			"type":        "string",
			"description": "Optional one-sentence description for the new topic. Provide only when isNew=true.",
		},
		"color": map[string]any{
			"type":        "string",
			"description": "Optional hex color (#RRGGBB) for the new topic. Provide only when isNew=true; leave empty to use a system default.",
		},
	}
}

// callClaudeForTopic issues one Messages.New request with a forced
// tool call. Returns the parsed result; caller resolves to an ID
// (or creates a new topic) via resolveOrCreateTopic.
func callClaudeForTopic(ctx context.Context, apiKey, model, title, description string, topics []models.EntryTopic) (topicToolResult, error) {
	if len(description) > topicDescriptionTruncateLimit {
		description = description[:topicDescriptionTruncateLimit]
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	userPrompt := fmt.Sprintf("Title: %s\n\nDescription: %s", title, description)

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 256,
		System: []anthropic.TextBlockParam{
			{Text: buildTopicSystemPrompt(topics)},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
		Tools: []anthropic.ToolUnionParam{{
			OfTool: &anthropic.ToolParam{
				Name:        topicToolName,
				Description: anthropic.String("Record the topic that best applies to this feedback entry, creating a new topic if needed."),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: buildTopicToolSchema(),
					Required:   []string{"topic", "isNew", "reason"},
				},
			},
		}},
		ToolChoice: anthropic.ToolChoiceParamOfTool(topicToolName),
	})
	if err != nil {
		return topicToolResult{}, err
	}
	for _, block := range msg.Content {
		u := block.AsToolUse()
		if u.Name != topicToolName {
			continue
		}
		var r topicToolResult
		if err := json.Unmarshal(u.Input, &r); err != nil {
			return topicToolResult{}, fmt.Errorf("decode tool_use input: %w", err)
		}
		return r, nil
	}
	return topicToolResult{}, fmt.Errorf("model did not emit tool_use for %s", topicToolName)
}
