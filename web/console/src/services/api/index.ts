// Hand-typed API client for the Console.
//
// Authentication is via the HTTPOnly mixdive_session cookie set by /api/login.
// Every request runs with credentials: 'include' so the cookie travels both
// in dev (Vite proxy on a different origin) and in production.

export type ApiEntryCreator = {
  id: string
  name?: string
  username?: string
  imageUrl?: string
}

export type ApiEntrySource = 'portal' | 'console'

// EntryType is a hardcoded enum on the backend (see models/entry.go).
// The wire shape carries the value plus its display metadata so the
// chip on every entry card renders without a separate lookup.
// Frontends keep an identical list in src/utils/entry-type.ts.
export type ApiEntryTypeValue =
  | 'feature-request'
  | 'bug'
  | 'support'
  | 'other'

export type ApiEntryType = {
  value: ApiEntryTypeValue
  title: string
  color?: string
  icon?: string
}

// EntryStatus is a hardcoded enum on the backend (see
// models/entry.go). The wire shape carries the value plus its display
// metadata so the chip on every entry card renders without a separate
// lookup. Frontends keep an identical list in src/lib/entry-status.ts —
// see that file when adding or renaming statuses.
export type ApiEntryStatusValue =
  | 'new'
  | 'evaluation'
  | 'in-progress'
  | 'completed'
  | 'cancelled'

export type ApiEntryStatus = {
  value: ApiEntryStatusValue
  title: string
  color?: string
  icon?: string
}

// Topics group entries by area of the customer product (sign up,
// feed, settings…). Console-only; never rendered on the Portal.
// Source records who created the topic — admins create them by
// hand from the Topics settings page; the AI analyzer creates them
// on demand when no existing topic fits a fresh entry.
export type ApiEntryTopicSource = 'admin' | 'ai' | (string & {})

export type ApiEntryTopic = {
  id: string
  title: string
  description?: string
  color?: string
  sortOrder: number
  source: ApiEntryTopicSource
}

// Per-entry-type breakdown rendered alongside a parent count on the
// Topics and Releases list pages. Total = featureRequest + bug +
// support + other; the "other" bucket also absorbs untyped entries.
export type ApiEntryTypeCounts = {
  total: number
  featureRequest: number
  bug: number
  support: number
  other: number
}

export type ApiEntryTopicListItem = ApiEntryTopic & {
  entryCount: number
  entryTypeCounts: ApiEntryTypeCounts
}
export type ApiEntryTopicListResponse = { data: ApiEntryTopicListItem[] }

// Releases group entries shipped (or planned to ship) under a single
// version. Admin-managed from the Console settings; admins also pick a
// release on the entry edit form to attach the version label to a
// specific entry. Completed releases additionally surface on the
// Portal Changelog.
//
// State is "planned" or "completed" — manual transitions only. The
// release date is a calendar date (YYYY-MM-DD), no time.
export type ApiReleaseState = 'planned' | 'completed' | (string & {})

export type ApiRelease = {
  id: string
  versionName: string
  title?: string
  description?: string
  releaseDate?: string
  state: ApiReleaseState
  pdfFileUrl?: string
  pdfFileName?: string
  pdfFileSize?: number
}

export type ApiReleaseListItem = ApiRelease & {
  entryCount: number
  entryTypeCounts: ApiEntryTypeCounts
}
export type ApiReleaseListResponse = { data: ApiReleaseListItem[] }

export type ApiReleaseEntrySummary = {
  id: string
  title: string
  isInternal: boolean
}

export type ApiReleaseDetail = ApiRelease & {
  entries: ApiReleaseEntrySummary[]
}

// Relations are admin-managed entry-to-entry links rendered only on
// the Console — they never appear on the Portal. The link is symmetric:
// every relation lives on both peers' entry records. Two types are
// recognised in v0.1:
//
//   - duplicate: same problem/request as the peer.
//   - related:   catch-all for non-duplicate similarity.
//
// AI provenance is on each relation (isAI), not on the entry as a
// whole — an entry's relations array can mix admin-applied and
// AI-applied links.
export type ApiEntryRelationType = 'duplicate' | 'related' | (string & {})

