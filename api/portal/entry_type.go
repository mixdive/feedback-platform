package portal

import (
	"errors"

	"github.com/mixdive/feedback-platform/models"
)

// resolveEntryType validates a entry-type value arriving on a
// portal submit request. Empty input is allowed (no type picked).
// Non-empty values must match the hardcoded models.EntryType enum.
func resolveEntryType(value string) (models.EntryType, error) {
	ft := models.EntryType(value)
	if !models.IsValidEntryType(ft) {
		return "", errors.New("Unknown entry type.")
	}
	return ft, nil
}
