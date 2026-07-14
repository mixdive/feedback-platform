package demo

import (
	"time"

	"github.com/google/uuid"

	"github.com/mixdive/feedback-platform/models"
)

// data is the fully-built in-memory dataset the read-only repo serves.
// Built once by buildData() at process start and never mutated thereafter.
type data struct {
	settings   *models.Settings
	users      []models.User
	entries    []models.Entry
	comments   []models.Comment
	votes      []models.Vote
	topics     []models.EntryTopic
	releases   []models.Release
	activities []models.Activity

	// adminUser is the synthetic admin every demo visitor is
	// auto-authenticated as (injected by the demo auth middleware).
	adminUser *models.User
}

// builder resolves the dataset slugs to UUIDs and assembles the
// collections. It mirrors the structure of the previous Mongo seed, but
// every "insert" appends to an in-memory slice instead of hitting the DB,
// so there is no error path — buildData cannot fail.
type builder struct {
	now time.Time

	topicID   map[string]string
	userID    map[string]string
	releaseID map[string]string
	entryID   map[string]string

	adminID  string
	editorID string

	entryStatus map[string]models.EntryStatus

	d *data
}

// buildData constructs the demo dataset. Deterministic except for the
// UUIDs (irrelevant to the demo) and the now-relative timestamps, which
// keep the data looking freshly active every time the demo boots.
func buildData() *data {
	b := &builder{
		now:         time.Now().UTC(),
		topicID:     map[string]string{},
		userID:      map[string]string{},
		releaseID:   map[string]string{},
		entryID:     map[string]string{},
		entryStatus: map[string]models.EntryStatus{},
		d:           &data{},
	}
	b.allocateIDs()
	b.buildSettings()
	b.buildConsoleUsers()
	b.buildPortalUsers()
	b.buildTopics()
	b.buildReleases()
	b.buildEntries()
	return b.d
}

func (b *builder) allocateIDs() {
	for _, t := range topicDefs() {
		b.topicID[t.key] = uuid.New().String()
	}
	for _, u := range userDefs() {
		b.userID[u.key] = uuid.New().String()
	}
	for _, r := range releaseDefs() {
		b.releaseID[r.key] = uuid.New().String()
	}
	for _, e := range entryDefs() {
		b.entryID[e.key] = uuid.New().String()
		b.entryStatus[e.key] = e.status
		if e.mergedIntoKey != "" {
			b.entryStatus[e.key] = models.EntryStatusCancelTarget
		}
	}
}

func (b *builder) daysAgo(d float64) time.Time {
	return b.now.Add(-time.Duration(d * 24 * float64(time.Hour)))
}

// after returns a timestamp a fraction of the way from base to now,
// clamped to stay strictly before now.
func (b *builder) after(base time.Time, frac float64) time.Time {
	t := base.Add(time.Duration(float64(b.now.Sub(base)) * frac))
	if !t.Before(b.now) {
		t = b.now.Add(-time.Hour)
	}
	return t
}

func (b *builder) buildSettings() {
	portal, _ := models.NewPortalSettings()
	s := models.NewSettings()
	s.ProjectName = projectName
	s.PrimaryColor = primaryColor
	s.Portal = portal
	s.Feedback = models.FeedbackSettings{
		EntryTypeTemplatesByLang: models.DefaultEntryTypeTemplates(),
	}
	// AI is reported as off (no API key in a self-contained demo and no
	// worker runs against this store). The AI *badges* are baked onto
	// each entry's analysis sub-docs and render regardless of this flag.
	s.AI = models.AISettings{
		Enabled:        false,
		Model:          aiModelName,
		TotalAnalyzed:  142,
		LastAnalyzedAt: b.daysAgo(0.2),
	}
	s.Integrations = models.IntegrationsSettings{
		GitHub: models.GitHubIntegration{
			Owner:       githubOwner,
			Repo:        githubRepo,
			Token:       githubToken,
			ConnectedAt: b.daysAgo(60),
		},
		Slack: models.SlackIntegration{
			ClientID:        slackClientID,
			ClientSecret:    slackClientSecret,
			WebhookURL:      slackWebhookURL,
			ChannelName:     slackChannelName,
			TeamName:        slackTeamName,
			NotifyOnEntry:   true,
			NotifyOnComment: true,
			NotifyOnVote:    false,
			ConnectedAt:     b.daysAgo(45),
		},
	}
	s.SetupCompleted = true
	b.d.settings = s
}