export type ApiEntryRelationPeer = {
  id: string
  title: string
  voteCount: number
  isInternal: boolean
}

export type ApiEntryRelation = {
  type: ApiEntryRelationType
  isAI: boolean
  peer: ApiEntryRelationPeer
}

// GitHub issue linked to an entry. Populated after an admin clicks
// "Create GitHub issue" on a feature-request or bug — the AI
// summarizer drafts the issue body, pkg/github posts it, and the
// resulting link is persisted on the entry. Undefined when no issue
// has been created yet. Once set, the link is forever in v0.1 — there
// is no unlink path.
export type ApiGitHubIssue = {
  url: string
  number: number
  createdAt?: string
}

export type ApiEntry = {
  id: string
  title: string
  description?: string
  voteCount: number
  isVoted: boolean
  commentCount: number
  source: ApiEntrySource
  isInternal: boolean
  creator?: ApiEntryCreator
  entryType?: ApiEntryType
  // True when the current entry type was applied by the AI analyzer
  // (derived server-side from
  // EntryTypeAnalysis.SuggestedEntryTypeID matching the current type).
  // Drives the AI badge on the entry-type chip.
  entryTypeAppliedByAI: boolean
  // AI-suggested entry type (if the analyzer ran). Drives the "Apply
  // suggestion" badge on the Console detail sidebar.
  suggestedEntryType?: ApiEntryType
  suggestedEntryTypeReason?: string
  status: ApiEntryStatus
  topics: ApiEntryTopic[]
  // Subset of topic IDs in `topics` that the AI analyzer assigned.
  // Distinct from ApiEntryTopic.source, which marks topics whose
  // record was created by AI — a topic can be admin-created but
  // AI-assigned, or the reverse.
  aiTopicIds: string[]
  // Console-only entry-to-entry links. Populated on detail responses
  // (GET /entry/:id and the responses returned from any mutating
  // endpoint); list responses leave this as an empty array since
  // list cells don't surface relations.
  relations: ApiEntryRelation[]
  // Cross-surface — also rendered on the Portal entry. Undefined when
  // the entry isn't assigned to a release.
  release?: ApiRelease
  // Console-only: link to the GitHub issue an admin published from
  // this entry. Undefined when no issue has been created yet.
  githubIssue?: ApiGitHubIssue
  createdAt: string
  updatedAt: string
}

export type ApiComment = {
  id: string
  entryId: string
  body: string
  isInternal: boolean
  author?: ApiEntryCreator
  createdAt: string
  updatedAt: string
}

export type ApiCommentListResponse = { data: ApiComment[] }

// Activity is the per-entry audit-log row rendered on the Console
// timeline. `source` discriminates user / admin / ai; `actor` is set
// only when the row has a known acting user (AI rows always omit it).
// `fromValue` / `toValue` carry the before/after snapshot for
// transitions; `targetId` identifies the related entity (topic id,
// release id, peer entry id, merge target).
export type ApiActivityType =
  | 'entry-created'
  | 'status-changed'
  | 'entry-type-changed'
  | 'topic-added'
  | 'topic-removed'
  | 'release-set'
  | 'release-cleared'
  | 'relation-added'
  | 'relation-removed'
  | 'internal-enabled'
  | 'internal-disabled'
  | 'merged-into'
  | 'github-issue-created'

export type ApiActivitySource = 'user' | 'admin' | 'ai'

export type ApiActivity = {
  id: string
  entryId: string
  type: ApiActivityType
  source: ApiActivitySource
  actor?: ApiEntryCreator
  fromValue?: string
  toValue?: string
  targetId?: string
  createdAt: string
}

export type ApiActivityListResponse = { data: ApiActivity[] }

export type ApiListMeta = { total: number; page: number; limit: number; hasMore: boolean }
export type ApiEntryListResponse = { data: ApiEntry[]; meta: ApiListMeta }

export type ApiEntryAuthorListResponse = { data: ApiEntryCreator[] }

export type ApiPortalSettings = {
  customAuthEnabled: boolean
  authUrl: string
  customAuthButtonText: string
  // Server-generated at setup time. Read-only on the wire — the PATCH
  // endpoint refuses changes to it. Only sent to admins; editors and
  // other roles get an undefined value here.
  jwtPrivateKey?: string
}

