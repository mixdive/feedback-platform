package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// entryTypeCountsResponse is the per-entry-type breakdown rendered
// alongside a parent count on Topics and Releases list pages.
// Total = FeatureRequest + Bug + Support + Other.
type entryTypeCountsResponse struct {
	Total          int64 `json:"total"`
	FeatureRequest int64 `json:"featureRequest"`
	Bug            int64 `json:"bug"`
	Support        int64 `json:"support"`
	Other          int64 `json:"other"`
} //@name EntryTypeCounts

// entryTopicListItem extends the cross-surface topic projection with
// the "entries linked to this topic" count rendered next to each row
// on the top-level Topics page, plus the per-entry-type breakdown so
// the page can render one chip per type without a second round-trip.
type entryTopicListItem struct {
	EntryTopicResponse
	EntryCount       int64                   `json:"entryCount"`
	EntryTypeCounts  entryTypeCountsResponse `json:"entryTypeCounts"`
} //@name ConsoleEntryTopicListItem

// entryTopicListResponse is the envelope for topic lists.
type entryTopicListResponse struct {
	Data []entryTopicListItem `json:"data"`
} //@name EntryTopicList

// ListEntryTopicsHandler returns every entry topic with its entry
// count. The count comes from a single aggregation over the entries
// collection ($unwind topicids → $group) — no per-topic round-trip.
//
//	@ID			console-list-entry-topics
//	@Summary	List entry topics (admin)
//	@Tags		Console
//	@Produce	json
//	@Success	200	{object}	entryTopicListResponse
//	@Router		/api/console/entry-topic [get]
func ListEntryTopicsHandler(do dataoperations.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		topics, err := do.ListEntryTopics()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		typeCounts, err := do.CountEntriesPerTopicByEntryType()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		out := make([]entryTopicListItem, 0, len(topics))
		for _, t := range topics {
			tc := typeCounts[t.ID]
			out = append(out, entryTopicListItem{
				EntryTopicResponse: BuildEntryTopic(t),
				EntryCount:         tc.Total,
				EntryTypeCounts: entryTypeCountsResponse{
					Total:          tc.Total,
					FeatureRequest: tc.FeatureRequest,
					Bug:            tc.Bug,
					Support:        tc.Support,
					Other:          tc.Other,
				},
			})
		}
		response.Success(c, entryTopicListResponse{Data: out})
	}
}
