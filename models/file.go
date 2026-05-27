package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// File records a single uploaded blob — image, video, or document — that a
// user attached to an entry description or comment body. The blob itself
// lives in the configured storage backend (local filesystem, mounted volume,
// or GCS bucket); this struct is the metadata we keep in Mongo so the
// frontend can render the right markup and we can audit who uploaded what.
//
// ID is a UUID — unguessable enough that we don't gate /api/files/:id
// behind any access check. The on-disk / on-bucket key is composed of
// `<id>.<extension>` so a developer browsing the local data dir can preview
// images with the OS file viewer; the same composite is used in URLs.
//
// Extension is the canonical extension for the sniffed mime type (no
// leading dot) — derived server-side from the allowlist mapping, NOT from
// the uploaded filename, so a user can't disguise a png as .exe.
//
// UserID is empty for anonymous portal uploads (the same shape as anonymous
// entry submission). MimeType is sniffed server-side from the first bytes —
// we don't trust the browser-supplied Content-Type.
type File struct {
	ID           string `bson:"_id"`
	OriginalName string
	Extension    string
	MimeType     string
	Size         int64
	Checksum     string
	UserID       string
	CreatedAt    time.Time
}

// NewFile constructs a File with a fresh ID and the current UTC timestamp.
// Caller fills OriginalName, Extension, MimeType, Size, Checksum, UserID.
func NewFile() *File {
	return &File{
		ID:        uuid.New().String(),
		CreatedAt: time.Now().UTC(),
	}
}

// StorageKey is the object name passed to the storage backend — the same
// string that appears at the tail of /api/files/<key>. Including the
// extension here makes the on-disk file double-clickable and gives signed
// GCS URLs a recognizable suffix.
func (f *File) StorageKey() string {
	if f.Extension == "" {
		return f.ID
	}
	return f.ID + "." + f.Extension
}

// IsImage reports whether the file's mime type is one of the inline-renderable
// image formats. The frontend uses this to decide between `![](url)` and
// `[name](url)` markdown insertion.
func (f *File) IsImage() bool {
	return strings.HasPrefix(f.MimeType, "image/")
}