// AI settings live on the singleton settings document. The cleartext
// API key never travels to the browser — only `hasApiKey` and a masked
// `apiKeyPreview` ("•••• abcd") are exposed. To set or rotate the key,
// PATCH /api/console/settings/ai with `apiKey: "<new>"`; to clear it,
// send `apiKey: ""`.
export type ApiAISettings = {
  enabled: boolean
  hasApiKey: boolean
  apiKeyPreview?: string
  model: string
  lastAnalyzedAt?: string
  lastErrorAt?: string
  lastErrorMessage?: string
}

// Patch shape for the AI sub-block. Sparse — only present fields are
// applied. Backend rejects `enabled: true` when no API key is set.
export type ApiAISettingsPatch = {
  enabled?: boolean
  apiKey?: string
  model?: string
}

// Cluster-wide queue picture returned by GET /api/console/ai/queue.
// Pending = entries waiting for analysis (not currently claimed).
// InFlight = entries currently being analyzed by some instance.
// TotalAnalyzed = cumulative successful analyses over the deployment's
// lifetime (monotonic; doesn't reset).
export type ApiAIQueueStats = {
  enabled: boolean
  pollInterval: string
  claimTtl: string
  pendingByName: Record<string, number>
  inFlightByName: Record<string, number>
  totalPending: number
  totalInFlight: number
  totalAnalyzed: number
}

// Feedback policy lives on the singleton settings document. v0.1 ships
// a single knob — the per-user vote quota — but the sub-object exists
// so future feedback rules (per-user comment caps, weighted votes, …)
// land in one place rather than flattening more fields onto the parent.
//
// entryTypeTemplates is the admin-managed markdown template per entry
// type, used by the Portal to pre-fill the description field on the
// new-entry form. Keys are EntryType values; empty string = no
// template for that type.
export type ApiSupportRequestSettings = {
  enabled: boolean
  url: string
}

export type ApiFeedbackSettings = {
  maxVotesPerUser: number
  maxFeatureRequestsPerUser: number
  entryTypeTemplates: Record<string, string>
  // Bundled starter templates per entry type, static on every
  // response. The "Reset to default" button in the Console template
  // editor pulls from here so admins can restore the original copy
  // without a second round trip.
  defaultEntryTypeTemplates: Record<string, string>
  // Controls the Portal "New Support Request" button. Enabled+URL
  // pair behaves like the Portal customAuth pair — enabling the
  // button requires a valid absolute URL.
  supportRequest: ApiSupportRequestSettings
}

// Third-party integration state lives on the singleton settings
// document under the Integrations sub-block. Today GitHub is the only
// entry; future integrations land as sibling fields.
//
// The cleartext token never travels on the wire — only `hasToken` plus
// a masked `tokenPreview` ("•••• abcd"). `active` is derived
// server-side (= !disabled && owner && repo && token) so the UI
// doesn't have to repeat the rule.
export type ApiGitHubIntegration = {
  disabled: boolean
  owner?: string
  repo?: string
  hasToken: boolean
  tokenPreview?: string
  connectedAt?: string
  connectedBy?: string
  active: boolean
}

export type ApiIntegrationsSettings = {
  github: ApiGitHubIntegration
}

// Backend storing uploaded files. "local" writes to the on-disk
// LocalUploadPath directory (volume-mount in Docker / Cloud Run);
// "gcs" writes to a Google Cloud Storage bucket using Application
// Default Credentials.
export type ApiUploadBackend = 'local' | 'gcs'

// Admin-facing projection of the upload settings sub-block.
// lastError surfaces the most recent backend-construction failure so
// the admin can correct it from the same page. Empty when the active
// backend is healthy or uploads are disabled.
export type ApiUploadSettings = {
  enabled: boolean
  backend?: ApiUploadBackend
  gcsBucket?: string
  lastError?: string
}

export type ApiSettings = {
  projectName: string
  logoUrl?: string
  primaryColor?: string
  portal: ApiPortalSettings
  ai: ApiAISettings
  feedback: ApiFeedbackSettings
  uploads: ApiUploadSettings
  integrations: ApiIntegrationsSettings
}

