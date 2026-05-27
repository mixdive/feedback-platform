// Hand-typed API client for the Portal.
//
// Authentication is via the HTTPOnly mixdive_session cookie set by the
// custom-auth callback (POST /api/portal/auth/custom). Every request runs
// with credentials: 'include' so the cookie travels both in dev (Vite proxy
// on a different origin) and in production.

export type ApiPortalConfig = {
  projectName: string
  logoUrl?: string
  primaryColor?: string
  customAuthEnabled: boolean
  authUrl?: string
  customAuthButtonText?: string
  // Admin-managed markdown templates used to pre-fill the description
  // field on the new-entry form. Keys are EntryType values
  // ("feature-request", "bug", "support", "other"); empty string = no
  // template, in which case the new-entry form falls back to the
  // per-type placeholder copy.
  entryTypeTemplates: Record<string, string>
  // "New Support Request" Portal button. When supportRequestEnabled
  // is true and supportRequestUrl is set, the entries page renders
  // the button as a link that opens the URL in a new tab. When
  // false, the button is hidden — there is no in-portal support
  // submission path.
  supportRequestEnabled: boolean
  supportRequestUrl?: string
}

export type ApiEntryCreator = {
  id: string
  name?: string
  username?: string
  imageUrl?: string
}

// EntryType is a hardcoded enum on the backend. The wire payload
// carries the value plus display metadata (title/color/icon) so the
// Portal chip on every entry card renders without a separate lookup.
// Frontends keep the same metadata in src/utils/entry-type.ts.
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

// EntryStatus is a hardcoded enum on the backend. The wire payload
// carries the value plus display metadata (title/color/icon) so the
// Portal chip on every entry card renders without a separate lookup.
// Frontends keep the same metadata in src/utils/entry-status.ts.
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

// Releases group entries shipped under a single version. The Portal
// only sees them via entry badges and on the dedicated changelog
// page; planned releases are hidden from this surface.
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

export type ApiChangelogEntry = {
  id: string
  title: string
  description?: string
}

export type ApiChangelogItem = ApiRelease & {
  entries: ApiChangelogEntry[]
}

export type ApiChangelogResponse = { data: ApiChangelogItem[] }

export type ApiEntry = {
  id: string
  title: string
  description?: string
  voteCount: number
  isVoted: boolean
  commentCount: number
  creator?: ApiEntryCreator
  entryType?: ApiEntryType
  status: ApiEntryStatus
  release?: ApiRelease
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

export type ApiListMeta = { total: number; page: number; limit: number; hasMore: boolean }
export type ApiEntryListResponse = { data: ApiEntry[]; meta: ApiListMeta }

// AI-suggested existing entry for the create-entry similarity panel.
// `aiEnabled` distinguishes "no similar entries" from "AI is off / no
// API key configured" so the UI can show the right empty state.
export type ApiFindSimilarMatch = { entry: ApiEntry; reason?: string }
export type ApiFindSimilarResponse = {
  data: ApiFindSimilarMatch[]
  aiEnabled: boolean
}

// Per-user vote quota state. Console-access users (admins + editors)
// have `unlimited: true` and ignore `used` / `max`. Non-admin portal
// users see counts in terms of votes already cast against open
// entries.
export type ApiVoteQuota = {
  used: number
  max: number
  unlimited: boolean
}

// Per-user open-feature-request quota state. Same shape and Console
// bypass rule as ApiVoteQuota; `used` counts the caller's currently
// open authored feature requests.
export type ApiFeatureRequestQuota = {
  used: number
  max: number
  unlimited: boolean
}

export type ApiPortalUser = {
  id: string
  email?: string
  name?: string
  username?: string
  imageUrl?: string
  roles: string[]
  voteQuota: ApiVoteQuota
  featureRequestQuota: ApiFeatureRequestQuota
}

export type ApiFile = {
  id: string
  url: string
  originalName: string
  mimeType: string
  size: number
  isImage: boolean
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

function qs(params: Record<string, string | number | undefined>): string {
  const u = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') u.set(k, String(v))
  }
  const s = u.toString()
  return s ? '?' + s : ''
}

export const API = () => ({
  auth: {
    me: () => request<{ user: ApiPortalUser }>('GET', '/api/me'),
    logout: () => request<void>('POST', '/api/logout'),
    customLogin: (token: string) =>
      request<{ user: ApiPortalUser }>('POST', '/api/portal/auth/custom', { token }),
  },
  portal: {
    config: () => request<ApiPortalConfig>('GET', '/api/portal/config'),
    listEntries: (
      params: {
        sort?: string
        search?: string
        page?: number
        limit?: number
        mine?: boolean
        entryType?: ApiEntryTypeValue
        openOnly?: boolean
      } = {},
    ) =>
      request<ApiEntryListResponse>(
        'GET',
        '/api/portal/entry' +
          qs({
            sort: params.sort,
            search: params.search,
            page: params.page,
            limit: params.limit,
            mine: params.mine ? 'true' : undefined,
            entryType: params.entryType,
            openOnly: params.openOnly ? 'true' : undefined,
          }),
      ),
    getEntry: (id: string) => request<ApiEntry>('GET', `/api/portal/entry/${id}`),
    addVote: (entryId: string) =>
      request<ApiEntry>('POST', `/api/portal/entry/${entryId}/vote`),
    submit: (body: {
      title: string
      description?: string
      entryType?: ApiEntryTypeValue
      isInternal?: boolean
    }) => request<ApiEntry>('POST', '/api/portal/entry', body),
    findSimilar: (body: { title: string; description?: string }) =>
      request<ApiFindSimilarResponse>('POST', '/api/portal/entry/find-similar', body),
    listChangelog: () =>
      request<ApiChangelogResponse>('GET', '/api/portal/changelog'),
    getChangelogRelease: (slug: string) =>
      request<ApiChangelogItem>(
        'GET',
        `/api/portal/changelog/${encodeURIComponent(slug)}`,
      ),
    listComments: (entryId: string) =>
      request<ApiCommentListResponse>('GET', `/api/portal/entry/${entryId}/comment`),
    createComment: (entryId: string, body: { body: string }) =>
      request<ApiComment>('POST', `/api/portal/entry/${entryId}/comment`, body),
    uploadFile: (file: File) => uploadFile('/api/portal/files', file),
  },
})