func (b *builder) buildConsoleUsers() {
	admin := models.NewUser()
	b.adminID = admin.ID
	admin.Roles = []models.UserRole{models.RoleAdmin}
	admin.SetAccount(models.UserAccountTypeEmail, models.UserAccount{
		ID:              AdminEmail,
		Name:            AdminName,
		Email:           AdminEmail,
		IsEmailVerified: true,
		CreatedAt:       b.daysAgo(120),
	})
	admin.CreatedAt = b.daysAgo(120)
	b.d.users = append(b.d.users, *admin)
	// Keep a pointer to the stored copy for auth injection.
	b.d.adminUser = &b.d.users[len(b.d.users)-1]

	editor := models.NewUser()
	b.editorID = editor.ID
	editor.Roles = []models.UserRole{models.RoleEditor}
	editor.SetAccount(models.UserAccountTypeEmail, models.UserAccount{
		ID:              EditorEmail,
		Name:            EditorName,
		Email:           EditorEmail,
		IsEmailVerified: true,
		CreatedAt:       b.daysAgo(110),
	})
	editor.CreatedAt = b.daysAgo(110)
	b.d.users = append(b.d.users, *editor)
}

func (b *builder) buildPortalUsers() {
	for _, ud := range userDefs() {
		u := models.NewUser()
		u.ID = b.userID[ud.key]
		u.SetAccount(models.UserAccountTypeCustom, models.UserAccount{
			ID:        ud.email,
			Name:      ud.name,
			Email:     ud.email,
			CreatedAt: b.daysAgo(100),
		})
		u.CreatedAt = b.daysAgo(100)
		b.d.users = append(b.d.users, *u)
	}
}

func (b *builder) buildTopics() {
	for i, td := range topicDefs() {
		t := models.NewEntryTopic()
		t.ID = b.topicID[td.key]
		t.Title = td.title
		t.Description = td.description
		t.Color = td.color
		t.SortOrder = i
		if td.aiCreated {
			t.Source = models.EntryTopicSourceAI
		}
		t.CreatedAt = b.daysAgo(90)
		t.UpdatedAt = b.daysAgo(90)
		b.d.topics = append(b.d.topics, *t)
	}
}

func (b *builder) buildReleases() {
	for _, rd := range releaseDefs() {
		r := models.NewRelease()
		r.ID = b.releaseID[rd.key]
		r.VersionName = rd.versionName
		r.Title = rd.title
		r.Description = rd.description
		r.State = rd.state
		d := b.now.AddDate(0, 0, rd.daysFromNow)
		r.ReleaseDate = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		created := rd.daysFromNow
		if created > -7 {
			created = -7
		}
		r.CreatedAt = b.now.AddDate(0, 0, created)
		r.UpdatedAt = r.CreatedAt
		b.d.releases = append(b.d.releases, *r)
	}
}

// relationBucket accumulates the mirrored relations for one entry.
type relationBucket struct {
	relations   []models.EntryRelation
	aiRelations []models.EntryRelation
	reasons     map[string]string // peerEntryID -> reason (source side only)
}

