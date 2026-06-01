package models

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

// SettingsID is the fixed _id for the singleton settings document. Using a
// known string instead of a UUID lets us read/write the doc without first
// having to look it up.
const SettingsID = "settings"

// JWTPrivateKeyBytes is the size of the random key generated at setup time
// for signing portal JWTs. 32 bytes (256 bits) matches the HS256
// recommendation.
const JWTPrivateKeyBytes = 32

// Settings is the singleton runtime configuration document. Filled by the
// admin via the first-run setup form, then editable from the Console.
//
// Portal-related knobs (custom auth, auth URL, JWT key) live under the
// Portal sub-object so future portal features land in one place rather
// than spreading across the parent.
type Settings struct {
	ID             string `bson:"_id"`
	ProjectName    string
	LogoURL        string
	PrimaryColor   string
	Portal         PortalSettings
	AI             AISettings
	Feedback       FeedbackSettings
	Uploads        UploadSettings
	Integrations   IntegrationsSettings
	SetupCompleted bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UploadBackend names the blob backend that holds uploaded files. Empty
// reads as Local — matches the zero-value-is-default discipline used by
// the other enums on this model.
type UploadBackend string

const (
	UploadBackendLocal UploadBackend = "local"
	UploadBackendGCS   UploadBackend = "gcs"
)

// LocalUploadPath is the on-disk root used when UploadBackend is
// Local. Hardcoded so admins control where files land via a volume
// mount at this path rather than via a settings field that's trivial
// to misconfigure. Resolved relative to the server's working
// directory; Docker / Cloud Run images should mount persistent
// storage here.
const LocalUploadPath = "./data/files"

// UploadSettings controls whether file uploads are accepted and which
// backend stores the blobs. Lives on the Settings document next to the
// other sub-objects even though the admin UI renders it on the Feedback
// settings page — uploads aren't a feedback knob conceptually, they
// just share a UI surface.
//
// Zero value ({Enabled: false, Backend: "", GCSBucket: ""}) is the safe
// default. Fresh deploys start with uploads off; the admin opts in.
//
// Backend transitions are hot-swapped: the settings update handler
// rebuilds the active storage backend in-process after a successful
// write. Files uploaded under a previous backend keep their original
// StorageKey and only resolve when that backend is active again.
//
// GCS authentication uses Application Default Credentials (Cloud Run /
// GCE / GKE inject these via the runtime service account). Bucket is
// the only knob the admin needs to provide; non-GCP self-hosters who
// want a GCS bucket can run with local storage instead.
type UploadSettings struct {
	Enabled   bool
	Backend   UploadBackend
	GCSBucket string
}

// DefaultMaxVotesPerUser is the default per-user vote quota seeded into
// FeedbackSettings on first-run setup and used as the fallback when the
// stored value is zero (pre-feature deployments that haven't gone
// through the startup ensure-pass yet).
const DefaultMaxVotesPerUser = 20

// DefaultMaxFeatureRequestsPerUser caps the number of OPEN feature
// requests a single user can have at once. Seeded into FeedbackSettings
// on first-run setup and used as the fallback when the stored value is
// zero — same backfill-by-fallback pattern as DefaultMaxVotesPerUser.
const DefaultMaxFeatureRequestsPerUser = 10

// DefaultTemplateLanguage is the canonical language for entry-type
// templates. New deployments get this language seeded with bundled
// copy; the Portal falls back to this language at render time when
// the user's active language has no template configured for the
// chosen entry type.
const DefaultTemplateLanguage = "en"

// Default templates seeded into FeedbackSettings.EntryTypeTemplatesByLang
// on first-run setup and backfilled by EnsureFeedbackDefaults on running
// deployments that came up before the templates feature shipped. Empty
// string = no template — the Portal description field renders the
// per-type placeholder as before.
const DefaultFeatureRequestTemplate = `## Problem
What problem are you trying to solve? Who is affected, and how often does it come up?

## Proposed solution
What would you like to see? Sketch the behavior; screenshots or mockups welcome.

## Alternatives considered
Workarounds you've tried or other approaches you've thought about.

## Additional context
Links, examples, or anything else that would help us understand.`

const DefaultBugTemplate = `## What happened
A clear description of the bug.

## Steps to reproduce
1.
2.
3.

## Expected behavior
What you expected to happen.

## Actual behavior
What actually happened. Attach screenshots, screen recordings, or error messages if possible.

## Environment
- Browser / OS / device:
- App version:
- URL where it happened:`

// Turkish counterparts of the bundled defaults. Shipped so deployments
// whose Portal users select Turkish from the language picker get a
// usable template out of the box rather than the English fallback.
const DefaultFeatureRequestTemplateTR = `## Sorun
Çözmek istediğiniz sorun nedir? Kimleri etkiliyor ve ne sıklıkla yaşanıyor?

## Önerilen çözüm
Görmek istediğiniz davranış nedir? Ekran görüntüleri veya taslaklar memnuniyetle kabul edilir.

## Değerlendirilen alternatifler
Denediğiniz geçici çözümler veya düşündüğünüz diğer yaklaşımlar.

## Ek bağlam
Anlamamıza yardımcı olacak bağlantılar, örnekler veya başka her şey.`

const DefaultBugTemplateTR = `## Ne oldu
Hatanın net bir açıklaması.

## Adım adım nasıl tekrarlanır
1.
2.
3.

## Beklenen davranış
Olmasını beklediğiniz şey.

## Gerçekleşen davranış
Aslında ne oldu? Mümkünse ekran görüntüleri, ekran kayıtları veya hata mesajları ekleyin.

## Ortam
- Tarayıcı / işletim sistemi / cihaz:
- Uygulama sürümü:
- Olayın yaşandığı URL:`

// FeedbackSettings groups every feedback-domain knob that lives on the
// singleton Settings document. Keeping them under one sub-object keeps
// future feedback policy (per-user comment quotas, vote weighting, …)
// landing here instead of flattening more fields onto the parent.
//
// MaxVotesPerUser caps the number of votes a single user can have
// allocated to OPEN entries at once. Votes on closed entries
// (completed / cancelled) don't count against the cap; when an entry
// transitions from open to closed every voter's quota is refunded by 1.
//
// MaxFeatureRequestsPerUser caps the number of OPEN entries with
// category=feature-request a single user can have authored at once.
// Mirrors the votes cap: closed (completed / cancelled) feature
// requests don't count against the cap; on an open→closed transition
// the author's quota is refunded by 1.
//
// EntryTypeTemplatesByLang is the admin-managed markdown template per
// (entry type, language) pair, used by the Portal to pre-fill the
// description field on the new-entry form. Outer key is EntryType
// (kebab-case — "feature-request", "bug"); inner key is a BCP-47
// language code matching the Portal's SUPPORTED_LANGUAGES set ("en",
// "tr"). Empty inner value = no template for that pair, and at render
// time the Portal falls back to the DefaultTemplateLanguage entry
// before giving up to the per-type placeholder copy.
//
// A nil outer map is the pre-feature legacy state and triggers a
// one-shot seed via EnsureFeedbackDefaults; once the map exists, even
// an admin who has cleared every entry will not have defaults
// restored.
//
// The Go field was renamed from EntryTypeTemplates (flat
// map[string]string) on the multi-language rollout. The BSON key
// changed in lockstep (entrytypetemplates → entrytypetemplatesbylang)
// so legacy flat documents decode silently as nil on first boot under
// the new code; MigrateEntryTypeTemplatesToMultiLang then lifts each
// legacy value into {"en": v} under the new key and $unsets the old.
type FeedbackSettings struct {
	MaxVotesPerUser           int
	MaxFeatureRequestsPerUser int
	EntryTypeTemplatesByLang  map[string]map[string]string
	SupportRequest            SupportRequestSettings
}

// SupportRequestSettings controls the Portal "New Support Request"
// button. Support entries themselves remain a valid EntryType — they
// just are no longer created from the Portal. When Enabled is true,
// the Portal renders the button as a link that opens URL in a new
// tab; when false the button is hidden entirely. Zero value
// ({Enabled: false, URL: ""}) is the safe default — admins opt in
// explicitly from the Console feedback-settings page.
type SupportRequestSettings struct {
	Enabled bool
	URL     string
}

// DefaultEntryTypeTemplates returns a fresh nested map of the bundled
// default templates, keyed first by entry type then by language code.
// Allocated on every call so callers can mutate the result without
// affecting the source of truth.
func DefaultEntryTypeTemplates() map[string]map[string]string {
	return map[string]map[string]string{
		string(EntryTypeFeatureRequest): {
			"en": DefaultFeatureRequestTemplate,
			"tr": DefaultFeatureRequestTemplateTR,
		},
		string(EntryTypeBug): {
			"en": DefaultBugTemplate,
			"tr": DefaultBugTemplateTR,
		},
	}
}

// NewSettings constructs a Settings document with the fixed singleton ID
// and current timestamps. Every other field is zero-valued for the caller
// to fill in.
func NewSettings() *Settings {
	now := time.Now().UTC()
	return &Settings{
		ID:        SettingsID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// PortalSettings groups every portal-facing knob that lives on the singleton
// Settings document. Keeping them under one sub-object means any future
// portal feature (custom domain, branding override, …) lands here without
// flattening more fields onto the parent.
type PortalSettings struct {
	CustomAuthEnabled    bool
	AuthURL              string
	CustomAuthButtonText string
	JWTPrivateKey        string
}

// NewPortalSettings constructs a default PortalSettings with a freshly
// generated JWT signing key. The key is generated unconditionally at setup
// so it is available the moment custom auth is toggled on; no key rotation
// flow is provided in v0.1.
func NewPortalSettings() (PortalSettings, error) {
	key, err := generateJWTPrivateKey()
	if err != nil {
		return PortalSettings{}, err
	}
	return PortalSettings{
		CustomAuthEnabled: false,
		AuthURL:           "",
		JWTPrivateKey:     key,
	}, nil
}

func generateJWTPrivateKey() (string, error) {
	b := make([]byte, JWTPrivateKeyBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// IntegrationsSettings groups every third-party integration on the
// singleton settings document. Adding a new integration lands a sibling
// field here (Slack, Linear, …) rather than flattening more knobs onto
// the parent. Zero value is the safe "no integrations configured" state.
type IntegrationsSettings struct {
	GitHub GitHubIntegration
}

// GitHubIntegration is the single-repo connection to GitHub. Single-tenant
// per deployment, so one owner/repo is enough — multi-repo waits until a
// real customer asks. Token is a Personal Access Token with issue write
// access; stored cleartext for the same reason AISettings.APIKey is (the
// "only MONGO_URI" rule rules out app-level encryption).
//
// Disabled follows the inverse-bool pattern so the zero value
// ({Disabled: false, Owner: "", Repo: "", Token: ""}) reads as
// "enabled but unconfigured" — the create-issue button stays hidden
// until Owner/Repo/Token are non-empty, regardless of Disabled, so
// the zero state is harmless. Disabled is a separate kill-switch for
// admins who want to keep the credentials around while temporarily
// turning the feature off.
//
// ConnectedAt / ConnectedBy record when and by whom the credentials were
// last set, surfaced on the Console Integrations card. Token is never
// returned on the wire — the GET handler exposes only HasToken plus a
// short prefix preview.
type GitHubIntegration struct {
	Disabled    bool
	Owner       string
	Repo        string
	Token       string
	ConnectedAt time.Time
	ConnectedBy string
}

// IsGitHubIntegrationActive reports whether the GitHub integration is
// fully wired and not kill-switched. Used by handlers to gate the
// create-issue action and by the Entry response builders to decide
// whether to surface the "Create on GitHub" affordance.
func IsGitHubIntegrationActive(g GitHubIntegration) bool {
	return !g.Disabled && g.Owner != "" && g.Repo != "" && g.Token != ""
}

// AISettings groups every AI-analyzer knob on the singleton settings
// document. Default zero value ({Enabled: false, APIKey: "", Model: ""})
// is the safe "AI off" state; admins opt in from the Console settings
// page after first-run setup.
//
// APIKey is stored cleartext in Mongo. Single-tenant per deployment on
// the customer's own infrastructure makes app-level encryption a poor
// trade — it would require a second key-management story that violates
// the "only MONGO_URI" configuration rule.
//
// LastAnalyzedAt / LastErrorAt / LastErrorMessage drive the live status
// line on the Console AI settings page. The dispatcher updates them
// after every analyzer run.
type AISettings struct {
	Enabled          bool
	APIKey           string
	Model            string
	LastAnalyzedAt   time.Time
	LastErrorAt      time.Time
	LastErrorMessage string
	// TotalAnalyzed is a monotonically-increasing counter of successful
	// analyzer runs across this deployment's lifetime. Incremented
	// atomically (Mongo $inc) by the dispatcher after every successful
	// Process call. Doesn't reset when AI is toggled off; doesn't
	// decrement when entries are deleted — strictly a "how many LLM
	// calls have we successfully completed" running total.
	TotalAnalyzed int
}
