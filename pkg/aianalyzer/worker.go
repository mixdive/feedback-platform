package aianalyzer

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/mixdive/feedback-platform/dataoperations"
)

// Defaults the worker uses when not overridden. Tuned for a small pilot
// deployment: a 5-second tick is responsive enough that submitted
// entries appear analyzed before an admin can navigate to them, while
// keeping the Mongo query rate negligible (12 polls/min/instance). The
// 5-minute claim TTL is comfortably longer than the worst-case LLM
// latency (typical Haiku call is 1-2s) so a healthy worker never has
// its claim stolen, while still bounding crashed-instance recovery to
// ~5 minutes.
const (
	DefaultPollInterval = 5 * time.Second
	DefaultClaimTTL     = 5 * time.Minute
)

// Stats is the response shape returned by the queue-stats endpoint.
//
// The two number pairs (PendingByName / TotalPending and
// InFlightByName / TotalInFlight) come straight from atomic Mongo
// counts — they are the cluster-wide truth, not a per-instance view.
// On a multi-instance Cloud Run deployment, every instance returns the
// same numbers because the underlying state lives in Mongo.
type Stats struct {
	Enabled        bool           `json:"enabled"`
	PollInterval   string         `json:"pollInterval"`
	ClaimTTL       string         `json:"claimTtl"`
	PendingByName  map[string]int `json:"pendingByName"`
	InFlightByName map[string]int `json:"inFlightByName"`
	TotalPending   int            `json:"totalPending"`
	TotalInFlight  int            `json:"totalInFlight"`
	// TotalAnalyzed is the cumulative count of successful analyzer
	// runs over this deployment's lifetime, sourced from
	// Settings.AI.TotalAnalyzed which is $inc'd atomically per success.
	TotalAnalyzed int `json:"totalAnalyzed"`
}

// Worker drives the AI analysis pipeline by polling Mongo for entries
// needing each registered analyzer to run.
//
// Designed for multi-instance Cloud Run safety: claim is via Mongo's
// atomic findOneAndUpdate, so two instances polling at the same instant
// will never process the same entry. A claim TTL means a crashed
// instance's claims are automatically reclaimable by the survivor; no
// separate sweeper job is needed.
//
// The Worker holds no per-entry state — every fact about the queue
// (what's pending, what's in flight, what's done) lives in Mongo. This
// is what makes the design Cloud-Run-scale-to-N safe.
type Worker struct {
	do           dataoperations.Store
	registry     map[string]Analyzer
	pollInterval time.Duration
	claimTTL     time.Duration

	snapMu sync.RWMutex
	snap   Snapshot

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// NewWorker constructs a Worker with default poll interval and claim
// TTL. Analyzers are added with Register before Start.
func NewWorker(do dataoperations.Store) *Worker {
	return &Worker{
		do:           do,
		registry:     make(map[string]Analyzer),
		pollInterval: DefaultPollInterval,
		claimTTL:     DefaultClaimTTL,
	}
}

// Register adds an analyzer to the worker's registry. Must be called
// before Start (the registry is not protected by a mutex). Panics on
// duplicate name — programmer error, not runtime.
func (w *Worker) Register(a Analyzer) {
	if _, dup := w.registry[a.Name()]; dup {
		panic("aianalyzer: duplicate analyzer name " + a.Name())
	}
	w.registry[a.Name()] = a
}

// Snapshot returns the current AI runtime configuration. Used by
// analyzer implementations to read the API key / model on each call.
func (w *Worker) Snapshot() Snapshot {
	w.snapMu.RLock()
	defer w.snapMu.RUnlock()
	return w.snap
}

// Refresh re-reads Settings.AI and updates the in-memory snapshot.
// Called by the AI settings handler post-write so the next poll picks
// up the new configuration without a server restart.
func (w *Worker) Refresh() {
	s, err := w.do.GetSettings()
	next := Snapshot{}
	if err == nil && s != nil {
		next.Enabled = s.AI.Enabled
		next.APIKey = s.AI.APIKey
		next.Model = s.AI.Model
	}
	w.snapMu.Lock()
	w.snap = next
	w.snapMu.Unlock()
}

// Start refreshes the snapshot once and spins up the polling goroutine.
// Pass a long-lived context (e.g. context.Background) — the worker
// derives a cancellable child from it.
func (w *Worker) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	w.cancel = cancel
	w.Refresh()
	w.wg.Add(1)
	go w.run(ctx)
}

// Stop cancels the polling context and waits for any in-flight
// analyzer call to finish. Safe to call once after Start.
func (w *Worker) Stop() {
	if w.cancel == nil {
		return
	}
	w.cancel()
	w.wg.Wait()
	w.cancel = nil
}

// Stats returns the cluster-wide queue picture. Cheap (one Mongo
// count per analyzer, plus a single Settings doc read for the
// cumulative TotalAnalyzed counter).
func (w *Worker) Stats() Stats {
	pending := make(map[string]int, len(w.registry))
	inFlight := make(map[string]int, len(w.registry))
	totalPending := 0
	totalInFlight := 0
	for name, a := range w.registry {
		if n, err := a.PendingCount(w.do, w.claimTTL); err == nil {
			pending[name] = n
			totalPending += n
		}
		if n, err := a.InFlightCount(w.do, w.claimTTL); err == nil {
			inFlight[name] = n
			totalInFlight += n
		}
	}
	totalAnalyzed := 0
	if s, err := w.do.GetSettings(); err == nil && s != nil {
		totalAnalyzed = s.AI.TotalAnalyzed
	}
	return Stats{
		Enabled:        w.Snapshot().Enabled,
		PollInterval:   w.pollInterval.String(),
		ClaimTTL:       w.claimTTL.String(),
		PendingByName:  pending,
		InFlightByName: inFlight,
		TotalPending:   totalPending,
		TotalInFlight:  totalInFlight,
		TotalAnalyzed:  totalAnalyzed,
	}
}

// run is the polling loop. One tick per pollInterval; on each tick,
// every registered analyzer gets a chance to claim and process one
// entry. Multiple analyzers don't compete for the same claim — each
// owns its own per-analyzer sub-struct, so claims are independent.
//
// We process at most one entry per analyzer per tick. Throughput
// scales by raising the tick rate or by running multiple Cloud Run
// instances; both approaches respect the atomic-claim contract.
func (w *Worker) run(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	if !w.Snapshot().Enabled {
		return
	}
	for _, a := range w.registry {
		w.processOne(ctx, a)
	}
}

func (w *Worker) processOne(ctx context.Context, a Analyzer) {
	e, err := a.ClaimNext(w.do, w.claimTTL)
	if err != nil {
		log.Printf("aianalyzer: %s claim: %v", a.Name(), err)
		return
	}
	if e == nil {
		return
	}
	if err := a.Process(ctx, w.do, e); err != nil {
		log.Printf("aianalyzer: %s on %s: %v", a.Name(), e.ID, err)
		_ = w.do.RecordAISettingsError(err.Error())
		return
	}
	_ = w.do.RecordAISettingsSuccess()
}
