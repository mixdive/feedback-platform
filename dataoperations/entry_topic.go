package dataoperations

import (
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/mongodb"
)

// ListEntryTopics returns every topic ordered by SortOrder, with
// Title as the secondary sort so two topics sharing a SortOrder
// land in a stable position.
func (do *DataOperations) ListEntryTopics() ([]models.EntryTopic, error) {
	opts := options.Find().SetSort(bson.D{
		{Key: "sortorder", Value: 1},
		{Key: "title", Value: 1},
	})
	return mongodb.Query[models.EntryTopic](do.DB, CollectionEntryTopics, bson.M{}, opts)
}

// FindEntryTopicByID returns the topic with the given ID, or nil.
func (do *DataOperations) FindEntryTopicByID(id string) (*models.EntryTopic, error) {
	if id == "" {
		return nil, nil
	}
	return mongodb.GetOneById[models.EntryTopic](do.DB, CollectionEntryTopics, id)
}

// FindEntryTopicByTitle is the case-insensitive lookup used by the
// create handler to reject duplicate titles and by the AI analyzer
// to avoid double-creating a topic that already exists.
func (do *DataOperations) FindEntryTopicByTitle(title string) (*models.EntryTopic, error) {
	filter := bson.M{
		"title": bson.M{
			"$regex":   "^" + regexp.QuoteMeta(title) + "$",
			"$options": "i",
		},
	}
	return mongodb.QueryOne[models.EntryTopic](do.DB, CollectionEntryTopics, filter, nil)
}

// InsertEntryTopic persists a new topic record.
func (do *DataOperations) InsertEntryTopic(t *models.EntryTopic) error {
	return mongodb.InsertOne(do.DB, CollectionEntryTopics, *t)
}

// UpdateEntryTopicFields applies a sparse patch to a single topic.
// nil pointers leave the field untouched; non-nil pointers
// overwrite. Touches updatedat when at least one field is present.
func (do *DataOperations) UpdateEntryTopicFields(id string, title, description, color *string, sortOrder *int) error {
	if title == nil && description == nil && color == nil && sortOrder == nil {
		return nil
	}
	if title != nil {
		if err := mongodb.SetValue(do.DB, CollectionEntryTopics, id, "title", *title); err != nil {
			return err
		}
	}
	if description != nil {
		if err := mongodb.SetValue(do.DB, CollectionEntryTopics, id, "description", *description); err != nil {
			return err
		}
	}
	if color != nil {
		if err := mongodb.SetValue(do.DB, CollectionEntryTopics, id, "color", *color); err != nil {
			return err
		}
	}
	if sortOrder != nil {
		if err := mongodb.SetValue(do.DB, CollectionEntryTopics, id, "sortorder", *sortOrder); err != nil {
			return err
		}
	}
	return mongodb.SetValue(do.DB, CollectionEntryTopics, id, "updatedat", time.Now().UTC())
}

// DeleteEntryTopic removes a single topic record by ID.
func (do *DataOperations) DeleteEntryTopic(id string) error {
	return mongodb.DeleteOne(do.DB, CollectionEntryTopics, id)
}

// topicCountRow is the projection returned by CountEntriesPerTopic —
// the $group stage emits {_id: <topicID>, count: <int>} per topic.
type topicCountRow struct {
	ID    string `bson:"_id"`
	Count int64  `bson:"count"`
}

// CountEntriesPerTopic returns a map of topic ID → number of entries
// whose topicids array contains that ID. Single-roundtrip aggregation:
// unwind topicids, group by id, count. Topics with zero entries are
// absent from the map — callers default to 0.
func (do *DataOperations) CountEntriesPerTopic() (map[string]int64, error) {
	pipeline := []bson.D{
		{{Key: "$match", Value: bson.M{"topicids.0": bson.M{"$exists": true}}}},
		{{Key: "$unwind", Value: "$topicids"}},
		{{Key: "$group", Value: bson.M{"_id": "$topicids", "count": bson.M{"$sum": 1}}}},
	}
	rows, err := mongodb.Aggregate[topicCountRow](do.DB, CollectionEntries, pipeline)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		if r.ID == "" {
			continue
		}
		out[r.ID] = r.Count
	}
	return out, nil
}

// EntryTypeCounts is the per-bucket count of entries for one group
// (topic or release), bucketed by entry type. Other is the catch-all
// — both the EntryTypeOther enum value and the empty/untyped ""
// land here so Total = FeatureRequest + Bug + Support + Other.
type EntryTypeCounts struct {
	Total          int64
	FeatureRequest int64
	Bug            int64
	Support        int64
	Other          int64
}

// topicTypeCountRow is the projection returned by
// CountEntriesPerTopicByEntryType — one row per (topic, entrytype).
type topicTypeCountRow struct {
	ID struct {
		TopicID   string `bson:"topicid"`
		EntryType string `bson:"entrytype"`
	} `bson:"_id"`
	Count int64 `bson:"count"`
}

// CountEntriesPerTopicByEntryType returns a map of topic ID →
// EntryTypeCounts. Single $unwind + $group pass. Topics with zero
// entries are absent from the map.
func (do *DataOperations) CountEntriesPerTopicByEntryType() (map[string]EntryTypeCounts, error) {
	pipeline := []bson.D{
		{{Key: "$match", Value: bson.M{"topicids.0": bson.M{"$exists": true}}}},
		{{Key: "$unwind", Value: "$topicids"}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"topicid": "$topicids", "entrytype": "$entrytype"},
			"count": bson.M{"$sum": 1},
		}}},
	}
	rows, err := mongodb.Aggregate[topicTypeCountRow](do.DB, CollectionEntries, pipeline)
	if err != nil {
		return nil, err
	}
	out := make(map[string]EntryTypeCounts)
	for _, r := range rows {
		if r.ID.TopicID == "" {
			continue
		}
		c := out[r.ID.TopicID]
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
		out[r.ID.TopicID] = c
	}
	return out, nil
}

// RemoveTopicIDFromAllEntries cascade-removes a topic ID from every
// entry's topicids array. Used by the delete-topic handler so an
// admin removing a topic doesn't leave dangling IDs across the
// collection.
//
// Intentionally does NOT touch updatedat on affected entries — this
// is system-side housekeeping, not a user edit.
func (do *DataOperations) RemoveTopicIDFromAllEntries(topicID string) error {
	if topicID == "" {
		return nil
	}
	// Pull from both the main array and the AI-provenance subset so
	// no dangling IDs remain after a topic delete. Also $unsets the
	// reason key on topicanalysis.suggestedtopicreasons.<topicID>.
	pullModel := mongo.NewUpdateManyModel().
		SetFilter(bson.M{"$or": bson.A{
			bson.M{"topicids": topicID},
			bson.M{"aitopicids": topicID},
		}}).
		SetUpdate(bson.M{"$pull": bson.M{
			"topicids":   topicID,
			"aitopicids": topicID,
		}})
	unsetModel := mongo.NewUpdateManyModel().
		SetFilter(bson.M{"topicanalysis.suggestedtopicreasons." + topicID: bson.M{"$exists": true}}).
		SetUpdate(bson.M{"$unset": bson.M{
			"topicanalysis.suggestedtopicreasons." + topicID: "",
		}})
	_, err := mongodb.BulkUpdate(do.DB, CollectionEntries, []mongo.WriteModel{pullModel, unsetModel})
	return err
}
