package main

import (
	"context"
	"log"
	"os"

	"github.com/mixdive/feedback-platform/dataoperations"
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
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("mixdive: MONGO_URI is required")
	}

	// pkg/mongodb is lazy — first method call triggers connection. We do
	// no bootstrapping here: status seeding, organization name, admin user
	// — every one of those is filled in by the first-run setup form, which
	// is what the user sees at the root URL on a fresh deployment.
	mongoConn := mongodb.New(mongoURI, mongoDatabaseName, false)
	do := dataoperations.New(mongoConn)
	defer do.Close()

	// Idempotent ensure-passes for already-set-up deployments. Gated on
	// SetupCompleted so a fresh deploy (no Mongo, no settings) doesn't
	// trigger spurious work before the user opens the setup form.
	var uploadSettings models.UploadSettings
	if s, err := do.GetSettings(); err == nil && s != nil && s.SetupCompleted {
		// Backfill feedback policy defaults on deployments that came up
		// before the per-user vote quota shipped. Idempotent — no-op
		// once feedback.maxvotesperuser is set.
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
		uploadSettings = s.Uploads
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

	r := newRouter(do, store, worker)
	log.Printf("mixdive: listening on %s", httpAddr)
	if err := r.Run(httpAddr); err != nil {
		log.Fatalf("mixdive: server exited: %v", err)
	}
}