// Patch shape: every leaf is optional. Nested portal/feedback patches
// are also sparse.
export type ApiSettingsPatch = {
  projectName?: string
  logoUrl?: string
  primaryColor?: string
  portal?: Partial<
    Pick<
      ApiPortalSettings,
      'customAuthEnabled' | 'authUrl' | 'customAuthButtonText'
    >
  >
  feedback?: Partial<Omit<ApiFeedbackSettings, 'supportRequest'>> & {
    // When present, replaces the entire templates map. Omit the field
    // to leave templates untouched.
    entryTypeTemplates?: Record<string, string>
    // Sparse: omit a leaf to leave it untouched. Flipping enabled to
    // true requires a usable url; the server validates this. Omit
    // from the parent Partial so this Partial<> override actually
    // wins the intersection.
    supportRequest?: Partial<ApiSupportRequestSettings>
  }
  // Sparse: omit a leaf to leave it untouched. Enabling uploads
  // requires a backend; selecting "gcs" requires a non-empty bucket.
  // The server validates the merged state.
  uploads?: Partial<Omit<ApiUploadSettings, 'lastError'>>
}

export type ApiUser = {
  id: string
  email: string
  roles: string[]
}

export type ApiMyProfile = {
  id: string
  email?: string
  name?: string
  username?: string
  imageUrl?: string
  hasPasswordAccount: boolean
}

export type ApiMyProfilePatch = {
  name?: string
  imageUrl?: string
}

export type ApiMyPasswordPatch = {
  currentPassword: string
  newPassword: string
}

export type ApiAdministrator = {
  id: string
  email: string
  name?: string
  username?: string
  imageUrl?: string
  roles: string[]
  accounts: ApiAccountType[]
  isBlocked: boolean
  createdAt: string
}

export type ApiAdministratorListResponse = { data: ApiAdministrator[] }

export type ApiAccountType = 'email' | 'custom' | 'google'

export type ApiUserRow = {
  id: string
  email: string
  name?: string
  username?: string
  imageUrl?: string
  roles: string[]
  accounts: ApiAccountType[]
  isBlocked: boolean
  createdAt: string
  entryCount: number
  featureRequestCount: number
  bugCount: number
  supportCount: number
  otherCount: number
}

export type ApiUserRole = 'all' | 'admin' | 'editor' | 'none'
export type ApiUserStatus = 'all' | 'active' | 'blocked'
// new/old/email are legacy modes kept for the Administrators add-member
// search dialog; alpha/created/entries/feature-requests/bugs/support/others
// power the dedicated Users page.
export type ApiUserSort =
  | 'new'
  | 'old'
  | 'email'
  | 'alpha'
  | 'created'
  | 'entries'
  | 'feature-requests'
  | 'bugs'
  | 'support'
  | 'others'

export type ApiSortDirection = 'asc' | 'desc'

export type ApiUserListParams = {
  search?: string
  role?: ApiUserRole
  status?: ApiUserStatus
  sort?: ApiUserSort
  direction?: ApiSortDirection
  page?: number
  limit?: number
}

export type ApiUserListResponse = { data: ApiUserRow[]; meta: ApiListMeta }

export type ApiFile = {
  id: string
  url: string
  originalName: string
  mimeType: string
  size: number
  isImage: boolean
}

// Dashboard stats — aggregated picture rendered on the Console
// Dashboard page. Numbers are recomputed on every request; there is
// no client-side cache invalidation tied to entry/topic/release
// mutations beyond React Query's normal staleness rules.
export type ApiDashboardEntryRow = {
  id: string
  title: string
  entryType?: string
  status?: string
  voteCount: number
  commentCount: number
  isInternal: boolean
  createdAt: string
}

export type ApiDashboardDayBucket = {
  date: string
  count: number
}

