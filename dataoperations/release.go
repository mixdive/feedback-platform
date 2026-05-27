package dataoperations

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/mongodb"
)

// ListReleases returns every release ordered by ReleaseDate descending
// (newer first), with CreatedAt as a stable tie-breaker so two
// releases sharing a date land in a deterministic order.
func (do *DataOperations) ListReleases() ([]models.Release, error) {
	opts := options.Find().SetSort(bson.D{
		{Key: "releasedate", Value: -1},
		{Key: "createdat", Value: -1},
	})
	return mongodb.Query[models.Release](do.DB, CollectionReleases, bson.M{}, opts)
}

// ListReleasesByState returns releases whose State equals the given
// value. Used by the Portal changelog (state="completed").
func (do *DataOperations) ListReleasesByState(state models.ReleaseState) ([]models.Release, error) {
	opts := options.Find().SetSort(bson.D{
		{Key: "releasedate", Value: -1},
		{Key: "createdat", Value: -1},
	})
	return mongodb.Query[models.Release](do.DB, CollectionReleases, bson.M{"state": string(state)}, opts)
}

// FindReleaseByID returns the release with the given ID, or nil.
func (do *DataOperations) FindReleaseByID(id string) (*models.Release, error) {
	if id == "" {
		return nil, nil
	}
	return mongodb.GetOneById[models.Release](do.DB, CollectionReleases, id)
}

// FindReleaseByVersionName is the case-sensitive lookup used by the
// create handler to reject duplicate version names.
func (do *DataOperations) FindReleaseByVersionName(versionName string) (*models.Release, error) {
	return mongodb.QueryOne[models.Release](do.DB, CollectionReleases, bson.M{"versionname": versionName}, nil)
}

// InsertRelease persists a new release record.
func (do *DataOperations) InsertRelease(r *models.Release) error {
	return mongodb.InsertOne(do.DB, CollectionReleases, *r)
}

// ReleasePdfPatch is the optional triple applied to a release's PDF
// attachment slot. A non-nil patch overwrites all three fields
// together (URL, name, size) — clearing the attachment is expressed
// as a patch with empty strings and zero size, never by passing nil.
type ReleasePdfPatch struct {
	URL  string
	Name string
	Size int64
}

// UpdateReleaseFields applies a sparse patch to a single release. nil
// pointers leave the field untouched; non-nil pointers overwrite.
// Touches updatedat when at least one field is present.
func (do *DataOperations) UpdateReleaseFields(
	id string,
	versionName, title, description *string,
	releaseDate *time.Time,
	state *models.ReleaseState,
	pdf *ReleasePdfPatch,
) error {
	if versionName == nil && title == nil && description == nil && releaseDate == nil && state == nil && pdf == nil {
		return nil
	}
	if versionName != nil {
		if err := mongodb.SetValue(do.DB, CollectionReleases, id, "versionname", *versionName); err != nil {
			return err
		}
	}
	if title != nil {
		if err := mongodb.SetValue(do.DB, CollectionReleases, id, "title", *title); err != nil {
			return err
		}
	}
	if description != nil {
		if err := mongodb.SetValue(do.DB, CollectionReleases, id, "description", *description); err != nil {
			return err
		}
	}
	if releaseDate != nil {
		if err := mongodb.SetValue(do.DB, CollectionReleases, id, "releasedate", releaseDate.UTC()); err != nil {
			return err
		}
	}
	if state != nil {
		if err := mongodb.SetValue(do.DB, CollectionReleases, id, "state", string(*state)); err != nil {
			return err
		}
	}
	if pdf != nil {
		if err := mongodb.SetValue(do.DB, CollectionReleases, id, "pdffileurl", pdf.URL); err != nil {
			return err
		}
		if err := mongodb.SetValue(do.DB, CollectionReleases, id, "pdffilename", pdf.Name); err != nil {
			return err
		}
		if err := mongodb.SetValue(do.DB, CollectionReleases, id, "pdffilesize", pdf.Size); err != nil {
			return err
		}
	}
	return mongodb.SetValue(do.DB, CollectionReleases, id, "updatedat", time.Now().UTC())
}

// DeleteRelease removes a single release record by ID.
func (do *DataOperations) DeleteRelease(id string) error {
	return mongodb.DeleteOne(do.DB, CollectionReleases, id)
}

