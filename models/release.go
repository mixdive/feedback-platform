package models

import (
	"time"

	"github.com/google/uuid"
)

// ReleaseState is the lifecycle stamp on a Release. Stored kebab-case
// per project convention. Zero value ("") is treated as "planned" by
// the constructor — fresh records always start in the planned state
// regardless of date, and an admin manually flips to completed once
// the work is shipped.
type ReleaseState string

const (
	ReleaseStatePlanned   ReleaseState = "planned"
	ReleaseStateCompleted ReleaseState = "completed"
)

// IsValidReleaseState reports whether v is a known release state.
// Handlers reject unknown values rather than persist them.
func IsValidReleaseState(v ReleaseState) bool {
	switch v {
	case ReleaseStatePlanned, ReleaseStateCompleted:
		return true
	}
	return false
}

// Release groups a set of entries shipped (or planned to ship) under
// a single version. Admin-managed from the Console settings;
// completed releases are surfaced on the Portal Changelog page.
//
// VersionName is the human label admins type ("v1.2.0", "May 2026").
// Title is a one-line headline; Description is a markdown body
// rendered above the linked-entries list on the Portal changelog.
// ReleaseDate is a calendar date (no time) — stored as UTC midnight
// for consistency with the rest of the time fields.
//
// State is one of "planned" | "completed". Transitions are manual
// from the Console; the date does not auto-flip the state.
type Release struct {
	ID          string `bson:"_id"`
	VersionName string
	Title       string
	Description string
	ReleaseDate time.Time
	State       ReleaseState
	// PdfFileURL is a dedicated attachment slot for a single PDF
	// document (release notes, slide deck, ...). It lives outside
	// Description so the Portal can render a stand-alone download
	// link rather than relying on a markdown reference inside the
	// description body. Empty when no PDF is attached.
	//
	// The URL points at the standard /api/files/<key> endpoint; the
	// upload itself goes through the existing UploadFile handler.
	// PdfFileName preserves the uploader's original filename for
	// display (the URL path uses an opaque storage key + extension).
	PdfFileURL  string
	PdfFileName string
	PdfFileSize int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewRelease constructs a Release with a fresh UUID, current
// timestamps, and State defaulted to ReleaseStatePlanned. Caller
// fills the rest.
func NewRelease() *Release {
	now := time.Now().UTC()
	return &Release{
		ID:        uuid.New().String(),
		State:     ReleaseStatePlanned,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