export type ApiDashboard = {
  totalEntries: number
  openEntries: number
  closedEntries: number
  internalEntries: number
  publicEntries: number
  totalVotes: number
  totalComments: number
  totalUsers: number
  totalTopics: number
  totalReleases: number
  entriesByType: Record<string, number>
  entriesByStatus: Record<string, number>
  entriesPerDay: ApiDashboardDayBucket[]
  topEntriesByVote: ApiDashboardEntryRow[]
  recentEntries: ApiDashboardEntryRow[]
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function uploadFile(path: string, file: File): Promise<ApiFile> {
  const form = new FormData()
  form.append('file', file)
  const res = await fetch(path, {
    method: 'POST',
    body: form,
    credentials: 'include',
  })
  if (!res.ok) {
    let message = res.statusText
    try {
      const data = await res.json()
      if (data?.message) message = data.message
    } catch {
      // non-JSON body — leave default
    }
    throw new ApiError(res.status, message)
  }
  return res.json() as Promise<ApiFile>
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
    credentials: 'include',
  })
  if (!res.ok) {
    let message = res.statusText
    try {
      const data = await res.json()
      if (data?.message) message = data.message
    } catch {
      // non-JSON body — leave default
    }
    throw new ApiError(res.status, message)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

function qs(params: Record<string, string | number | string[] | undefined>): string {
  const u = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null) continue
    if (Array.isArray(v)) {
      for (const item of v) {
        if (item !== undefined && item !== null && item !== '') u.append(k, String(item))
      }
      continue
    }
    if (v !== '') u.set(k, String(v))
  }
  const s = u.toString()
  return s ? '?' + s : ''
}

