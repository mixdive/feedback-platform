package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

// filesystem stores each blob as a file under root, named by the blob's id
// (a UUID). No subdirectory sharding — Mongo holds the metadata; the
// directory only has to grow as fast as the upload count, which for the
// pilot is well under any filesystem's per-directory limit.
type filesystem struct {
	root string
}

func newFilesystem(root string) (*filesystem, error) {
	if root == "" {
		return nil, errors.New("storage: filesystem root path is empty")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &filesystem{root: root}, nil
}

func (s *filesystem) path(id string) string {
	return filepath.Join(s.root, id)
}

func (s *filesystem) Put(ctx context.Context, id, mimeType string, size int64, r io.Reader) error {
	tmp, err := os.CreateTemp(s.root, ".upload-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		// If we never renamed, remove the temp file. Best-effort.
		_ = os.Remove(tmpName)
	}()
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.path(id))
}

func (s *filesystem) Open(ctx context.Context, id string) (io.ReadCloser, error) {
	f, err := os.Open(s.path(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

// SignedURL is unsupported on the filesystem backend — the caller falls back
// to Open and streams the response.
func (s *filesystem) SignedURL(ctx context.Context, id string, ttl time.Duration) (string, error) {
	return "", nil
}

func (s *filesystem) Delete(ctx context.Context, id string) error {
	err := os.Remove(s.path(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
