package console

import (
	"errors"
	"fmt"

	"github.com/mixdive/feedback-platform/dataoperations"
	"github.com/mixdive/feedback-platform/models"
)

// resolveReleaseID validates the releaseId arriving on an entry update
// request. Empty string is allowed — the entry simply has no release.
// A non-empty value must reference an existing release; we return an
// error on an unknown ID so the handler maps it to a 400.
func resolveReleaseID(do *dataoperations.DataOperations, id string) (string, error) {
	if id == "" {
		return "", nil
	}
	r, err := do.FindReleaseByID(id)
	if err != nil {
		return "", err
	}
	if r == nil {
		return "", fmt.Errorf("Unknown release: %s.", id)
	}
	return id, nil
}

// findReleaseOrNotFound is the pattern used by the update/delete
// handlers to look up an existing release and respond with the right
// error class when it doesn't exist. Returns (nil, nil) on a clean
// not-found — the caller is expected to render a 404 in that case.
func findReleaseOrNotFound(do *dataoperations.DataOperations, id string) (*models.Release, error) {
	if id == "" {
		return nil, errors.New("Release id is required.")
	}
	return do.FindReleaseByID(id)
}
