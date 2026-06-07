package demo

import "github.com/mixdive/feedback-platform/models"

// This file holds the demo *content* — the fictional "Contoso" B2B SaaS
// dataset (a project-management / team-collaboration tool). build.go turns
// these definitions into in-memory model documents served by the read-only
// repo.
//
// Everything is keyed by stable local slugs (topic keys, user keys, entry
// keys, release keys). build.go generates one UUID per key up front so
// cross-references (an entry's author, its relations, its votes) resolve
// cleanly without hardcoding UUIDs here.
//
// When you add a product feature, extend the relevant *def struct and the
// content below so the demo keeps exercising the whole surface area. See
// the maintenance checklist at the top of repo.go.

// ---------------------------------------------------------------------
// Builder types (local to the demo package, not persisted directly)
// ---------------------------------------------------------------------

// topicDef is one EntryTopic. aiCreated flips Source to "ai" so the
// Console settings list renders the "AI-created" badge.
type topicDef struct {
	key         string
	title       string
	description string
	color       string
	aiCreated   bool
}

// userDef is one portal end-user (a Contoso customer leaving feedback).
// Admin + editor Console users are created separately in build.go.
type userDef struct {
	key   string
	name  string
	email string
}

// releaseDef is one Release. daysFromNow is signed: negative = shipped in
// the past, positive = planned for the future.
type releaseDef struct {
	key         string
	versionName string
	title       string
	description string
	daysFromNow int
	state       models.ReleaseState
}

// commentDef is one Comment on an entry. isInternal=true keeps it
// Console-only (Portal never renders it). byAdmin routes authorship to a
// Console user instead of a portal end-user.
type commentDef struct {
	authorKey  string
	body       string
	isInternal bool
	byAdmin    bool
	daysAgo    float64
}

// relationDef links the owning entry to peerKey. Define each pair ONCE
// (on either side) — build.go mirrors it onto the peer automatically.
// byAI routes it through the AI provenance subset and stamps reason onto
// the source entry's relation analysis.
type relationDef struct {
	peerKey string
	relType models.EntryRelationType
	byAI    bool
	reason  string
}

// entryDef is one feedback Entry plus everything hanging off it.
//
//   - aiType / aiTypeReason: when aiType is set, a faux entry-type
//     analysis is stamped (Status=done) suggesting that type. The
//     Console derives the "AI suggested" badge from EntryType ==
//     SuggestedEntryTypeID, so set aiType == entryType to show an
//     applied suggestion.
//   - aiTopicKeys / aiTopicReasons: the subset of topicKeys that an AI
//     analyzer applied, with a one-line reason each (keyed by topic key).
//   - mergedIntoKey: marks this entry as merged away. build.go forces its
//     status to cancelled, adds a duplicate relation to the target, and
//     emits a merged-into activity — mirroring the merge handler.
type entryDef struct {
	key         string
	title       string
	description string
	entryType   models.EntryType
	status      models.EntryStatus
	authorKey   string
	topicKeys   []string
	releaseKey  string
	isInternal  bool
	daysAgo     float64
	voterKeys   []string
	comments    []commentDef
	relations   []relationDef

	aiType         models.EntryType
	aiTypeReason   string
	aiTopicKeys    []string
	aiTopicReasons map[string]string

	githubIssue   bool
	mergedIntoKey string
}

// ---------------------------------------------------------------------
// Content
// ---------------------------------------------------------------------

func topicDefs() []topicDef {
	return []topicDef{
		{key: "onboarding", title: "Onboarding", description: "Sign-up, setup wizard, first-run experience.", color: "#0ea5e9"},
		{key: "dashboard", title: "Dashboard", description: "Home dashboard, widgets, and reporting views.", color: "#8b5cf6"},
		{key: "integrations", title: "Integrations", description: "Third-party connections — Slack, Jira, GitHub, Zapier.", color: "#22c55e"},
		{key: "mobile", title: "Mobile App", description: "iOS and Android native apps.", color: "#f97316"},
		{key: "billing", title: "Billing", description: "Plans, invoices, seats, and payment.", color: "#ef4444"},
		// AI-created topic: the topic analyzer spun this up when no
		// existing topic fit an incoming entry. Source=ai → badge.
		{key: "notifications", title: "Notifications", description: "Email, in-app, and push notification preferences.", color: "#eab308", aiCreated: true},
	}
}