// RemoveReleaseIDFromAllEntries cascade-clears releaseid on every
// entry that referenced the given release. Mirrors the cascade
// pattern used by tag/topic delete — system-side housekeeping, does
// NOT touch updatedat on affected entries.
func (do *DataOperations) RemoveReleaseIDFromAllEntries(releaseID string) error {
	if releaseID == "" {
		return nil
	}
	clearModel := mongo.NewUpdateManyModel().
		SetFilter(bson.M{"releaseid": releaseID}).
		SetUpdate(bson.M{"$set": bson.M{"releaseid": ""}})
	_, err := mongodb.BulkUpdate(do.DB, CollectionEntries, []mongo.WriteModel{clearModel})
	return err
}

// CountEntriesByReleaseID returns how many entries reference the
// given release. Drives the "X entries" count rendered next to each
// release in the Console settings list and on the Portal changelog.
func (do *DataOperations) CountEntriesByReleaseID(releaseID string) (int64, error) {
	if releaseID == "" {
		return 0, nil
	}
	return mongodb.Count(do.DB, CollectionEntries, bson.M{"releaseid": releaseID}, nil)
}

// releaseTypeCountRow is the projection returned by
// CountEntriesPerReleaseByEntryType — one row per (release, entrytype).
type releaseTypeCountRow struct {
	ID struct {
		ReleaseID string `bson:"releaseid"`
		EntryType string `bson:"entrytype"`
	} `bson:"_id"`
	Count int64 `bson:"count"`
}

// CountEntriesPerReleaseByEntryType returns a map of release ID →
// per-type counts. Single $group pass over the entries collection.
// Releases with zero entries are absent from the map — callers default
// to a zero-valued EntryTypeCounts.
func (do *DataOperations) CountEntriesPerReleaseByEntryType() (map[string]EntryTypeCounts, error) {
	pipeline := []bson.D{
		{{Key: "$match", Value: bson.M{"releaseid": bson.M{"$nin": bson.A{"", nil}}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"releaseid": "$releaseid", "entrytype": "$entrytype"},
			"count": bson.M{"$sum": 1},
		}}},
	}
	rows, err := mongodb.Aggregate[releaseTypeCountRow](do.DB, CollectionEntries, pipeline)
	if err != nil {
		return nil, err
	}
	out := make(map[string]EntryTypeCounts)
	for _, r := range rows {
		if r.ID.ReleaseID == "" {
			continue
		}
		c := out[r.ID.ReleaseID]
		c.Total += r.Count
		switch models.EntryType(r.ID.EntryType) {
		case models.EntryTypeFeatureRequest:
			c.FeatureRequest += r.Count
		case models.EntryTypeBug:
			c.Bug += r.Count
		case models.EntryTypeSupport:
			c.Support += r.Count
		default:
			c.Other += r.Count
		}
		out[r.ID.ReleaseID] = c
	}
	return out, nil
}

// ListEntriesByReleaseID returns every entry assigned to the given
// release, newest first. Used by the Portal changelog page to render
// the "shipped in this version" list per release.
//
// publicOnly=true narrows to non-internal entries — Portal callers
// must not see internal records, even when they're assigned to a
// completed release. publicOnly=false (Console) returns the full set.
func (do *DataOperations) ListEntriesByReleaseID(releaseID string, publicOnly bool) ([]models.Entry, error) {
	if releaseID == "" {
		return []models.Entry{}, nil
	}
	filter := bson.M{"releaseid": releaseID}
	if publicOnly {
		filter["isinternal"] = false
	}
	opts := options.Find().SetSort(bson.D{
		{Key: "createdat", Value: -1},
		{Key: "_id", Value: -1},
	})
	return mongodb.Query[models.Entry](do.DB, CollectionEntries, filter, opts)
}

// SetEntryRelease writes the releaseid field on a single entry and
// touches updatedat. Empty value clears the assignment. Caller is
// responsible for validating the ID against the releases collection
// before calling.
func (do *DataOperations) SetEntryRelease(id, releaseID string) error {
	if err := mongodb.SetValue(do.DB, CollectionEntries, id, "releaseid", releaseID); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionEntries, id, "updatedat", time.Now().UTC())
}
