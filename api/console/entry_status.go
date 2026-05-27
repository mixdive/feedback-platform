package console

import (
	"errors"

	"github.com/mixdive/feedback-platform/models"
)

// resolveEntryStatus validates a status value arriving on an entry
// update request. Empty input is rejected — every entry must have a
// status. A non-empty value must be one of the hardcoded
// models.EntryStatus enum values; unknown values are rejected so the
// handler maps to a 400.
func resolveEntryStatus(value string) (models.EntryStatus, error) {
	if value == "" {
		return "", errors.New("Status is required.")
	}
	s := models.EntryStatus(value)
	if !models.IsValidEntryStatus(s) {
		return "", errors.New("Unknown entry status.")
	}
	return s, nil
}
