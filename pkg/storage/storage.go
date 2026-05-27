// Package storage abstracts the blob backend that holds uploaded files.
//
// Three deployment scenarios are supported:
//
//   - filesystem: local directory on disk, used both for developer machines
//     and for self-hosted Docker installations where files are persisted on
//     a mounted volume.
//   - gcs: Google Cloud Storage bucket, used for Cloud Run deployments where
//     the container's filesystem is ephemeral. Authentication uses
//     Application Default Credentials (Cloud Run injects these via the
//     runtime service account).
//
// The choice is made at startup by reading the STORAGE_TYPE env var. The
// rest of the codebase only sees the Storage interface and a single
// constructor (New) that returns the configured implementation.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ErrNotFound is returned by Open / SignedURL when the blob does not exist.
// Callers translate it into HTTP 404. Other errors are treated as 5xx.
var ErrNotFound = errors.New("storage: object not found")

// Storage is the contract every backend satisfies. Implementations are not
// required to be concurrency-safe across multiple processes — the assumption
// is that the configured backend is the single writer.
type Storage interface {
	// Put writes the blob identified by id, recording mime type and size as
	// best the backend can. Implementations are free to ignore size when the
	// reader is fully buffered (filesystem) but should honor it when streaming
	// (GCS).
	Put(ctx context.Context, id, mimeType string, size int64, r io.Reader) error

	// Open returns a ReadCloser for the blob's bytes. Used by backends that
	// want the application to stream the response (filesystem). Backends that
	// prefer to redirect (GCS) return an empty ReadCloser and rely on
	// SignedURL.
	Open(ctx context.Context, id string) (io.ReadCloser, error)

	// SignedURL returns a short-lived URL the caller can 302 to. Returns an
	// empty string if the backend does not support signed URLs (filesystem),
	// in which case the caller falls back to Open.
	SignedURL(ctx context.Context, id string, ttl time.Duration) (string, error)

	// Delete removes the blob. Idempotent — deleting a missing blob is not
	// an error.
	Delete(ctx context.Context, id string) error
}

// Config bundles the env-var inputs that drive backend selection. New reads
// these and returns the matching implementation.
type Config struct {
	Type   string // "filesystem" or "gcs"
	Path   string // filesystem only — root directory holding blobs
	Bucket string // gcs only — bucket name
}

// LoadConfig builds a Config from the process environment. STORAGE_TYPE
// defaults to "filesystem"; STORAGE_PATH defaults to "./data/files" so a
// dev run with no env vars Just Works.
func LoadConfig() Config {
	return Config{
		Type:   strings.TrimSpace(strings.ToLower(getEnvDefault("STORAGE_TYPE", "filesystem"))),
		Path:   getEnvDefault("STORAGE_PATH", "./data/files"),
		Bucket: os.Getenv("STORAGE_BUCKET"),
	}
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// New constructs the Storage implementation that matches cfg.Type. Errors
// surface at startup so a misconfigured deployment fails fast instead of
// 500ing the first upload.
func New(cfg Config) (Storage, error) {
	switch cfg.Type {
	case "filesystem", "":
		return newFilesystem(cfg.Path)
	case "gcs":
		if cfg.Bucket == "" {
			return nil, fmt.Errorf("storage: STORAGE_BUCKET is required when STORAGE_TYPE=gcs")
		}
		return newGCS(cfg.Bucket)
	default:
		return nil, fmt.Errorf("storage: unknown STORAGE_TYPE %q (expected filesystem or gcs)", cfg.Type)
	}
}
