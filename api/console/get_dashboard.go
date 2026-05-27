package console

import (
	"github.com/gin-gonic/gin"

	"github.com/mixdive/feedback-platform/api/response"
	"github.com/mixdive/feedback-platform/dataoperations"
)

// dashboardEntryRowResponse is the per-entry projection rendered in
// the Top-voted / Recent cards on the Console Dashboard.
type dashboardEntryRowResponse struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	EntryType    string `json:"entryType,omitempty"`
	Status       string `json:"status,omitempty"`
	VoteCount    int    `json:"voteCount"`
	CommentCount int    `json:"commentCount"`
	IsInternal   bool   `json:"isInternal"`
	CreatedAt    string `json:"createdAt"`
} //@name ConsoleDashboardEntryRow

// dashboardDayBucketResponse is one day's submission count for the
// 30-day trend chart on the Dashboard.
type dashboardDayBucketResponse struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
} //@name ConsoleDashboardDayBucket

// dashboardResponse is the aggregated picture rendered on the Console
// Dashboard page. Numbers are recomputed on every request — there is
// no cached snapshot.
type dashboardResponse struct {
	TotalEntries     int64                        `json:"totalEntries"`
	OpenEntries      int64                        `json:"openEntries"`
	ClosedEntries    int64                        `json:"closedEntries"`
	InternalEntries  int64                        `json:"internalEntries"`
	PublicEntries    int64                        `json:"publicEntries"`
	TotalVotes       int64                        `json:"totalVotes"`
	TotalComments    int64                        `json:"totalComments"`
	TotalUsers       int64                        `json:"totalUsers"`
	TotalTopics      int64                        `json:"totalTopics"`
	TotalReleases    int64                        `json:"totalReleases"`
	EntriesByType    map[string]int64             `json:"entriesByType"`
	EntriesByStatus  map[string]int64             `json:"entriesByStatus"`
	EntriesPerDay    []dashboardDayBucketResponse `json:"entriesPerDay"`
	TopEntriesByVote []dashboardEntryRowResponse  `json:"topEntriesByVote"`
	RecentEntries    []dashboardEntryRowResponse  `json:"recentEntries"`
} //@name ConsoleDashboard

// GetDashboardHandler returns the aggregated stats rendered on the
// Console Dashboard page.
//
//	@ID			console-get-dashboard
//	@Summary	Get dashboard stats (admin/editor)
//	@Tags		Console
//	@Produce	json
//	@Success	200	{object}	dashboardResponse
//	@Router		/api/console/dashboard [get]
func GetDashboardHandler(do *dataoperations.DataOperations) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := do.GetDashboardStats()
		if err != nil {
			response.SystemError(c, err)
			return
		}
		days := make([]dashboardDayBucketResponse, 0, len(stats.EntriesPerDay))
		for _, d := range stats.EntriesPerDay {
			days = append(days, dashboardDayBucketResponse{Date: d.Date, Count: d.Count})
		}
		topRows := make([]dashboardEntryRowResponse, 0, len(stats.TopEntriesByVote))
		for _, e := range stats.TopEntriesByVote {
			topRows = append(topRows, dashboardEntryRowResponse{
				ID:           e.ID,
				Title:        e.Title,
				EntryType:    e.EntryType,
				Status:       e.Status,
				VoteCount:    e.VoteCount,
				CommentCount: e.CommentCount,
				IsInternal:   e.IsInternal,
				CreatedAt:    e.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			})
		}
		recentRows := make([]dashboardEntryRowResponse, 0, len(stats.RecentEntries))
		for _, e := range stats.RecentEntries {
			recentRows = append(recentRows, dashboardEntryRowResponse{
				ID:           e.ID,
				Title:        e.Title,
				EntryType:    e.EntryType,
				Status:       e.Status,
				VoteCount:    e.VoteCount,
				CommentCount: e.CommentCount,
				IsInternal:   e.IsInternal,
				CreatedAt:    e.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			})
		}
		response.Success(c, dashboardResponse{
			TotalEntries:     stats.TotalEntries,
			OpenEntries:      stats.OpenEntries,
			ClosedEntries:    stats.ClosedEntries,
			InternalEntries:  stats.InternalEntries,
			PublicEntries:    stats.PublicEntries,
			TotalVotes:       stats.TotalVotes,
			TotalComments:    stats.TotalComments,
			TotalUsers:       stats.TotalUsers,
			TotalTopics:      stats.TotalTopics,
			TotalReleases:    stats.TotalReleases,
			EntriesByType:    stats.EntriesByType,
			EntriesByStatus:  stats.EntriesByStatus,
			EntriesPerDay:    days,
			TopEntriesByVote: topRows,
			RecentEntries:    recentRows,
		})
	}
}