export const API = () => ({
  auth: {
    login: (body: { email: string; password: string }) =>
      request<{ user: ApiUser }>('POST', '/api/login', body),
    logout: () => request<void>('POST', '/api/logout'),
    me: () => request<{ user: ApiUser }>('GET', '/api/me'),
  },
  console: {
    getDashboard: () => request<ApiDashboard>('GET', '/api/console/dashboard'),
    listEntries: (
      params: {
        sort?: string
        search?: string
        page?: number
        limit?: number
        entryType?: ApiEntryTypeValue
        status?: ApiEntryStatusValue
        authorId?: string
        topicId?: string
      } = {},
    ) => request<ApiEntryListResponse>('GET', '/api/console/entry' + qs(params)),
    getEntry: (id: string) => request<ApiEntry>('GET', `/api/console/entry/${id}`),
    updateEntry: (
      id: string,
      body: {
        isInternal?: boolean
        entryType?: ApiEntryTypeValue
        status?: ApiEntryStatusValue
        topicIds?: string[]
        releaseId?: string
      },
    ) => request<ApiEntry>('PATCH', `/api/console/entry/${id}`, body),
    listEntryAuthors: () =>
      request<ApiEntryAuthorListResponse>('GET', '/api/console/entry-author'),
    listEntryTopics: () =>
      request<ApiEntryTopicListResponse>('GET', '/api/console/entry-topic'),
    createEntryTopic: (body: {
      title: string
      description?: string
      color?: string
      sortOrder?: number
    }) => request<ApiEntryTopic>('POST', '/api/console/entry-topic', body),
    updateEntryTopic: (
      id: string,
      body: { title?: string; description?: string; color?: string; sortOrder?: number },
    ) => request<ApiEntryTopic>('PATCH', `/api/console/entry-topic/${id}`, body),
    deleteEntryTopic: (id: string) =>
      request<void>('DELETE', `/api/console/entry-topic/${id}`),
    listReleases: () =>
      request<ApiReleaseListResponse>('GET', '/api/console/release'),
    getRelease: (id: string) =>
      request<ApiReleaseDetail>('GET', `/api/console/release/${id}`),
    createRelease: (body: {
      versionName: string
      title?: string
      description?: string
      releaseDate?: string
      state?: ApiReleaseState
      pdfFileUrl?: string
      pdfFileName?: string
      pdfFileSize?: number
    }) => request<ApiRelease>('POST', '/api/console/release', body),
    updateRelease: (
      id: string,
      body: {
        versionName?: string
        title?: string
        description?: string
        releaseDate?: string
        state?: ApiReleaseState
        // Set to clear or replace the attachment; omit to leave it
        // untouched. Pass empty url/name/size=0 to clear.
        pdfFile?: { url: string; name: string; size: number }
      },
    ) => request<ApiRelease>('PATCH', `/api/console/release/${id}`, body),
    deleteRelease: (id: string) =>
      request<void>('DELETE', `/api/console/release/${id}`),
    toggleVote: (id: string) =>
      request<ApiEntry>('POST', `/api/console/entry/${id}/vote`),
    addEntryRelation: (
      id: string,
      body: { peerEntryId: string; type: ApiEntryRelationType },
    ) => request<ApiEntry>('POST', `/api/console/entry/${id}/relation`, body),
    deleteEntryRelation: (id: string, peerEntryId: string) =>
      request<ApiEntry>(
        'DELETE',
        `/api/console/entry/${id}/relation/${encodeURIComponent(peerEntryId)}`,
      ),
    mergeEntry: (id: string, body: { targetEntryId: string }) =>
      request<ApiEntry>('POST', `/api/console/entry/${id}/merge`, body),
    listComments: (entryId: string) =>
      request<ApiCommentListResponse>('GET', `/api/console/entry/${entryId}/comment`),
    listActivities: (entryId: string) =>
      request<ApiActivityListResponse>('GET', `/api/console/entry/${entryId}/activity`),
    updateComment: (
      entryId: string,
      commentId: string,
      body: { isInternal?: boolean },
    ) =>
      request<ApiComment>(
        'PATCH',
        `/api/console/entry/${entryId}/comment/${commentId}`,
        body,
      ),
    getMyProfile: () => request<ApiMyProfile>('GET', '/api/console/me'),
    updateMyProfile: (body: ApiMyProfilePatch) =>
      request<ApiMyProfile>('PATCH', '/api/console/me', body),
    updateMyPassword: (body: ApiMyPasswordPatch) =>
      request<void>('POST', '/api/console/me/password', body),
    getSettings: () => request<ApiSettings>('GET', '/api/console/settings'),
    updateSettings: (body: ApiSettingsPatch) =>
      request<ApiSettings>('PATCH', '/api/console/settings', body),
    updateAISettings: (body: ApiAISettingsPatch) =>
      request<ApiSettings>('PATCH', '/api/console/settings/ai', body),
    getAIQueue: () => request<ApiAIQueueStats>('GET', '/api/console/ai/queue'),
    getIntegrations: () =>
      request<{ github: ApiGitHubIntegration }>('GET', '/api/console/integrations'),
    // Connect or rotate the GitHub integration. Backend verifies the
    // (owner, repo, token) triple against GitHub before persisting —
    // a 400 surfaces the upstream message verbatim ("Bad credentials",
    // "Not Found") so a typo or scope mistake is self-explanatory.
    updateGitHubIntegration: (body: {
      owner: string
      repo: string
      token: string
    }) =>
      request<{ github: ApiGitHubIntegration }>(
        'PUT',
        '/api/console/integrations/github',
        body,
      ),
    deleteGitHubIntegration: () =>
      request<{ github: ApiGitHubIntegration }>(
        'DELETE',
        '/api/console/integrations/github',
      ),
    // Publish a feature-request or bug as a GitHub issue. AI
    // summarises the entry server-side; the resulting issue link is
    // persisted on the entry and returned in the response so the
    // caller can flip the button to "View on GitHub →" without a
    // refetch.
    createGitHubIssue: (entryId: string) =>
      request<ApiEntry>('POST', `/api/console/entry/${entryId}/github-issue`),
    listAdministrators: () =>
      request<ApiAdministratorListResponse>('GET', '/api/console/administrator'),
    listUsers: (params: ApiUserListParams = {}) =>
      request<ApiUserListResponse>('GET', '/api/console/user' + qs(params)),
    updateUserRoles: (id: string, body: { roles: string[] }) =>
      request<ApiAdministrator>('PATCH', `/api/console/user/${id}/roles`, body),
    updateUserBlock: (id: string, body: { isBlocked: boolean }) =>
      request<ApiAdministrator>('PATCH', `/api/console/user/${id}/block`, body),
    uploadFile: (file: File) => uploadFile('/api/console/files', file),
  },
})
