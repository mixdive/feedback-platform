package dataoperations

import (
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/mixdive/feedback-platform/pkg/mongodb"
)

// uniqueVoteIndexName is the unique (userid, entryid) constraint on the
// votes collection. Named explicitly so a later spec change can be spotted
// — Mongo rejects a redefinition under the same name rather than silently
// keeping the old one.
const uniqueVoteIndexName = "uniq_vote_userid_entryid"

// EnsureIndexes creates the indexes Mixdive's invariants depend on. Called
// once at startup, after ReconcileVoteCounts — the unique vote index cannot
// build while duplicate rows are still present, so the cleanup has to go
// first.
//
// Creating an index never modifies or removes documents. The one failure
// mode is a unique index that existing data violates, and Mongo's answer to
// that is to refuse to build the index and leave the collection exactly as
// it was. That's why the caller logs a failure here and keeps booting: a
// missing index costs the belt-and-braces guarantee, not the data, and the
// atomic upsert in InsertVoteIfAbsent still holds the line on its own.
//
// Idempotent: re-creating an index with an identical spec is a no-op.
func (do *DataOperations) EnsureIndexes() error {
	var errs []error

	// One vote per (user, entry). Partial so the constraint applies only to
	// attributable rows: legacy votes with an empty userid would otherwise
	// all collide with each other on a single ("", entryid) key and take
	// the whole index build down with them.
	if err := mongodb.EnsureIndex(do.DB, CollectionVotes, mongo.IndexModel{
		Keys: bson.D{{Key: "userid", Value: 1}, {Key: "entryid", Value: 1}},
		Options: options.Index().
			SetName(uniqueVoteIndexName).
			SetUnique(true).
			SetPartialFilterExpression(bson.M{"userid": bson.M{"$gt": ""}}),
	}); err != nil {
		errs = append(errs, wrapIndexError(CollectionVotes, uniqueVoteIndexName, err))
	}

	// Supports the per-entry vote lookups: CountVotesForEntry, the merge
	// handler's migration, and the reconcile pass's tally.
	if err := mongodb.EnsureIndex(do.DB, CollectionVotes, mongo.IndexModel{
		Keys:    bson.D{{Key: "entryid", Value: 1}},
		Options: options.Index().SetName("vote_entryid"),
	}); err != nil {
		errs = append(errs, wrapIndexError(CollectionVotes, "vote_entryid", err))
	}

	return errors.Join(errs...)
}

// wrapIndexError annotates an index failure with the collection and index
// name, and calls out the duplicate-data case explicitly — that's the one an
// operator can act on, by re-running the reconcile pass.
func wrapIndexError(collection, name string, err error) error {
	if mongodb.IsDuplicateKeyError(err) || strings.Contains(err.Error(), "E11000") {
		return errors.New(collection + "." + name +
			": existing documents violate the unique key (run ReconcileVoteCounts, then retry): " + err.Error())
	}
	return errors.New(collection + "." + name + ": " + err.Error())
}
