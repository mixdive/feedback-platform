package console

import (
	"errors"

	"github.com/mixdive/feedback-platform/models"
)

// resolveEntryType validates an entry-type value arriving on an entry
// update request. Empty input is allowed — clears the type. Non-empty
// values must match the hardcoded models.EntryType enum; unknown
// values are rejected so the handler maps to a 400.
func resolveEntryType(value string) (models.EntryType, error) {
	t := models.EntryType(value)
	if !models.IsValidEntryType(t) {
		return "", errors.New("Unknown entry type.")
	}
	return t, nil
}
