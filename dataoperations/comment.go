package dataoperations

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/mongodb"
)

// InsertComment persists a new comment.
func (do *DataOperations) InsertComment(c *models.Comment) error {
	return mongodb.InsertOne(do.DB, CollectionComments, *c)
}

// FindCommentByID returns the comment with the given ID, or nil.
func (do *DataOperations) FindCommentByID(id string) (*models.Comment, error) {
	return mongodb.GetOneById[models.Comment](do.DB, CollectionComments, id)
}

// ListCommentsByEntryID returns every comment on the given entry,
// oldest first (so the conversation reads top-to-bottom). When
// includeInternal is false the listing is restricted to external
// comments — Portal callers pass false; Console callers pass true.
func (do *DataOperations) ListCommentsByEntryID(entryID string, includeInternal bool) ([]models.Comment, error) {
	if entryID == "" {
		return []models.Comment{}, nil
	}
	filter := bson.M{"entryid": entryID}
	if !includeInternal {
		filter["isinternal"] = false
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdat", Value: 1}})
	return mongodb.Query[models.Comment](do.DB, CollectionComments, filter, opts)
}

// SetCommentIsInternal flips the IsInternal flag on a single comment
// and touches updatedat.
func (do *DataOperations) SetCommentIsInternal(id string, value bool) error {
	if err := mongodb.SetValue(do.DB, CollectionComments, id, "isinternal", value); err != nil {
		return err
	}
	return mongodb.SetValue(do.DB, CollectionComments, id, "updatedat", time.Now().UTC())
}

// CountCommentsByEntryID returns the total number of comments on the
// entry. Used by the startup reconciliation pass to repair drift in
// Entry.CommentCount.
func (do *DataOperations) CountCommentsByEntryID(entryID string) (int64, error) {
	if entryID == "" {
		return 0, nil
	}
	return mongodb.Count(do.DB, CollectionComments, bson.M{"entryid": entryID}, nil)
}