func userDefs() []userDef {
	return []userDef{
		{key: "maya", name: "Maya Chen", email: "maya@brightloop.io"},
		{key: "diego", name: "Diego Alvarez", email: "diego@finchpay.com"},
		{key: "priya", name: "Priya Nair", email: "priya@harborhealth.org"},
		{key: "tom", name: "Tom Becker", email: "tom@northwind.co"},
		{key: "aisha", name: "Aisha Rahman", email: "aisha@lumenlabs.dev"},
		{key: "lukas", name: "Lukas Meyer", email: "lukas@vandelay.de"},
		{key: "sara", name: "Sara Okafor", email: "sara@meridian.io"},
		{key: "ken", name: "Ken Tanaka", email: "ken@sakura-soft.jp"},
		{key: "nina", name: "Nina Petrova", email: "nina@orbitcrm.com"},
		{key: "omar", name: "Omar Haddad", email: "omar@cedarworks.co"},
	}
}

func releaseDefs() []releaseDef {
	return []releaseDef{
		{key: "v2_3", versionName: "v2.3", title: "Faster dashboards & saved views", daysFromNow: -58, state: models.ReleaseStateCompleted,
			description: "A performance-focused release. Dashboards now load in under a second on large workspaces, and you can save and share filtered views."},
		{key: "v2_4", versionName: "v2.4", title: "Slack & Jira, done right", daysFromNow: -24, state: models.ReleaseStateCompleted,
			description: "Deeper two-way sync with Slack and Jira, plus a redesigned integrations directory."},
		{key: "v2_5", versionName: "v2.5", title: "Mobile polish & offline mode", daysFromNow: 21, state: models.ReleaseStatePlanned,
			description: "Planned: offline support on mobile, push notification preferences, and a refreshed task detail screen."},
		{key: "v3_0", versionName: "v3.0", title: "Contoso AI & the new home", daysFromNow: 52, state: models.ReleaseStatePlanned,
			description: "Our biggest release yet: AI-assisted planning, a reimagined home dashboard, and a new permissions model."},
	}
}

