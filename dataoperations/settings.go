package dataoperations

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mixdive/feedback-platform/models"
	"github.com/mixdive/feedback-platform/pkg/mongodb"
)

// GetSettings returns the singleton settings document, or nil if first-run
// setup has not happened yet.
func (do *DataOperations) GetSettings() (*models.Settings, error) {
	return mongodb.GetOneById[models.Settings](do.DB, CollectionSettings, models.SettingsID)
}

// InsertSettings persists the singleton settings document. Used only by the
// first-run setup handler.
func (do *DataOperations) InsertSettings(s *models.Settings) error {
	return mongodb.InsertOne(do.DB, CollectionSettings, *s)
}

// UpdateSettings applies a sparse patch to the singleton settings document.
// Caller passes a map keyed by lowercase BSON field names.
func (do *DataOperations) UpdateSettings(set map[string]any) error {
	set["updatedat"] = time.Now().UTC()
	for k, v := range set {
		if err := mongodb.SetValue(do.DB, CollectionSettings, models.SettingsID, k, v); err != nil {
			return err
		}
	}
	return nil
}

// RecordAISettingsSuccess stamps Settings.AI.LastAnalyzedAt to now AND
// atomically increments Settings.AI.TotalAnalyzed by 1. Called by the
// dispatcher after every successful analyzer run. Does not touch the
// LastError* fields — the Console UI compares timestamps to decide
// which status line to show.
//
// $inc + $set in two roundtrips is acceptable: TotalAnalyzed is the
// stat that matters for the running counter; LastAnalyzedAt is
// idempotent under racing writes (last-writer-wins is fine for a "when
// did anything succeed last" timestamp).
func (do *DataOperations) RecordAISettingsSuccess() error {
	if err := mongodb.IncrementValue(do.DB, CollectionSettings, models.SettingsID, "ai.totalanalyzed", 1); err != nil {
		return err
	}
	return do.UpdateSettings(map[string]any{
		"ai.lastanalyzedat": time.Now().UTC(),
	})
}

// MaxVotesPerUser returns the per-user vote quota stored on the
// singleton settings document, falling back to the hardcoded default
// when the field is missing or zero (pre-feature deployments before
// the EnsureFeedbackDefaults pass has run, or a freshly-inserted
// settings doc whose feedback sub-document hasn't been seeded yet).
func (do *DataOperations) MaxVotesPerUser() (int, error) {
	s, err := do.GetSettings()
	if err != nil || s == nil {
		if err == nil {
			return models.DefaultMaxVotesPerUser, nil
		}
		return 0, err
	}
	if s.Feedback.MaxVotesPerUser > 0 {
		return s.Feedback.MaxVotesPerUser, nil
	}
	return models.DefaultMaxVotesPerUser, nil
}

// MaxFeatureRequestsPerUser returns the per-user feature-request quota
// stored on the singleton settings document, falling back to the
// hardcoded default when the field is missing or zero. Same fallback
// shape as MaxVotesPerUser.
func (do *DataOperations) MaxFeatureRequestsPerUser() (int, error) {
	s, err := do.GetSettings()
	if err != nil || s == nil {
		if err == nil {
			return models.DefaultMaxFeatureRequestsPerUser, nil
		}
		return 0, err
	}
	if s.Feedback.MaxFeatureRequestsPerUser > 0 {
		return s.Feedback.MaxFeatureRequestsPerUser, nil
	}
	return models.DefaultMaxFeatureRequestsPerUser, nil
}

