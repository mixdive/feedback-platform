package dataoperations

import (
	"testing"
	"time"
)

// The reconcile pass archives every row in a duplicate group except the one
// pickSurvivingVote points at, so this function decides which vote a user
// keeps. These cases pin that down.
func TestPickSurvivingVote(t *testing.T) {
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		votes []voteRef
		want  string // ID expected to survive, "" for none
	}{
		{
			name:  "empty group keeps nothing",
			votes: nil,
			want:  "",
		},
		{
			name:  "single row survives",
			votes: []voteRef{{ID: "a", CreatedAt: base}},
			want:  "a",
		},
		{
			name: "earliest wins regardless of order",
			votes: []voteRef{
				{ID: "late", CreatedAt: base.Add(2 * time.Hour)},
				{ID: "first", CreatedAt: base},
				{ID: "mid", CreatedAt: base.Add(time.Hour)},
			},
			want: "first",
		},
		{
			name: "identical timestamps break the tie on lowest id",
			votes: []voteRef{
				{ID: "c", CreatedAt: base},
				{ID: "a", CreatedAt: base},
				{ID: "b", CreatedAt: base},
			},
			want: "a",
		},
		{
			name: "a flood of same-instant duplicates still keeps exactly one",
			votes: []voteRef{
				{ID: "z", CreatedAt: base},
				{ID: "y", CreatedAt: base},
				{ID: "x", CreatedAt: base},
				{ID: "w", CreatedAt: base},
			},
			want: "w",
		},
		{
			name: "rows with no id are never chosen",
			votes: []voteRef{
				{ID: "", CreatedAt: base.Add(-time.Hour)},
				{ID: "real", CreatedAt: base},
			},
			want: "real",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pickSurvivingVote(tc.votes)
			if tc.want == "" {
				if got != -1 {
					t.Fatalf("pickSurvivingVote() = %d, want -1 (no survivor)", got)
				}
				return
			}
			if got < 0 || got >= len(tc.votes) {
				t.Fatalf("pickSurvivingVote() = %d, out of range for %d rows", got, len(tc.votes))
			}
			if tc.votes[got].ID != tc.want {
				t.Fatalf("survivor = %q, want %q", tc.votes[got].ID, tc.want)
			}
		})
	}
}

// The choice must not depend on the order Mongo happened to return the
// group in — two instances reconciling the same group concurrently have to
// keep the same row, or they'd archive each other's survivor.
func TestPickSurvivingVoteIsOrderIndependent(t *testing.T) {
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	votes := []voteRef{
		{ID: "b", CreatedAt: base},
		{ID: "a", CreatedAt: base},
		{ID: "c", CreatedAt: base.Add(-time.Second)},
	}
	reversed := []voteRef{votes[2], votes[1], votes[0]}

	if got := votes[pickSurvivingVote(votes)].ID; got != "c" {
		t.Fatalf("forward order survivor = %q, want %q", got, "c")
	}
	if got := reversed[pickSurvivingVote(reversed)].ID; got != "c" {
		t.Fatalf("reversed order survivor = %q, want %q", got, "c")
	}
}

// VoteReconcileReport.Clean gates the log line, so a healthy run has to
// report itself as clean.
func TestVoteReconcileReportClean(t *testing.T) {
	if !(VoteReconcileReport{}).Clean() {
		t.Fatal("zero report should be clean")
	}
	if (VoteReconcileReport{DuplicatesArchived: 1}).Clean() {
		t.Fatal("report with archived duplicates should not be clean")
	}
	if (VoteReconcileReport{EntriesRepaired: 1}).Clean() {
		t.Fatal("report with repaired entries should not be clean")
	}
}
