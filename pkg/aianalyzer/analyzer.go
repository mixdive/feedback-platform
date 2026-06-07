// Package aianalyzer wraps Anthropic's Claude SDK behind a small set of
// interfaces and ships an async Worker that polls Mongo for entries
// needing analysis.
//
// The polling design is deliberately Cloud Run friendly: there is no
// in-memory queue, no fan-out from request handlers, and no shared
// state between instances. Instead, the entry document IS the queue
// item — its categoryanalysis sub-struct holds the lifecycle state
// (status + claimedat). Every instance polls Mongo on a ticker;
// findOneAndUpdate guarantees that two instances polling at the same
// instant cannot claim the same entry, and a TTL on claimedat means
// crashed-instance claims are reclaimable by the survivor.
//
// The package mirrors pkg/mongodb and pkg/storage in role: a runtime
// adapter providing infrastructure that the rest of the codebase
// composes.
package aianalyzer

import (
	"context"
	"time"

	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// DefaultModel is the Claude model used when Settings.AI.Model is empty.
// Haiku is chosen for category-style classification — cheapest, fastest,
// and well above task difficulty. Admins override per-deployment from
// the Console AI settings page.
const DefaultModel = "claude-haiku-4-5"

// Snapshot is a point-in-time view of the AI runtime configuration.
// Returned by Worker.Snapshot so analyzers can read the current API
// key + model on every call without holding a Worker reference (which
// would create an analyzer ↔ worker import cycle and would also make
// analyzers harder to mock).
type Snapshot struct {
	Enabled bool
	APIKey  string
	Model   string
}

// Analyzer is the contract every LLM-backed analysis type implements.
// Each implementation owns its own per-entry sub-struct on the Entry
// document plus the four DB queries needed to drive the polling worker:
//
//   - ClaimNext atomically claims one entry needing this analyzer to
//     run (returns nil when nothing is pending).
//   - Process runs the LLM call against the claimed entry and persists
//     the result, releasing the claim. The worker calls this with an
//     entry it already received from ClaimNext.
//   - PendingCount returns how many entries still need analysis AND
//     are not currently claimed (drives the queue-depth gauge).
//   - InFlightCount returns how many entries are currently claimed by
//     some instance (drives the in-flight gauge).
//
// Every method takes claimTTL because the lifecycle uses the same
// distributed-lock TTL semantics across all four methods.
//
// Adding a new analyzer type later is a self-contained delta: implement
// this interface, add the matching DataOperations methods (count /
// in-flight / claim / set-result / fail), register the analyzer at
// startup. The Worker itself stays unchanged.
type Analyzer interface {
	Name() string
	ClaimNext(do dataoperations.Store, claimTTL time.Duration) (*models.Entry, error)
	Process(ctx context.Context, do dataoperations.Store, entry *models.Entry) error
	PendingCount(do dataoperations.Store, claimTTL time.Duration) (int, error)
	InFlightCount(do dataoperations.Store, claimTTL time.Duration) (int, error)
}
