package demo

import (
	"errors"
	"testing"

	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// TestRepoConsistency builds the demo dataset and asserts it is internally
// consistent and that the read-only contract holds. Runs with the normal
// `go test ./...` — no Mongo, no network.
func TestRepoConsistency(t *testing.T) {
	r := New()

	// Settings + auto-login admin.
	if r.DemoAdmin() == nil || !r.DemoAdmin().IsAdmin() {
		t.Fatal("DemoAdmin must be a non-nil admin")
	}
	s, _ := r.GetSettings()
	if s == nil || !s.SetupCompleted || s.ProjectName != projectName {
		t.Fatalf("settings not provisioned: %+v", s)
	}

	all, _ := r.ListEntries(dataoperations.EntryListFilter{Limit: 100})
	if len(all) == 0 {
		t.Fatal("no entries built")
	}

	// VoteCount on each entry equals its actual vote rows.
	for _, e := range all {
		got, _ := r.CountVotesForEntry(e.ID)
		if got != e.VoteCount {
			t.Errorf("entry %q VoteCount=%d but %d vote rows", e.Title, e.VoteCount, got)
		}
		// CommentCount equals actual comment rows.
		cc, _ := r.CountCommentsByEntryID(e.ID)
		if int64(e.CommentCount) != cc {
			t.Errorf("entry %q CommentCount=%d but %d comment rows", e.Title, e.CommentCount, cc)
		}
	}

	// Relations are mirrored on both peers.
	byID := map[string]models.Entry{}
	for _, e := range all {
		byID[e.ID] = e
	}
	for _, e := range all {
		for _, rel := range e.Relations {
			peer, ok := byID[rel.EntryID]
			if !ok {
				t.Errorf("entry %q relates to unknown peer %q", e.Title, rel.EntryID)
				continue
			}
			mirrored := false
			for _, back := range peer.Relations {
				if back.EntryID == e.ID {
					mirrored = true
				}
			}
			if !mirrored {
				t.Errorf("relation %q -> %q not mirrored on peer", e.Title, peer.Title)
			}
		}
	}

	// Portal-filtered list (IsInternal=false) hides every internal entry.
	pub, _ := r.ListEntries(dataoperations.EntryListFilter{Limit: 100, IsInternal: boolPtr(false)})
	for _, e := range pub {
		if e.IsInternal {
			t.Errorf("internal entry %q leaked into public list", e.Title)
		}
	}
	if len(pub) >= len(all) {
		t.Error("expected at least one internal entry filtered out")
	}

	// Writes are rejected.
	if err := r.InsertEntry(models.NewEntry()); !errors.Is(err, dataoperations.ErrReadOnly) {
		t.Errorf("InsertEntry should return ErrReadOnly, got %v", err)
	}
	if err := r.SetEntryStatus("x", models.EntryStatusCompleted); !errors.Is(err, dataoperations.ErrReadOnly) {
		t.Errorf("SetEntryStatus should return ErrReadOnly, got %v", err)
	}

	// Dashboard aggregations are populated.
	ds, _ := r.GetDashboardStats()
	if ds.TotalEntries != int64(len(all)) {
		t.Errorf("dashboard TotalEntries=%d want %d", ds.TotalEntries, len(all))
	}
	if len(ds.TopEntriesByVote) == 0 || len(ds.RecentEntries) == 0 {
		t.Error("dashboard top/recent entries empty")
	}
	if len(ds.EntriesPerDay) != 30 {
		t.Errorf("dashboard EntriesPerDay len=%d want 30", len(ds.EntriesPerDay))
	}

	t.Logf("demo: %d entries, dashboard totals: entries=%d users=%d topics=%d releases=%d votes=%d comments=%d",
		len(all), ds.TotalEntries, ds.TotalUsers, ds.TotalTopics, ds.TotalReleases, ds.TotalVotes, ds.TotalComments)
}

func boolPtr(b bool) *bool { return &b }