func (b *builder) buildEntries() {
	defs := entryDefs()

	// Pass 1: build the mirrored relation graph.
	buckets := map[string]*relationBucket{}
	bucket := func(key string) *relationBucket {
		if buckets[key] == nil {
			buckets[key] = &relationBucket{reasons: map[string]string{}}
		}
		return buckets[key]
	}
	addRel := func(srcKey, dstKey string, t models.EntryRelationType, ai bool, reason string) {
		srcID, dstID := b.entryID[srcKey], b.entryID[dstKey]
		bucket(srcKey).relations = upsertRel(bucket(srcKey).relations, models.EntryRelation{EntryID: dstID, Type: t})
		bucket(dstKey).relations = upsertRel(bucket(dstKey).relations, models.EntryRelation{EntryID: srcID, Type: t})
		if ai {
			bucket(srcKey).aiRelations = upsertRel(bucket(srcKey).aiRelations, models.EntryRelation{EntryID: dstID, Type: t})
			bucket(dstKey).aiRelations = upsertRel(bucket(dstKey).aiRelations, models.EntryRelation{EntryID: srcID, Type: t})
			if reason != "" {
				bucket(srcKey).reasons[dstID] = reason
			}
		}
	}
	for _, e := range defs {
		for _, rel := range e.relations {
			addRel(e.key, rel.peerKey, rel.relType, rel.byAI, rel.reason)
		}
		if e.mergedIntoKey != "" {
			addRel(e.key, e.mergedIntoKey, models.EntryRelationTypeDuplicate, false, "")
		}
	}

	// Pass 2: build entries, comments, votes, activities.
	for _, e := range defs {
		created := b.daysAgo(e.daysAgo)
		entry := models.NewEntry()
		entry.ID = b.entryID[e.key]
		entry.UserID = b.userID[e.authorKey]
		entry.EntryType = e.entryType
		entry.Status = b.entryStatus[e.key]
		entry.Title = e.title
		entry.Description = e.description
		entry.Source = models.EntrySourcePortal
		entry.IsInternal = e.isInternal
		entry.CreatedAt = created
		entry.UpdatedAt = created
		entry.VoteCount = len(e.voterKeys)
		entry.CommentCount = len(e.comments)

		entry.TopicIDs = b.resolveTopicIDs(e.topicKeys)
		entry.AITopicIDs = b.resolveTopicIDs(e.aiTopicKeys)
		if rb := buckets[e.key]; rb != nil {
			if rb.relations != nil {
				entry.Relations = rb.relations
			}
			if rb.aiRelations != nil {
				entry.AIRelations = rb.aiRelations
			}
		}

		b.applyAIAnalysis(entry, e, buckets[e.key])

		if e.githubIssue {
			entry.GitHubIssue = models.GitHubIssue{
				URL:       githubIssueURL,
				Owner:     githubOwner,
				Repo:      githubRepo,
				Number:    312,
				CreatedAt: b.after(created, 0.6),
				CreatedBy: b.adminID,
			}
		}

		b.d.entries = append(b.d.entries, *entry)

		b.buildComments(e)
		b.buildVotes(e, created)
		b.buildActivities(e, *entry, created)
	}
}

const githubIssueURL = "https://github.com/" + githubOwner + "/" + githubRepo + "/issues/312"

