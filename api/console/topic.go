package console

import (
	"errors"
	"fmt"

	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// EntryTopicResponse is the console-only projection of a topic
// rendered next to an entry record and on the Topics settings
// screen. Topics are console-only metadata so this lives in the
// console package rather than api/shared.go.
//
// Source surfaces who created the topic — admin-managed entries
// versus AI-created ones get a small badge in the settings list.
type EntryTopicResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
	SortOrder   int    `json:"sortOrder"`
	Source      string `json:"source"`
} //@name EntryTopic

// BuildEntryTopic projects the persisted topic into wire shape.
func BuildEntryTopic(t models.EntryTopic) EntryTopicResponse {
	source := string(t.Source)
	if source == "" {
		source = string(models.EntryTopicSourceAdmin)
	}
	return EntryTopicResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Color:       t.Color,
		SortOrder:   t.SortOrder,
		Source:      source,
	}
}

// LoadEntryTopics batch-resolves the unique non-empty IDs in
// topicIDs into an EntryTopicResponse map keyed by topic ID.
// Missing topics are absent from the map — the caller silently
// drops unknown IDs.
func LoadEntryTopics(do dataoperations.Store, topicIDs []string) (map[string]EntryTopicResponse, error) {
	uniq := map[string]struct{}{}
	for _, id := range topicIDs {
		if id == "" {
			continue
		}
		uniq[id] = struct{}{}
	}
	out := map[string]EntryTopicResponse{}
	if len(uniq) == 0 {
		return out, nil
	}
	all, err := do.ListEntryTopics()
	if err != nil {
		return nil, err
	}
	for _, t := range all {
		if _, ok := uniq[t.ID]; ok {
			out[t.ID] = BuildEntryTopic(t)
		}
	}
	return out, nil
}

// projectEntryTopics resolves an entry's TopicIDs against the
// loaded map and returns a slice in the same order as the IDs on
// the entry. IDs missing from the map are silently dropped.
func projectEntryTopics(topicIDs []string, topics map[string]EntryTopicResponse) []EntryTopicResponse {
	out := make([]EntryTopicResponse, 0, len(topicIDs))
	for _, id := range topicIDs {
		if id == "" {
			continue
		}
		if t, ok := topics[id]; ok {
			out = append(out, t)
		}
	}
	return out
}

// resolveTopicIDs validates a topicIds slice arriving on an entry
// create or update request. nil/empty input is allowed — the entry
// simply has no topics. Every non-empty ID must reference an
// existing topic; duplicates are de-duplicated. We return an error
// on the first unknown ID so the handler maps it to a 400.
func resolveTopicIDs(do dataoperations.Store, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}
	all, err := do.ListEntryTopics()
	if err != nil {
		return nil, err
	}
	known := make(map[string]struct{}, len(all))
	for _, t := range all {
		known[t.ID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		if _, ok := known[id]; !ok {
			return nil, fmt.Errorf("Unknown entry topic: %s.", id)
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

// findTopicOrNotFound is the pattern used by the update/delete
// handlers to look up an existing topic. Returns (nil, nil) on a
// clean not-found.
func findTopicOrNotFound(do dataoperations.Store, id string) (*models.EntryTopic, error) {
	if id == "" {
		return nil, errors.New("Topic id is required.")
	}
	return do.FindEntryTopicByID(id)
}
