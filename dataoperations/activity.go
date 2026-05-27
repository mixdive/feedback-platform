package dataoperations

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/mongodb"
)

// InsertActivity persists a single activity row. Callers (handlers and
// the AI analyzer worker) emit one row per side-effect the Console
// timeline should surface.
func (do *DataOperations) InsertActivity(a *models.Activity) error {
	return mongodb.InsertOne(do.DB, CollectionActivities, *a)
}

// ListActivitiesByEntryID returns every activity row for an entry,
// oldest first. Console renderers merge this stream with the entry's
// comments by CreatedAt to produce the unified timeline.
func (do *DataOperations) ListActivitiesByEntryID(entryID string) ([]models.Activity, error) {
	filter := bson.M{"entryid": entryID}
	// _id desc as a stable tie-breaker keeps two activities written in
	// the same millisecond (e.g. topic-added pair from a multi-select
	// save) in a deterministic order across requests.
	opts := options.Find().SetSort(bson.D{
		{Key: "createdat", Value: 1},
		{Key: "_id", Value: 1},
	})
	return mongodb.Query[models.Activity](do.DB, CollectionActivities, filter, opts)
}

// EnsureEntryCreatedActivities backfills a single entry-created
// activity per existing entry that doesn't already have one. Runs
// once at startup so pilot deployments that pre-date the activities
// collection don't show empty timelines. CreatedAt mirrors the
// entry's own creation time; ActorID mirrors the entry's UserID
// (empty for anonymous portal submissions). Idempotent — re-running
// is a no-op once every entry has its created row.
func (do *DataOperations) EnsureEntryCreatedActivities() (int, error) {
	entries, err := mongodb.GetAll[models.Entry](do.DB, CollectionEntries)
	if err != nil {
		return 0, err
	}
	if len(entries) == 0 {
		return 0, nil
	}
	existing, err := mongodb.Query[models.Activity](do.DB, CollectionActivities, bson.M{
		"type": string(models.ActivityTypeEntryCreated),
	}, nil)
	if err != nil {
		return 0, err
	}
	have := make(map[string]struct{}, len(existing))
	for _, a := range existing {
		have[a.EntryID] = struct{}{}
	}
	inserted := 0
	for _, e := range entries {
		if _, ok := have[e.ID]; ok {
			continue
		}
		a := models.NewActivity()
		a.EntryID = e.ID
		a.Type = models.ActivityTypeEntryCreated
		a.Source = models.ActivitySourceUser
		a.ActorID = e.UserID
		a.CreatedAt = e.CreatedAt
		if err := do.InsertActivity(a); err != nil {
			return inserted, err
		}
		inserted++
	}
	return inserted, nil
}
