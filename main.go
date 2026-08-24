package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/demo"
	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/aianalyzer"
	"github.com/mixdive/feedback-platform/pkg/mongodb"
	"github.com/mixdive/feedback-platform/pkg/storage"
)

// httpAddr is hardcoded for v0.1 — Cloud Run expects 8080 by default and
// every supported deployment target accepts it. Promote to settings if a
// future deployment ever needs a different port.
const httpAddr = ":8080"

// mongoDatabaseName is the database that holds every Mixdive collection.
// Hardcoded for v0.1 — the only env var we accept is MONGO_URI.
const mongoDatabaseName = "mixdive"

// @title			Mixdive API
// @version		0.1.0
// @description	Open-source feedback portal with an admin console.
// @description	Single tenant per deployment. Authentication via HTTPOnly session cookie.
//
// @host		localhost:8080
// @BasePath	/
// @schemes	http https
//
// @securityDefinitions.apikey	CookieAuth
// @in							cookie
// @name						mixdive_session
func main() {
	// do is the data-access layer behind the Store interface. demoUser is
	// non-nil only in DEMO mode, where it carries the synthetic admin that
	// every visitor is auto-authenticated as.
	var do dataoperations.Store
	var demoUser *models.User
	var uploadSettings models.UploadSettings
	// voteJanitorEnabled stays false in DEMO mode and on a not-yet-set-up
	// deployment — neither has data worth reconciling.
	var voteJanitorEnabled bool

	if isDemoEnabled(os.Getenv("DEMO")) {
		// DEMO mode: a self-contained, in-memory, read-only store. No
		// MongoDB connection is required at all. This is the single
		// sanctioned exception to the "only MONGO_URI" config rule.
		repo := demo.New()
		do = repo
		demoUser = repo.DemoAdmin()
		// Uploads stay off in the demo (no blob storage); the rest of the
		// settings come from the in-memory repo. The idempotent
		// ensure-passes are skipped — the dataset is already complete and
		// immutable.
		log.Printf("mixdive: DEMO mode — in-memory read-only data, auto-login as %s", demo.AdminEmail)
	} else {
		mongoURI := os.Getenv("MONGO_URI")
		if mongoURI == "" {
			log.Fatal("mixdive: MONGO_URI is required")
		}

		// pkg/mongodb is lazy — first method call triggers connection. We
		// do no bootstrapping here: status seeding, organization name,
		// admin user — every one of those is filled in by the first-run
		// setup form, which is what the user sees at the root URL on a
		// fresh deployment.
		mongoConn := mongodb.New(mongoURI, mongoDatabaseName, false)
		do = dataoperations.New(mongoConn)

		// Idempotent ensure-passes for already-set-up deployments. Gated
		// on SetupCompleted so a fresh deploy (no Mongo, no settings)
		// doesn't trigger spurious work before the user opens the setup
		// form.
		if s, err := do.GetSettings(); err == nil && s != nil && s.SetupCompleted {
			// Lift legacy flat entry-type templates (map[string]string,
			// keyed by EntryType) into the nested multi-language shape
			// (map[string]map[string]string, keyed by EntryType then by
			// language code). Idempotent; must run BEFORE
			// EnsureFeedbackDefaults so a deployment with an existing
			// flat map does not get bundled defaults stacked on top.
			if err := do.MigrateEntryTypeTemplatesToMultiLang(); err != nil {
				log.Printf("mixdive: migrate entry-type templates to multi-lang: %v", err)
			}
			// Backfill feedback policy defaults (entry-type templates) on
			// deployments that came up before those fields shipped.
			// Idempotent — no-op once the templates map is set.
			if err := do.EnsureFeedbackDefaults(); err != nil {
				log.Printf("mixdive: ensure feedback defaults: %v", err)
			}
			// Backfill the entry-created activity row for entries that
			// pre-date the activities collection. Subsequent activities
			// (status changes, topic edits, …) are recorded forward-only
			// from the moment activities lands — we can't fabricate them.
			// Idempotent.
			if n, err := do.EnsureEntryCreatedActivities(); err != nil {
				log.Printf("mixdive: ensure entry-created activities: %v", err)
			} else if n > 0 {
				log.Printf("mixdive: backfilled %d entry-created activity row(s)", n)
			}
			// Re-sync Entry.VoteCount with the votes collection and move
			// any duplicate (user, entry) vote rows into votes_archive.
			// Nothing is destroyed — the earliest vote of each pair stays
			// put and the surplus is archived before it's removed. Must run
			// BEFORE EnsureIndexes: the unique vote index cannot build over
			// duplicates.
			logVoteReconcile(do.ReconcileVoteCounts())
			// Indexes last, and never fatal. A unique index that existing
			// data violates simply doesn't get built; the collection is
			// untouched and the atomic upsert in InsertVoteIfAbsent still
			// enforces one vote per user on its own.
			if err := do.EnsureIndexes(); err != nil {
				log.Printf("mixdive: ensure indexes: %v", err)
			}
			voteJanitorEnabled = true
			uploadSettings = s.Uploads
		}
	}
	defer do.Close()

	// Keep the counters honest while the process runs, not just at boot.
	// Cheap and write-free on a healthy deployment, and idempotent enough
	// that two instances ticking at once is harmless.
	if voteJanitorEnabled {
		stopJanitor := startVoteJanitor(do)
		defer stopJanitor()
	}

	// Storage holder for uploaded blobs. Backend choice lives on the
	// settings document; this constructor never fails — a misconfigured
	// backend surfaces on the next Put and on the Console settings
	// page rather than killing boot. The handler that patches settings
	// calls Reload to hot-swap the backend in place.
	store := storage.NewHolder(uploadSettings)
	if msg := store.LastBuildError(); msg != "" {
		log.Printf("mixdive: storage init: %s (admin can fix in Console settings)", msg)
	}

	// AI analyzer worker. Polls Mongo on a ticker; atomically claims
	// entries needing analysis via findOneAndUpdate. Multi-instance
	// safe — two Cloud Run instances polling at the same instant
	// cannot claim the same entry, and a TTL on claimedat makes
	// crashed-instance recovery automatic. Adding a future analyzer
	// (spam, sentiment, …) is one extra Register call here; the
	// Worker itself stays unchanged.
	worker := aianalyzer.NewWorker(do)
	worker.Register(aianalyzer.NewEntryTypeAnalyzer(worker.Snapshot))
	worker.Register(aianalyzer.NewTopicAnalyzer(worker.Snapshot))
	worker.Register(aianalyzer.NewRelationAnalyzer(worker.Snapshot))
	worker.Start(context.Background())
	defer worker.Stop()

	r := newRouter(do, store, worker, demoUser)
	log.Printf("mixdive: listening on %s", httpAddr)
	if err := r.Run(httpAddr); err != nil {
		log.Fatalf("mixdive: server exited: %v", err)
	}
}