// EnsureFeedbackDefaults backfills any missing feedback policy fields on
// the singleton settings document. Idempotent — running deployments
// that came up before a feature shipped get its default applied the
// first time the server boots with this code; any admin-set value
// already on the doc is preserved.
//
// Templates: only seeded when EntryTypeTemplatesByLang is nil (legacy
// deployments that never had the field, AFTER the
// MigrateEntryTypeTemplatesToMultiLang pass has lifted any flat
// legacy map into the nested shape). Once the field exists — even if
// the admin has cleared every entry — defaults are NOT restored, so
// admin intent (an explicitly empty template) wins on every
// subsequent boot.
func (do *DataOperations) EnsureFeedbackDefaults() error {
	s, err := do.GetSettings()
	if err != nil || s == nil {
		return err
	}
	patch := map[string]any{}
	if s.Feedback.MaxVotesPerUser <= 0 {
		patch["feedback.maxvotesperuser"] = models.DefaultMaxVotesPerUser
	}
	if s.Feedback.MaxFeatureRequestsPerUser <= 0 {
		patch["feedback.maxfeaturerequestsperuser"] = models.DefaultMaxFeatureRequestsPerUser
	}
	if s.Feedback.EntryTypeTemplatesByLang == nil {
		patch["feedback.entrytypetemplatesbylang"] = models.DefaultEntryTypeTemplates()
	}
	if len(patch) == 0 {
		return nil
	}
	return do.UpdateSettings(patch)
}

// MigrateEntryTypeTemplatesToMultiLang lifts a legacy flat
// FeedbackSettings.entrytypetemplates (Mongo: map[string]string —
// keyed by EntryType value) into the nested
// FeedbackSettings.entrytypetemplatesbylang shape (keyed by EntryType
// then by language code). Each legacy value lands under the
// DefaultTemplateLanguage key; the old field is $unset so only one
// document shape per collection is in flight at any time.
//
// Idempotent: a deploy that already has the nested field set (either
// because this pass ran previously or because the deployment was born
// after the rename) is a no-op. Must run BEFORE EnsureFeedbackDefaults
// at boot so an existing flat map blocks the "seed bundled defaults"
// branch instead of doubling up.
func (do *DataOperations) MigrateEntryTypeTemplatesToMultiLang() error {
	// Pull the raw settings doc so we can inspect both the legacy and
	// the new field by their Mongo names — the typed Settings struct
	// no longer carries the legacy field, so a typed read would lose
	// the values we need to migrate.
	raw, err := mongodb.GetOneById[bson.M](do.DB, CollectionSettings, models.SettingsID)
	if err != nil || raw == nil {
		return err
	}
	feedback, _ := (*raw)["feedback"].(bson.M)
	if feedback == nil {
		return nil
	}
	// If the new nested field is already populated, there is nothing
	// to do. We do not attempt to merge — a populated nested field is
	// authoritative.
	if existing, ok := feedback["entrytypetemplatesbylang"].(bson.M); ok && existing != nil {
		return nil
	}
	legacy, _ := feedback["entrytypetemplates"].(bson.M)
	if legacy == nil {
		// No legacy field either — EnsureFeedbackDefaults will seed
		// the bundled defaults under the nested shape.
		return nil
	}
	lifted := map[string]map[string]string{}
	for entryType, v := range legacy {
		if entryType == "" {
			continue
		}
		s, ok := v.(string)
		if !ok {
			// Unexpected shape — skip rather than corrupt. The
			// EnsureFeedbackDefaults pass will fall back to bundled
			// defaults if the resulting lifted map is empty.
			continue
		}
		lifted[entryType] = map[string]string{models.DefaultTemplateLanguage: s}
	}
	if len(lifted) == 0 {
		// Nothing usable to lift; let EnsureFeedbackDefaults seed
		// bundled defaults. Still unset the legacy field so the
		// document shape collapses to a single canonical form.
		return mongodb.UnsetValue(do.DB, CollectionSettings, models.SettingsID, "feedback.entrytypetemplates")
	}
	if err := do.UpdateSettings(map[string]any{
		"feedback.entrytypetemplatesbylang": lifted,
	}); err != nil {
		return err
	}
	return mongodb.UnsetValue(do.DB, CollectionSettings, models.SettingsID, "feedback.entrytypetemplates")
}

// RecordAISettingsError stamps Settings.AI.LastErrorAt to now and
// LastErrorMessage to msg. Called by the dispatcher after every analyzer
// run that returned an error.
func (do *DataOperations) RecordAISettingsError(msg string) error {
	now := time.Now().UTC()
	return do.UpdateSettings(map[string]any{
		"ai.lasterrorat":      now,
		"ai.lasterrormessage": msg,
	})
}
