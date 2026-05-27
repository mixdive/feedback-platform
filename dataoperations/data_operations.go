// Package dataoperations is the single data-access layer for Mixdive.
//
// All HTTP handlers receive a *DataOperations via the handler-factory
// pattern; nothing else touches MongoDB directly. The methods are grouped by
// model in per-file sets but all hang off the same *DataOperations.
//
// Mongo access goes through the generic helpers in pkg/mongodb. The default
// driver behavior lowercases Go field names for BSON, so every filter, sort
// key, and field name in this package uses the lowercase form.
package dataoperations

import "github.com/mixdive/feedback-platform/pkg/mongodb"

// Collection name constants.
const (
	CollectionUsers           = "users"
	CollectionSettings        = "settings"
	CollectionEntries     = "entries"
	CollectionEntryTopics = "entry_topics"
	CollectionReleases        = "releases"
	CollectionSessions        = "sessions"
	CollectionVotes           = "votes"
	CollectionComments        = "comments"
	CollectionFiles           = "files"
	CollectionActivities      = "activities"
)

// DataOperations bundles the underlying Mongo connector and exposes typed
// methods for every collection access in Mixdive.
type DataOperations struct {
	DB *mongodb.MongoDB
}

// New constructs a DataOperations around the given pkg/mongodb connector.
// The connector is not connected eagerly; the first method call triggers
// connection.
func New(db *mongodb.MongoDB) *DataOperations {
	return &DataOperations{DB: db}
}

// Close terminates the underlying Mongo connection. Call at shutdown.
func (do *DataOperations) Close() {
	if do == nil || do.DB == nil {
		return
	}
	do.DB.Close()
}
