package demo

// Demo identity and branding constants for the in-memory demo store.
//
// In demo mode there is no login: a middleware injects the synthetic
// demo admin (see DemoAdmin) on every request, so these "credentials"
// are display values, not secrets. They are logged at startup purely so
// whoever runs the demo knows which identity they are browsing as.
const (
	// AdminEmail / AdminName identify the synthetic admin every demo
	// visitor is auto-authenticated as.
	AdminEmail = "demo@contoso.com"
	AdminName  = "Demo Admin"

	// EditorEmail / EditorName is a second Console user (editor role) so
	// the demo shows the admin-vs-editor distinction in the team list.
	EditorEmail = "editor@contoso.com"
	EditorName  = "Demo Editor"
)

const (
	// projectName is the fictional B2B SaaS the demo feedback is about:
	// a project-management / team-collaboration tool.
	projectName = "Contoso"

	// primaryColor brands the Console + Portal. A confident indigo.
	primaryColor = "#4f46e5"

	// aiModelName is stamped onto every faux analysis sub-doc so the
	// Console AI badges read like a real model produced them. The demo
	// never makes a live LLM call.
	aiModelName = "claude-sonnet-4-6"

	// GitHub integration display values. Token is an obvious placeholder
	// — the Integrations card renders as "connected" so the feature is
	// visible, but nothing is ever sent to GitHub (writes are blocked).
	githubOwner = "contoso"
	githubRepo  = "contoso-app"
	githubToken = "ghp_DEMO_PLACEHOLDER_0000000000000000000000"

	// Slack integration display values. Obvious placeholders — the
	// Integrations card renders as "connected to #product-feedback" so
	// the feature is visible, but nothing is ever posted (writes are
	// blocked, and none of these resolve anyway).
	slackClientID     = "0000000000000.0000000000000"
	slackClientSecret = "DEMO_PLACEHOLDER_slack_client_secret"
	slackWebhookURL   = "https://hooks.slack.com/services/DEMO/PLACEHOLDER/0000000000000000000000"
	slackChannelName  = "#product-feedback"
	slackTeamName     = "Contoso"
)
