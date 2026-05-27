package storage

import (
	"context"
	"errors"
	"io"
	"time"

	"cloud.google.com/go/storage"
)

// gcsStorage stores each blob as an object in a single GCS bucket, keyed by
// its UUID. Used for Cloud Run deployments where the container filesystem is
// ephemeral. Authentication is via Application Default Credentials — Cloud
// Run injects a token from the runtime service account, so no key file or
// extra env var is required.
//
// Reads are served by 302-redirecting the client to a short-lived signed
// URL. That keeps the Cloud Run instance out of the bytes path on a per-file
// basis, which matters for video previews where a single response can
// otherwise saturate an instance.
type gcsStorage struct {
	bucketName string
	client     *storage.Client
}

func newGCS(bucket string) (*gcsStorage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &gcsStorage{bucketName: bucket, client: client}, nil
}

func (s *gcsStorage) bucket() *storage.BucketHandle {
	return s.client.Bucket(s.bucketName)
}

func (s *gcsStorage) Put(ctx context.Context, id, mimeType string, size int64, r io.Reader) error {
	w := s.bucket().Object(id).NewWriter(ctx)
	w.ContentType = mimeType
	if _, err := io.Copy(w, r); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func (s *gcsStorage) Open(ctx context.Context, id string) (io.ReadCloser, error) {
	rc, err := s.bucket().Object(id).NewReader(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return nil, ErrNotFound
	}
	return rc, err
}

// SignedURL produces a v4 signed GET URL valid for ttl. With ADC (no private
// key on disk), the GCS SDK falls back to the IAM Sign Blob API, which
// requires the runtime service account to have iam.serviceAccountTokenCreator
// on itself. Document this in the deployment notes for Cloud Run.
func (s *gcsStorage) SignedURL(ctx context.Context, id string, ttl time.Duration) (string, error) {
	opts := &storage.SignedURLOptions{
		Method:  "GET",
		Expires: time.Now().Add(ttl),
		Scheme:  storage.SigningSchemeV4,
	}
	return s.bucket().SignedURL(id, opts)
}

func (s *gcsStorage) Delete(ctx context.Context, id string) error {
	err := s.bucket().Object(id).Delete(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return nil
	}
	return err
}
