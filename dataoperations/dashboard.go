package dataoperations

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/mongodb"
)

// DashboardStats is the aggregated picture rendered on the Console
// Dashboard. Every number is a count over the entries collection at
// query time — there is no denormalized snapshot to keep in sync.
//
// EntriesByType / EntriesByStatus are keyed by the lowercase enum
// value. Empty/untyped entries fall under "other" so the type buckets
// always sum to TotalEntries.
type DashboardStats struct {
	TotalEntries     int64
	OpenEntries      int64
	ClosedEntries    int64
	InternalEntries  int64
	PublicEntries    int64
	TotalVotes       int64
	TotalComments    int64
	TotalUsers       int64
	TotalTopics      int64
	TotalReleases    int64
	EntriesByType    map[string]int64
	EntriesByStatus  map[string]int64
	EntriesPerDay    []DashboardDayBucket
	TopEntriesByVote []DashboardEntryRow
	RecentEntries    []DashboardEntryRow
}

// DashboardDayBucket is one calendar day's submission count, used
// to render the 30-day trend chart on the dashboard. Date is the
// UTC date in YYYY-MM-DD form.
type DashboardDayBucket struct {
	Date  string
	Count int64
}

// DashboardEntryRow is the minimal projection of an entry rendered in
// the Top-voted / Recent list cards on the dashboard.
type DashboardEntryRow struct {
	ID           string
	Title        string
	EntryType    string
	Status       string
	VoteCount    int
	CommentCount int
	IsInternal   bool
	CreatedAt    time.Time
}

// GetDashboardStats runs every aggregation needed for the Console
// Dashboard in one helper. Each sub-step issues its own query — we do
// not stuff multiple unrelated $facet stages into one pipeline because
// the page is admin-only and small enough that round-trip count is not
// the bottleneck.
func (do *DataOperations) GetDashboardStats() (*DashboardStats, error) {
	out := &DashboardStats{
		EntriesByType:    map[string]int64{},
		EntriesByStatus:  map[string]int64{},
		EntriesPerDay:    []DashboardDayBucket{},
		TopEntriesByVote: []DashboardEntryRow{},
		RecentEntries:    []DashboardEntryRow{},
	}

	total, err := mongodb.Count(do.DB, CollectionEntries, bson.M{}, nil)
	if err != nil {
		return nil, err
	}
	out.TotalEntries = total

	internal, err := mongodb.Count(do.DB, CollectionEntries, bson.M{"isinternal": true}, nil)
	if err != nil {
		return nil, err
	}
	out.InternalEntries = internal
	out.PublicEntries = total - internal

	closed, err := mongodb.Count(do.DB, CollectionEntries, bson.M{
		"status": bson.M{"$in": bson.A{
			string(models.EntryStatusCompleted),
			string(models.EntryStatusCancelled),
		}},
	}, nil)
	if err != nil {
		return nil, err
	}
	out.ClosedEntries = closed
	out.OpenEntries = total - closed

	users, err := mongodb.Count(do.DB, CollectionUsers, bson.M{}, nil)
	if err != nil {
		return nil, err
	}
	out.TotalUsers = users

	topics, err := mongodb.Count(do.DB, CollectionEntryTopics, bson.M{}, nil)
	if err != nil {
		return nil, err
	}
	out.TotalTopics = topics

	releases, err := mongodb.Count(do.DB, CollectionReleases, bson.M{}, nil)
	if err != nil {
		return nil, err
	}
	out.TotalReleases = releases

	comments, err := mongodb.Count(do.DB, CollectionComments, bson.M{}, nil)
	if err != nil {
		return nil, err
	}
	out.TotalComments = comments

	// Total votes — vote rows are inserted on every toggle, so a
	// per-collection count is the live picture.
	votes, err := mongodb.Count(do.DB, CollectionVotes, bson.M{}, nil)
	if err != nil {
		return nil, err
	}
	out.TotalVotes = votes

	// Entries by entry type. Empty/untyped values fold into "other"
	// so the buckets always sum to TotalEntries.
	typePipeline := []bson.D{
		{{Key: "$group", Value: bson.M{
			"_id":   "$entrytype",
			"count": bson.M{"$sum": 1},
		}}},
	}
	type typeRow struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	typeRows, err := mongodb.Aggregate[typeRow](do.DB, CollectionEntries, typePipeline)
	if err != nil {
		return nil, err
	}
	for _, r := range typeRows {
		key := r.ID
		switch models.EntryType(key) {
		case models.EntryTypeFeatureRequest, models.EntryTypeBug, models.EntryTypeSupport:
			// keep
		default:
			key = string(models.EntryTypeOther)
		}
		out.EntriesByType[key] += r.Count
	}

	// Entries by status.
	statusPipeline := []bson.D{
		{{Key: "$group", Value: bson.M{
			"_id":   "$status",
			"count": bson.M{"$sum": 1},
		}}},
	}
	statusRows, err := mongodb.Aggregate[typeRow](do.DB, CollectionEntries, statusPipeline)
	if err != nil {
		return nil, err
	}
	for _, r := range statusRows {
		key := r.ID
		if !models.IsValidEntryStatus(models.EntryStatus(key)) {
			key = string(models.EntryStatusDefault)
		}
		out.EntriesByStatus[key] += r.Count
	}

	// Entries per day for the last 30 days. Group by the UTC date
	// portion of createdat. Days with zero entries are filled in by
	// the caller so the chart has a continuous x-axis.
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -29).Truncate(24 * time.Hour)
	dayPipeline := []bson.D{
		{{Key: "$match", Value: bson.M{"createdat": bson.M{"$gte": start}}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"$dateToString": bson.M{
					"format": "%Y-%m-%d",
					"date":   "$createdat",
				},
			},
			"count": bson.M{"$sum": 1},
		}}},
	}
	dayRows, err := mongodb.Aggregate[typeRow](do.DB, CollectionEntries, dayPipeline)
	if err != nil {
		return nil, err
	}
	byDate := map[string]int64{}
	for _, r := range dayRows {
		byDate[r.ID] = r.Count
	}
	for i := 0; i < 30; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		out.EntriesPerDay = append(out.EntriesPerDay, DashboardDayBucket{
			Date:  d,
			Count: byDate[d],
		})
	}

	// Top entries by vote (cap 5).
	top, err := do.ListEntries(EntryListFilter{Sort: "top", Limit: 5})
	if err != nil {
		return nil, err
	}
	for _, e := range top {
		out.TopEntriesByVote = append(out.TopEntriesByVote, buildDashboardEntryRow(e))
	}

	// Most recent entries (cap 5).
	recent, err := do.ListEntries(EntryListFilter{Sort: "new", Limit: 5})
	if err != nil {
		return nil, err
	}
	for _, e := range recent {
		out.RecentEntries = append(out.RecentEntries, buildDashboardEntryRow(e))
	}

	return out, nil
}

func buildDashboardEntryRow(e models.Entry) DashboardEntryRow {
	return DashboardEntryRow{
		ID:           e.ID,
		Title:        e.Title,
		EntryType:    string(e.EntryType),
		Status:       string(e.Status),
		VoteCount:    e.VoteCount,
		CommentCount: e.CommentCount,
		IsInternal:   e.IsInternal,
		CreatedAt:    e.CreatedAt,
	}
}
