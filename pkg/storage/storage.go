// Package storage abstracts the blob backend that holds uploaded files.
//
// Two backends are supported:
//
//   - filesystem: local directory on disk, used both for developer
//     machines and for self-hosted Docker installations where files
//     are persisted on a mounted volume at LocalUploadPath.
//   - gcs: Google Cloud Storage bucket, used for Cloud Run deployments
//     where the container's filesystem is ephemeral. Authentication
//     uses Application Default Credentials (Cloud Run injects these
//     via the runtime service account).
//
// The choice is made by the admin via the Console settings page and
// lives on the singleton settings document. The router holds a single
// *Holder that implements Storage; the holder rebuilds its inner
// backend whenever settings change so toggling backend is hot-swap
// without a restart.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/mixdive/feedback-platform/models"
)

// ErrNotFound is returned by Open / SignedURL when the blob does not
// exist. Callers translate it into HTTP 404. Other errors are treated
// as 5xx.
var ErrNotFound = errors.New("storage: object not found")

// ErrUploadsDisabled is returned by Put when the admin has turned
// uploads off on the settings page. Reads remain available so links
// to previously-uploaded files keep working until those blobs are
// explicitly deleted.
var ErrUploadsDisabled = errors.New("storage: uploads are disabled")

// Storage is the contract every backend satisfies. Implementations
// are not required to be concurrency-safe across multiple processes —
// the assumption is that the configured backend is the single writer.
type Storage interface {
	// Put writes the blob identified by id, recording mime type and
	// size as best the backend can. Implementations are free to
	// ignore size when the reader is fully buffered (filesystem) but
	// should honor it when streaming (GCS).
	Put(ctx context.Context, id, mimeType string, size int64, r io.Reader) error

	// Open returns a ReadCloser for the blob's bytes. Used by
	// backends that want the application to stream the response
	// (filesystem). Backends that prefer to redirect (GCS) return an
	// empty ReadCloser and rely on SignedURL.
	Open(ctx context.Context, id string) (io.ReadCloser, error)

	// SignedURL returns a short-lived URL the caller can 302 to.
	// Returns an empty string if the backend does not support signed
	// URLs (filesystem), in which case the caller falls back to Open.
	SignedURL(ctx context.Context, id string, ttl time.Duration) (string, error)

	// Delete removes the blob. Idempotent — deleting a missing blob
	// is not an error.
	Delete(ctx context.Context, id string) error
}

// Holder is the long-lived Storage handed to handlers. It wraps the
// active backend behind a mutex and rebuilds it in place when the
// admin saves new upload settings. Handlers never reconstruct
// storage; they hold a *Holder that implements Storage and forward
// every call through whatever inner backend is currently active.
type Holder struct {
	mu         sync.RWMutex
	enabled    bool
	backend    Storage
	backendErr error
}

// NewHolder constructs the holder from the settings already on the
// document and returns it ready for the router. Backend construction
// errors do NOT fail boot — they're recorded on the holder and
// returned from Put so the server still starts and the admin can fix
// the misconfiguration from the Console. Logging is the caller's job.
func NewHolder(s models.UploadSettings) *Holder {
	h := &Holder{}
	h.Reload(s)
	return h
}

// Reload swaps the active backend in place. Called by the settings
// update handler after a successful write. On backend-build failure
// the previous backend is retained — admin sees the error and can
// correct the input without the server falling into a broken state.
// Returns the build error (or nil) so the caller can surface it.
func (h *Holder) Reload(s models.UploadSettings) error {
	next, err := buildBackend(s)

	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = s.Enabled
	if err != nil {
		// Keep the previous backend wired for reads; record the
		// build error so future Put calls surface it instead of
		// silently writing through the old backend.
		h.backendErr = err
		return err
	}
	h.backend = next
	h.backendErr = nil
	return nil
}

// LastBuildError is exposed so the Console settings response can
// render the most recent backend-construction failure inline (e.g.
// "GCS client init failed: ...") without re-reading the holder
// elsewhere. Empty string when the current backend is healthy.
func (h *Holder) LastBuildError() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.backendErr == nil {
		return ""
	}
	return h.backendErr.Error()
}

func (h *Holder) Put(ctx context.Context, id, mimeType string, size int64, r io.Reader) error {
	h.mu.RLock()
	enabled := h.enabled
	backend := h.backend
	buildErr := h.backendErr
	h.mu.RUnlock()
	if !enabled {
		return ErrUploadsDisabled
	}
	if buildErr != nil {
		return buildErr
	}
	if backend == nil {
		return ErrUploadsDisabled
	}
	return backend.Put(ctx, id, mimeType, size, r)
}

func (h *Holder) Open(ctx context.Context, id string) (io.ReadCloser, error) {
	h.mu.RLock()
	backend := h.backend
	h.mu.RUnlock()
	if backend == nil {
		return nil, ErrNotFound
	}
	return backend.Open(ctx, id)
}

func (h *Holder) SignedURL(ctx context.Context, id string, ttl time.Duration) (string, error) {
	h.mu.RLock()
	backend := h.backend
	h.mu.RUnlock()
	if backend == nil {
		return "", nil
	}
	return backend.SignedURL(ctx, id, ttl)
}

func (h *Holder) Delete(ctx context.Context, id string) error {
	h.mu.RLock()
	backend := h.backend
	h.mu.RUnlock()
	if backend == nil {
		return nil
	}
	return backend.Delete(ctx, id)
}

// buildBackend constructs the concrete Storage that matches s.
// Empty Backend defaults to Local — matches the zero-value-is-default
// discipline on the model. Disabled settings still build a backend so
// Open / Delete can serve previously-uploaded files.
func buildBackend(s models.UploadSettings) (Storage, error) {
	backend := s.Backend
	if backend == "" {
		backend = models.UploadBackendLocal
	}
	switch backend {
	case models.UploadBackendLocal:
		return newFilesystem(models.LocalUploadPath)
	case models.UploadBackendGCS:
		if s.GCSBucket == "" {
			return nil, fmt.Errorf("storage: GCS bucket is required when backend is gcs")
		}
		return newGCS(s.GCSBucket)
	default:
		return nil, fmt.Errorf("storage: unknown upload backend %q", s.Backend)
	}
}