func (b *builder) resolveTopicIDs(keys []string) []string {
	ids := make([]string, 0, len(keys))
	for _, k := range keys {
		if id, ok := b.topicID[k]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func (b *builder) applyAIAnalysis(entry *models.Entry, e entryDef, rb *relationBucket) {
	analyzedAt := b.after(entry.CreatedAt, 0.15)

	if e.aiType != "" {
		entry.EntryTypeAnalysis = models.EntryTypeAnalysis{
			Status:                   models.AnalysisStatusDone,
			Model:                    aiModelName,
			AnalyzedAt:               analyzedAt,
			SuggestedEntryTypeID:     string(e.aiType),
			SuggestedEntryTypeReason: e.aiTypeReason,
		}
	}

	if len(e.aiTopicKeys) > 0 {
		reasons := map[string]string{}
		for k, r := range e.aiTopicReasons {
			if id, ok := b.topicID[k]; ok {
				reasons[id] = r
			}
		}
		entry.TopicAnalysis = models.EntryTopicAnalysis{
			Status:                models.AnalysisStatusDone,
			Model:                 aiModelName,
			AnalyzedAt:            analyzedAt,
			SuggestedTopicIDs:     b.resolveTopicIDs(e.aiTopicKeys),
			SuggestedTopicReasons: reasons,
		}
	}

	if rb != nil && len(rb.reasons) > 0 {
		suggested := make([]models.EntryRelation, 0, len(rb.aiRelations))
		for _, rel := range rb.aiRelations {
			if _, ok := rb.reasons[rel.EntryID]; ok {
				suggested = append(suggested, rel)
			}
		}
		entry.RelationAnalysis = models.EntryRelationAnalysis{
			Status:                   models.AnalysisStatusDone,
			Model:                    aiModelName,
			AnalyzedAt:               analyzedAt,
			SuggestedRelations:       suggested,
			SuggestedRelationReasons: rb.reasons,
		}
	}
}

func (b *builder) buildComments(e entryDef) {
	for _, cd := range e.comments {
		c := models.NewComment()
		c.EntryID = b.entryID[e.key]
		if cd.byAdmin {
			c.UserID = b.adminID
		} else {
			c.UserID = b.userID[cd.authorKey]
		}
		c.Body = cd.body
		c.IsInternal = cd.isInternal
		c.CreatedAt = b.daysAgo(cd.daysAgo)
		c.UpdatedAt = c.CreatedAt
		b.d.comments = append(b.d.comments, *c)
	}
}

func (b *builder) buildVotes(e entryDef, entryCreated time.Time) {
	for _, vk := range e.voterKeys {
		v := models.NewVote()
		v.EntryID = b.entryID[e.key]
		v.UserID = b.userID[vk]
		v.CreatedAt = b.after(entryCreated, 0.4)
		b.d.votes = append(b.d.votes, *v)
	}
}

func (b *builder) buildActivities(e entryDef, entry models.Entry, created time.Time) {
	emit := func(t models.ActivityType, src models.ActivitySourceType, actor, from, to, target string, at time.Time) {
		a := models.NewActivity()
		a.EntryID = entry.ID
		a.Type = t
		a.Source = src
		a.ActorID = actor
		a.FromValue = from
		a.ToValue = to
		a.TargetID = target
		a.CreatedAt = at
		b.d.activities = append(b.d.activities, *a)
	}

	emit(models.ActivityTypeEntryCreated, models.ActivitySourceUser, entry.UserID, "", "", "", created)

	if e.aiType != "" {
		emit(models.ActivityTypeEntryTypeChanged, models.ActivitySourceAI, "", "", string(e.aiType), "", b.after(created, 0.15))
	}

	aiTopics := map[string]bool{}
	for _, k := range e.aiTopicKeys {
		aiTopics[k] = true
	}
	for _, k := range e.topicKeys {
		src := models.ActivitySourceAdmin
		actor := b.adminID
		at := b.after(created, 0.3)
		if aiTopics[k] {
			src = models.ActivitySourceAI
			actor = ""
			at = b.after(created, 0.15)
		}
		emit(models.ActivityTypeTopicAdded, src, actor, "", "", b.topicID[k], at)
	}

	if e.releaseKey != "" {
		emit(models.ActivityTypeReleaseSet, models.ActivitySourceAdmin, b.adminID, "", "", b.releaseID[e.releaseKey], b.after(created, 0.5))
	}

	if reasons := b.relationReasonsFor(e); reasons != nil {
		for peerID := range reasons {
			emit(models.ActivityTypeRelationAdded, models.ActivitySourceAI, "", "", "", peerID, b.after(created, 0.2))
		}
	}

	if e.mergedIntoKey != "" {
		emit(models.ActivityTypeMergedInto, models.ActivitySourceAdmin, b.adminID, "", b.titleOf(e.mergedIntoKey), b.entryID[e.mergedIntoKey], b.after(created, 0.6))
	} else if e.status != models.EntryStatusNew {
		emit(models.ActivityTypeStatusChanged, models.ActivitySourceAdmin, b.adminID, string(models.EntryStatusNew), string(e.status), "", b.after(created, 0.55))
	}

	if e.githubIssue {
		emit(models.ActivityTypeGitHubIssueCreated, models.ActivitySourceAdmin, b.adminID, "", "", githubIssueURL, b.after(created, 0.6))
	}
}

func (b *builder) relationReasonsFor(e entryDef) map[string]string {
	out := map[string]string{}
	for _, rel := range e.relations {
		if rel.byAI {
			out[b.entryID[rel.peerKey]] = rel.reason
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (b *builder) titleOf(entryKey string) string {
	for _, e := range entryDefs() {
		if e.key == entryKey {
			return e.title
		}
	}
	return ""
}

// upsertRel appends rel unless a relation to the same peer already exists,
// in which case it overwrites the type — matching the "at most one
// relation per peer" rule on Entry.Relations.
func upsertRel(rels []models.EntryRelation, rel models.EntryRelation) []models.EntryRelation {
	for i := range rels {
		if rels[i].EntryID == rel.EntryID {
			rels[i].Type = rel.Type
			return rels
		}
	}
	return append(rels, rel)
}