// voteJanitorInterval is how often a running instance re-syncs
// Entry.VoteCount with the votes collection. Hourly is plenty: the atomic
// upsert prevents drift in the first place, so this is a safety net for
// crashes between the vote write and the counter increment, and for
// anything a future handler gets wrong.
const voteJanitorInterval = time.Hour

// voteRepairLogLimit caps the per-entry detail lines one reconcile run
// prints. The first run on a drifted deployment can repair a lot of
// entries at once, and burying the summary under thousands of lines helps
// nobody — the full picture always survives in votes_archive.
const voteRepairLogLimit = 100

// startVoteJanitor runs ReconcileVoteCounts on a ticker and returns a stop
// function. The first run has already happened at startup, so the ticker
// deliberately doesn't fire immediately.
func startVoteJanitor(do dataoperations.Store) func() {
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(voteJanitorInterval)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				logVoteReconcile(do.ReconcileVoteCounts())
			}
		}
	}()
	return func() { close(done) }
}

// logVoteReconcile writes the audit trail for one reconcile run. A clean
// run stays silent; a run that changed something logs every counter it
// rewrote, so an operator can see exactly what moved and reverse it from
// votes_archive if they disagree.
func logVoteReconcile(rep dataoperations.VoteReconcileReport, err error) {
	if err != nil {
		log.Printf("mixdive: reconcile vote counts: %v (%s before the failure)", err, rep.Summary())
		return
	}
	if rep.Clean() {
		return
	}
	log.Printf("mixdive: reconcile vote counts: %s", rep.Summary())
	for i, r := range rep.Repairs {
		if i == voteRepairLogLimit {
			log.Printf("mixdive: … and %d more counter repair(s) not listed",
				len(rep.Repairs)-voteRepairLogLimit)
			break
		}
		log.Printf("mixdive: entry %s votecount %d -> %d", r.EntryID, r.From, r.To)
	}
}

// isDemoEnabled reports whether the DEMO env var requests demo mode.
// Accepts the common truthy spellings (case-insensitive): 1, true, yes,
// on. Anything else (including empty/unset) is false.
func isDemoEnabled(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