// entryDefs is the heart of the demo. ~40 entries spanning every type,
// every status, internal vs public, topics, releases, relations, merges,
// AI provenance, and a GitHub link. Keep this list realistic — it ends up
// in marketing screenshots.
func entryDefs() []entryDef {
	return []entryDef{
		// ---- Feature requests --------------------------------------
		{
			key: "dark-mode", title: "Add a dark mode", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusInProgress, authorKey: "maya", topicKeys: []string{"dashboard"},
			releaseKey: "v3_0", daysAgo: 41, voterKeys: []string{"diego", "priya", "tom", "aisha", "lukas", "sara", "ken", "nina"},
			description: "The whole app is blindingly white. A proper dark theme (not just an inverted hack) would save my eyes during late-night planning sessions.",
			aiType:      models.EntryTypeFeatureRequest, aiTypeReason: "Explicitly requests a new UI capability ('add a dark mode').",
			aiTopicKeys: []string{"dashboard"}, aiTopicReasons: map[string]string{"dashboard": "Theming applies primarily to the dashboard and reporting surfaces."},
			comments: []commentDef{
				{authorKey: "diego", body: "+1, the contrast hurts on OLED screens.", daysAgo: 39},
				{authorKey: "tom", body: "Please make it respect the OS-level setting too.", daysAgo: 35},
				{body: "Thanks all — this is now in progress and targeted for v3.0. Early builds respect the system preference.", byAdmin: true, daysAgo: 12},
			},
		},
		{
			key: "saved-views", title: "Save and share filtered views", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusCompleted, authorKey: "sara", topicKeys: []string{"dashboard"}, releaseKey: "v2_3",
			daysAgo: 88, voterKeys: []string{"maya", "diego", "tom", "aisha", "nina", "omar"},
			description: "I rebuild the same filter (assignee = me, due this week) every morning. Let me save it and share it with my team.",
			comments: []commentDef{
				{body: "Shipped in v2.3 🎉 You can save a view from the filter bar and share the link.", byAdmin: true, daysAgo: 58},
				{authorKey: "sara", body: "This is fantastic, thank you!", daysAgo: 57},
			},
		},
		{
			key: "recurring-tasks", title: "Recurring tasks", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusEvaluation, authorKey: "priya", topicKeys: []string{"dashboard"},
			daysAgo: 30, voterKeys: []string{"maya", "tom", "lukas", "ken", "nina"},
			description: "We have standups every weekday and a retro every other Friday. Creating these by hand is tedious — support repeating tasks with a schedule.",
			aiTopicKeys: []string{"dashboard"}, aiTopicReasons: map[string]string{"dashboard": "Task scheduling lives in the main task/dashboard area."},
		},
		{
			key: "bulk-edit", title: "Bulk-edit multiple tasks at once", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusNew, authorKey: "tom", topicKeys: []string{"dashboard"}, daysAgo: 6,
			voterKeys:   []string{"maya", "diego", "sara"},
			description: "Select 20 tasks and change their status or assignee in one go. Right now it's one-by-one.",
		},
		{
			key: "zapier", title: "Native Zapier integration", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusEvaluation, authorKey: "diego", topicKeys: []string{"integrations"}, daysAgo: 23,
			voterKeys:   []string{"maya", "aisha", "nina", "omar", "lukas"},
			description: "We glue a lot of internal tools together with Zapier. A first-class Contoso app in the Zapier directory would unlock a ton of automations.",
			aiType:      models.EntryTypeFeatureRequest, aiTypeReason: "Requests a new third-party integration capability.",
			aiTopicKeys: []string{"integrations"}, aiTopicReasons: map[string]string{"integrations": "Directly about connecting an external automation tool."},
		},
		{
			key: "calendar-sync", title: "Two-way Google Calendar sync", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusNew, authorKey: "nina", topicKeys: []string{"integrations"}, daysAgo: 9,
			voterKeys:   []string{"maya", "diego", "priya"},
			description: "Push task due dates to my Google Calendar and reflect calendar events back as blocked time.",
		},
		{
			key: "offline-mobile", title: "Offline mode on mobile", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusInProgress, authorKey: "ken", topicKeys: []string{"mobile"}, releaseKey: "v2_5",
			daysAgo: 34, voterKeys: []string{"maya", "lukas", "omar", "aisha"},
			description: "I commute through tunnels with no signal. The app should let me read and check off tasks offline and sync when I'm back online.",
			comments: []commentDef{
				{authorKey: "ken", body: "Even read-only offline would be a huge improvement.", daysAgo: 33},
				{body: "Agreed — offline is the headline feature of v2.5. Sync-on-reconnect is the tricky part we're testing now.", byAdmin: true, daysAgo: 8},
			},
		},
		{
			key: "task-templates", title: "Reusable task templates", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusNew, authorKey: "omar", topicKeys: []string{"dashboard"}, daysAgo: 4,
			voterKeys:   []string{"priya", "sara"},
			description: "Onboarding a new client always involves the same 12 tasks. Let me save a template and instantiate it in one click.",
		},
		{
			key: "sso-saml", title: "SAML single sign-on", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusEvaluation, authorKey: "priya", topicKeys: []string{"onboarding", "billing"}, daysAgo: 19,
			voterKeys:   []string{"diego", "lukas", "ken", "nina", "omar", "tom"},
			description: "Our security team requires SAML SSO (Okta) before we can roll Contoso out company-wide. This is a hard blocker for our 200-seat expansion.",
			aiTopicKeys: []string{"onboarding"}, aiTopicReasons: map[string]string{"onboarding": "SSO is part of the account provisioning / sign-in flow."},
		},
		{
			key: "custom-fields", title: "Custom fields on tasks", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusEvaluation, authorKey: "aisha", topicKeys: []string{"dashboard"}, daysAgo: 27,
			voterKeys:   []string{"maya", "diego", "sara", "nina"},
			description: "We track a 'client' and a 'budget' field on every task using the description as a hack. Real custom fields (text, number, dropdown) would clean this up.",
		},
		{
			key: "notification-prefs", title: "Granular notification preferences", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusInProgress, authorKey: "lukas", topicKeys: []string{"notifications"}, releaseKey: "v2_5",
			daysAgo: 26, voterKeys: []string{"maya", "diego", "priya", "tom", "sara"},
			description: "I get an email for every single comment. Let me choose per-project whether I want email, push, in-app, or nothing.",
			aiType:      models.EntryTypeFeatureRequest, aiTypeReason: "Asks for new configurable notification controls.",
			aiTopicKeys: []string{"notifications"}, aiTopicReasons: map[string]string{"notifications": "Entirely about notification channels and frequency — created this topic."},
		},
		{
			key: "api-webhooks", title: "Outbound webhooks", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusNew, authorKey: "aisha", topicKeys: []string{"integrations"}, daysAgo: 11,
			voterKeys:   []string{"diego", "nina", "omar"},
			description: "Fire a webhook when a task changes status so we can react in our own systems. We'd build the rest ourselves if you give us the events.",
			githubIssue: true,
		},
		{
			key: "gantt", title: "Gantt / timeline view", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusNew, authorKey: "tom", topicKeys: []string{"dashboard"}, daysAgo: 14,
			voterKeys:   []string{"maya", "priya", "ken"},
			description: "Kanban is great for execution but I need a timeline view to plan dependencies across a quarter.",
		},
		{
			key: "time-tracking", title: "Built-in time tracking", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusCancelled, authorKey: "omar", topicKeys: []string{"dashboard"}, daysAgo: 52,
			voterKeys:   []string{"diego"},
			description: "Add a start/stop timer on each task so I can bill clients without a separate tool.",
			comments: []commentDef{
				{body: "After looking at this we've decided to stay focused and recommend our Toggl integration instead. Closing as won't-do for now.", byAdmin: true, daysAgo: 20},
			},
		},

		// ---- Bugs --------------------------------------------------
		{
			key: "login-loop", title: "Stuck in a login redirect loop on Safari", entryType: models.EntryTypeBug,
			status: models.EntryStatusCompleted, authorKey: "diego", topicKeys: []string{"onboarding"}, releaseKey: "v2_4",
			daysAgo: 40, voterKeys: []string{"maya", "priya", "tom"},
			description: "On Safari 17, after entering my password I bounce between /login and /dashboard forever. Works fine in Chrome. Clearing cookies doesn't help.",
			aiType:      models.EntryTypeBug, aiTypeReason: "Reports broken sign-in behavior with reproduction details — a defect, not a request.",
			comments: []commentDef{
				{authorKey: "diego", body: "Still happening on Safari 17.2.", daysAgo: 38},
				{body: "Reproduced — it was a SameSite cookie issue. Fix shipped in v2.4. Please hard-refresh.", byAdmin: true, daysAgo: 24},
				{authorKey: "diego", body: "Confirmed fixed, thank you!", daysAgo: 23},
			},
		},
		{
			key: "dup-notifications", title: "Duplicate email notifications", entryType: models.EntryTypeBug,
			status: models.EntryStatusInProgress, authorKey: "sara", topicKeys: []string{"notifications"}, daysAgo: 17,
			voterKeys:   []string{"maya", "diego", "lukas"},
			description: "I receive 2–3 identical emails for a single comment. Started about a week ago.",
			aiTopicKeys: []string{"notifications"}, aiTopicReasons: map[string]string{"notifications": "Describes a malfunction in the email notification pipeline."},
		},
		{
			key: "csv-export-broken", title: "CSV export drops the last column", entryType: models.EntryTypeBug,
			status: models.EntryStatusNew, authorKey: "nina", topicKeys: []string{"dashboard"}, daysAgo: 5,
			voterKeys:   []string{"tom", "omar"},
			description: "Exporting a board to CSV, the 'Due date' column header is present but every value is empty. Reproduces on every board I've tried.",
			aiType:      models.EntryTypeBug, aiTypeReason: "A specific, reproducible data-export defect.",
		},
		{
			key: "mobile-crash", title: "iOS app crashes when attaching a photo", entryType: models.EntryTypeBug,
			status: models.EntryStatusInProgress, authorKey: "ken", topicKeys: []string{"mobile"}, releaseKey: "v2_5",
			daysAgo: 13, voterKeys: []string{"maya", "lukas", "aisha"},
			description: "iPhone 14, iOS 17.3. Tapping 'Attach photo' on a task opens the picker, then the app crashes the instant I select an image.",
			comments: []commentDef{
				{body: "We found the crash (a memory spike on large HEIC files) and have a fix in the v2.5 beta.", byAdmin: true, daysAgo: 7, isInternal: false},
				{body: "Internal: root cause is we decode the full-res image on the main thread. Eng ticket CON-4821.", byAdmin: true, daysAgo: 7, isInternal: true},
			},
		},
		{
			key: "slow-search", title: "Search is very slow on large workspaces", entryType: models.EntryTypeBug,
			status: models.EntryStatusEvaluation, authorKey: "lukas", topicKeys: []string{"dashboard"}, daysAgo: 21,
			voterKeys:   []string{"maya", "diego", "priya", "tom"},
			description: "Searching across our 30k-task workspace takes 8–10 seconds. It used to be instant a couple months ago.",
		},
		{
			key: "timezone-due-dates", title: "Due dates shift by a day for non-US users", entryType: models.EntryTypeBug,
			status: models.EntryStatusNew, authorKey: "lukas", topicKeys: []string{"dashboard"}, daysAgo: 8,
			voterKeys:   []string{"ken", "priya"},
			description: "I'm in CET. A task I set to due 'March 10' shows as 'March 9' to my US colleagues. Looks like dates are stored without a timezone.",
		},
		{
			key: "billing-vat", title: "Invoices missing VAT number", entryType: models.EntryTypeBug,
			status: models.EntryStatusCompleted, authorKey: "lukas", topicKeys: []string{"billing"}, releaseKey: "v2_4",
			daysAgo: 44, voterKeys: []string{"diego"},
			description: "Our finance team can't process Contoso invoices because they don't include our company VAT number, even though I entered it in billing settings.",
			comments: []commentDef{
				{body: "Fixed — VAT numbers now render on the PDF invoice. Re-download last month's from Billing.", byAdmin: true, daysAgo: 24},
			},
		},
		{
			key: "drag-drop-glitch", title: "Drag-and-drop occasionally drops the card", entryType: models.EntryTypeBug,
			status: models.EntryStatusNew, authorKey: "maya", topicKeys: []string{"dashboard"}, daysAgo: 3,
			voterKeys:   []string{"sara"},
			description: "Sometimes when I drag a card between columns it snaps back to the original column. Feels like a race condition under load.",
		},

		// ---- Support -----------------------------------------------
		{
			key: "import-asana", title: "How do I import from Asana?", entryType: models.EntryTypeSupport,
			status: models.EntryStatusCompleted, authorKey: "omar", topicKeys: []string{"onboarding"}, daysAgo: 16,
			voterKeys:   []string{},
			description: "We're switching from Asana. Is there an importer, or do I have to recreate everything by hand?",
			aiType:      models.EntryTypeSupport, aiTypeReason: "A how-to question about using the product, not a defect or request.",
			comments: []commentDef{
				{body: "Yes! Settings → Import → Asana. Paste a personal access token and we'll pull projects, tasks, and assignees. Ping us if anything looks off.", byAdmin: true, daysAgo: 15},
				{authorKey: "omar", body: "Worked perfectly, all 14 projects came across.", daysAgo: 14},
			},
		},
		{
			key: "seat-billing-q", title: "Are guests counted as paid seats?", entryType: models.EntryTypeSupport,
			status: models.EntryStatusCompleted, authorKey: "diego", topicKeys: []string{"billing"}, daysAgo: 12,
			voterKeys:   []string{},
			description: "We want to invite a few external contractors as guests. Will that increase our bill?",
			comments: []commentDef{
				{body: "Guests with view+comment access are free. They only become billable seats if you give them edit access. You can check the breakdown under Billing → Seats.", byAdmin: true, daysAgo: 12},
			},
		},
		{
			key: "delete-workspace", title: "How to permanently delete a workspace?", entryType: models.EntryTypeSupport,
			status: models.EntryStatusNew, authorKey: "tom", topicKeys: []string{"billing"}, daysAgo: 2,
			voterKeys:   []string{},
			description: "We created a test workspace and want it gone, including all data, for compliance reasons.",
		},

		// ---- Untyped / "other" -------------------------------------
		{
			key: "love-it", title: "Just wanted to say the new dashboard is gorgeous", entryType: models.EntryTypeOther,
			status: models.EntryStatusCompleted, authorKey: "maya", topicKeys: []string{"dashboard"}, daysAgo: 50,
			voterKeys:   []string{"diego", "priya", "sara", "nina"},
			description: "The v2.3 dashboard redesign is a huge step up. Whoever did the typography work — chef's kiss.",
			comments: []commentDef{
				{body: "Thank you, this made our day! Sharing with the design team. 💜", byAdmin: true, daysAgo: 49},
			},
		},
		{
			key: "pricing-feedback", title: "The Pro/Business gap is too wide", entryType: "",
			status: models.EntryStatusEvaluation, authorKey: "omar", topicKeys: []string{"billing"}, daysAgo: 18,
			voterKeys:   []string{"nina", "ken"},
			description: "Pro is $10 and Business is $30 with a big jump in features. A middle tier around $18 would fit teams like ours much better.",
		},
		{
			key: "docs-typo", title: "Typo in the API docs auth section", entryType: models.EntryTypeOther,
			status: models.EntryStatusNew, authorKey: "aisha", topicKeys: []string{"integrations"}, daysAgo: 7,
			voterKeys:   []string{},
			description: "Minor: the API docs say 'Barer token' instead of 'Bearer token' in the authentication example.",
		},

		// ---- Internal entries (Console-only) -----------------------
		{
			key: "internal-churn-risk", title: "Harbor Health flagged churn risk — needs SSO", entryType: models.EntryTypeOther,
			status: models.EntryStatusEvaluation, authorKey: "priya", topicKeys: []string{"onboarding"}, isInternal: true, daysAgo: 15,
			voterKeys:   []string{},
			description: "Internal note from the AE: Harbor Health (Track A, healthcare) won't expand without SAML SSO. Tie this to the sso-saml request and prioritize for the renewal in Q3.",
			comments: []commentDef{
				{body: "Renewal is 90 days out. If SSO lands in v3.0 we're safe.", byAdmin: true, isInternal: true, daysAgo: 14},
			},
		},
		{
			key: "internal-competitor", title: "Competitor shipped AI standups — track response", entryType: models.EntryTypeOther,
			status: models.EntryStatusNew, authorKey: "maya", topicKeys: []string{"dashboard"}, isInternal: true, daysAgo: 6,
			voterKeys:   []string{},
			description: "Internal: a competitor launched AI-generated standup summaries this week. Several prospects asked about it on calls. Folding into the v3.0 Contoso AI scope.",
		},

		// ---- Merge demo: duplicate folded into the dark-mode request
		{
			key: "night-theme-dup", title: "Please add a night theme", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusNew, authorKey: "nina", topicKeys: []string{}, daysAgo: 28,
			voterKeys:     []string{"omar"},
			description:   "A dark / night color theme would be great for evening work.",
			mergedIntoKey: "dark-mode",
		},

		// ---- A couple more to round out volume & vote spread -------
		{
			key: "keyboard-shortcuts", title: "More keyboard shortcuts", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusNew, authorKey: "aisha", topicKeys: []string{"dashboard"}, daysAgo: 10,
			voterKeys:   []string{"maya", "lukas"},
			description: "Power users want shortcuts for assign, set due date, and move-to-column without touching the mouse.",
		},
		{
			key: "audit-log", title: "Admin audit log", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusEvaluation, authorKey: "priya", topicKeys: []string{"billing", "onboarding"}, daysAgo: 22,
			voterKeys:   []string{"diego", "lukas", "tom"},
			description: "For our compliance review we need a log of who changed what and when across the workspace — exportable.",
		},
		{
			key: "slack-unfurl", title: "Slack links should unfurl with task details", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusCompleted, authorKey: "diego", topicKeys: []string{"integrations"}, releaseKey: "v2_4",
			daysAgo: 47, voterKeys: []string{"maya", "sara", "nina"},
			description: "When I paste a Contoso task link in Slack it shows a bare URL. It should unfurl into a rich card with title, status, and assignee.",
			comments: []commentDef{
				{body: "Shipped in v2.4 — Contoso task links now unfurl in Slack with a live status badge.", byAdmin: true, daysAgo: 24},
			},
		},
		{
			key: "android-widget", title: "Android home-screen widget", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusNew, authorKey: "ken", topicKeys: []string{"mobile"}, daysAgo: 9,
			voterKeys:   []string{"omar"},
			description: "A widget showing my tasks due today would let me glance without opening the app.",
		},
		{
			key: "comment-reactions", title: "Emoji reactions on comments", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusNew, authorKey: "sara", topicKeys: []string{"dashboard"}, daysAgo: 5,
			voterKeys:   []string{"maya", "diego"},
			description: "Let me 👍 a comment instead of replying 'sounds good' every time.",
		},
		{
			key: "2fa", title: "Two-factor authentication", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusInProgress, authorKey: "priya", topicKeys: []string{"onboarding"}, releaseKey: "v3_0",
			daysAgo: 31, voterKeys: []string{"diego", "lukas", "ken", "nina"},
			description: "TOTP-based 2FA for accounts that don't use SSO. Security baseline for us.",
			aiTopicKeys: []string{"onboarding"}, aiTopicReasons: map[string]string{"onboarding": "Account security sits within the sign-in / onboarding domain."},
		},
		{
			key: "report-pdf", title: "Export reports as PDF", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusNew, authorKey: "tom", topicKeys: []string{"dashboard"}, daysAgo: 12,
			voterKeys:   []string{"omar", "nina"},
			description: "I screenshot the weekly report for my exec deck. A clean PDF export would save me the hassle.",
		},
		{
			key: "mobile-darkmode-bug", title: "Mobile app ignores dark mode on Android 14", entryType: models.EntryTypeBug,
			status: models.EntryStatusNew, authorKey: "ken", topicKeys: []string{"mobile"}, daysAgo: 4,
			voterKeys:   []string{"lukas"},
			description: "My phone is in system dark mode but the Contoso Android app stays white. Other apps switch fine.",
			relations: []relationDef{
				{peerKey: "dark-mode", relType: models.EntryRelationTypeRelated, byAI: true, reason: "Both concern dark theming; this is the mobile manifestation of the dark-mode request."},
			},
		},
		{
			key: "webhook-retries", title: "Retry failed webhook deliveries", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusNew, authorKey: "aisha", topicKeys: []string{"integrations"}, daysAgo: 6,
			voterKeys:   []string{"diego"},
			description: "If our endpoint is briefly down, Contoso should retry the webhook with backoff instead of dropping the event.",
			relations: []relationDef{
				{peerKey: "api-webhooks", relType: models.EntryRelationTypeRelated, byAI: true, reason: "Extends the outbound-webhooks request with delivery-reliability semantics."},
			},
		},
		{
			key: "guest-permissions", title: "Finer guest permissions", entryType: models.EntryTypeFeatureRequest,
			status: models.EntryStatusNew, authorKey: "omar", topicKeys: []string{"billing", "onboarding"}, daysAgo: 8,
			voterKeys:   []string{"nina", "ken"},
			description: "Let me give a guest access to a single project, not the whole workspace.",
		},
	}
}
