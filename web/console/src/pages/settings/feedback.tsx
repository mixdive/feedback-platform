import { useEffect, useState } from 'react'
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil, X } from 'lucide-react'

import Button from '@/components/button'
import MarkdownEditor from '@/components/markdown-editor'
import ReadOnlyBanner from '@/components/read-only-banner'
import {
  API,
  type ApiEntryTypeValue,
  type ApiSettings,
  type ApiSupportRequestSettings,
  type ApiUploadSettings,
} from '@/services/api'
import { useAppSelector } from '@/store/hooks'
import { entryTypeInfo } from '@/utils/entry-type'
import { message } from '@/utils/helpers'

const MIN_VOTES = 1
const MIN_FEATURE_REQUESTS = 1

// Entry types that get a per-type description template on the Portal
// new-entry form. Support is excluded — Portal can't create support
// entries (the "New Support Request" button opens an external URL),
// so its template has no surface to land on. "other" is admin-only
// and never reached from a Portal "New X" button.
const TEMPLATE_TYPES: ApiEntryTypeValue[] = [
  'feature-request',
  'bug',
]

export default function FeedbackSettingsPage() {
  const queryClient = useQueryClient()
  const isAdmin = !!useAppSelector((s) => s.auth.user?.roles?.includes('admin'))

  const [maxVotes, setMaxVotes] = useState<string | null>(null)
  const [maxFeatureRequests, setMaxFeatureRequests] = useState<string | null>(null)
  const [editingType, setEditingType] = useState<ApiEntryTypeValue | null>(null)

  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'settings'],
    queryFn: () => API().console.getSettings(),
  })

  useEffect(() => {
    if (!data) return
    if (maxVotes === null) {
      setMaxVotes(String(data.feedback.maxVotesPerUser))
    }
    if (maxFeatureRequests === null) {
      setMaxFeatureRequests(String(data.feedback.maxFeatureRequestsPerUser))
    }
  }, [data, maxVotes, maxFeatureRequests])

  const updateMut = useMutation({
    mutationFn: (next: {
      maxVotesPerUser?: number
      maxFeatureRequestsPerUser?: number
    }) => API().console.updateSettings({ feedback: next }),
    onSuccess: () => {
      message('Feedback settings saved', 'success')
      void queryClient.invalidateQueries({ queryKey: ['console', 'settings'] })
    },
    onError: (e) => message(e),
  })

  if (isLoading) return <p className="text-zinc-500">Loading…</p>
  if (error) return <p className="text-rose-600">{(error as Error).message}</p>
  if (maxVotes === null || maxFeatureRequests === null || !data) return null

  const parsedVotes = Number(maxVotes)
  const votesValidationError = (() => {
    if (maxVotes.trim() === '') return 'Required.'
    if (!Number.isInteger(parsedVotes)) return 'Must be a whole number.'
    if (parsedVotes < MIN_VOTES) return `Must be at least ${MIN_VOTES}.`
    return null
  })()

  const parsedFR = Number(maxFeatureRequests)
  const frValidationError = (() => {
    if (maxFeatureRequests.trim() === '') return 'Required.'
    if (!Number.isInteger(parsedFR)) return 'Must be a whole number.'
    if (parsedFR < MIN_FEATURE_REQUESTS) return `Must be at least ${MIN_FEATURE_REQUESTS}.`
    return null
  })()

  const votesDirty = parsedVotes !== data.feedback.maxVotesPerUser
  const frDirty = parsedFR !== data.feedback.maxFeatureRequestsPerUser
  const anyQuotaDirty = votesDirty || frDirty
  const anyQuotaError = !!votesValidationError || !!frValidationError

  return (
    <div className="space-y-6">
      {!isAdmin && <ReadOnlyBanner />}
      <form
        className="space-y-6 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6"
        onSubmit={(e) => {
          e.preventDefault()
          if (!isAdmin || anyQuotaError || !anyQuotaDirty) return
          const patch: { maxVotesPerUser?: number; maxFeatureRequestsPerUser?: number } = {}
          if (votesDirty) patch.maxVotesPerUser = parsedVotes
          if (frDirty) patch.maxFeatureRequestsPerUser = parsedFR
          updateMut.mutate(patch)
        }}
      >
        <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
          Feedback Settings
        </h2>
        <fieldset disabled={!isAdmin} className="space-y-5 disabled:opacity-70">
          <div>
            <label className="block text-sm font-medium mb-1">
              Votes per user
            </label>
            <input
              type="number"
              min={MIN_VOTES}
              step={1}
              value={maxVotes}
              onChange={(e) => setMaxVotes(e.target.value)}
              className="block w-32 h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed"
            />
            {votesValidationError ? (
              <p className="text-xs text-rose-600 mt-1">{votesValidationError}</p>
            ) : (
              <p className="text-xs text-zinc-500 mt-1">
                How many open entries a single user can have active votes on
                at once. A vote is returned to the user when its entry is
                completed or cancelled. Minimum 1. Admins and editors (anyone
                with Console access) are not subject to this limit.
              </p>
            )}
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">
              Open feature requests per user
            </label>
            <input
              type="number"
              min={MIN_FEATURE_REQUESTS}
              step={1}
              value={maxFeatureRequests}
              onChange={(e) => setMaxFeatureRequests(e.target.value)}
              className="block w-32 h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed"
            />
            {frValidationError ? (
              <p className="text-xs text-rose-600 mt-1">{frValidationError}</p>
            ) : (
              <p className="text-xs text-zinc-500 mt-1">
                How many feature requests a single user can have open at
                once. The slot is returned to the user when one of their
                feature requests is completed or cancelled. Minimum 1.
                Admins and editors (anyone with Console access) are not
                subject to this limit.
              </p>
            )}
          </div>
        </fieldset>
        {isAdmin && (
          <div className="flex justify-end">
            <Button
              type="submit"
              isLoading={updateMut.isPending}
              disabled={anyQuotaError || !anyQuotaDirty}
            >
              Save
            </Button>
          </div>
        )}
      </form>

      <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6 space-y-4">
        <div>
          <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
            Feedback Templates
          </h2>
          <p className="text-xs text-zinc-500 mt-1">
            Markdown shown as the starting point in the description field
            when a user opens "New feedback" on the Portal. Leave blank to
            skip the template — the per-type placeholder text shows instead.
          </p>
        </div>
        <ul className="divide-y divide-zinc-100 dark:divide-zinc-800 border border-zinc-200 dark:border-zinc-800 rounded-md">
          {TEMPLATE_TYPES.map((entryType) => {
            const info = entryTypeInfo(entryType)
            const stored = data.feedback.entryTypeTemplates?.[entryType] ?? ''
            const hasTemplate = stored.trim() !== ''
            return (
              <li
                key={entryType}
                className="flex items-center justify-between gap-3 px-4 py-3"
              >
                <div className="min-w-0">
                  <div className="text-sm font-medium">
                    {info?.title ?? entryType}
                  </div>
                  <div className="text-xs text-zinc-500">
                    {hasTemplate
                      ? `${stored.length} ${stored.length === 1 ? 'char' : 'chars'} configured`
                      : 'No template — placeholder text will show'}
                  </div>
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => setEditingType(entryType)}
                  disabled={!isAdmin}
                >
                  <Pencil className="size-4" />
                  Edit Template
                </Button>
              </li>
            )
          })}
        </ul>
      </div>

      <SupportRequestCard
        isAdmin={isAdmin}
        current={data.feedback.supportRequest}
      />

      <UploadsCard isAdmin={isAdmin} current={data.uploads} />

      <EditTemplateDialog
        entryType={editingType}
        settings={data}
        onClose={() => setEditingType(null)}
      />
    </div>
  )
}

// Validates that input is an http(s) absolute URL with a host —
// mirrors the server-side isValidAbsoluteURL gate.
function isValidAbsoluteUrl(s: string): boolean {
  try {
    const u = new URL(s)
    return (u.protocol === 'http:' || u.protocol === 'https:') && !!u.host
  } catch {
    return false
  }
}

interface SupportRequestCardProps {
  isAdmin: boolean
  current: ApiSupportRequestSettings
}

function SupportRequestCard({ isAdmin, current }: SupportRequestCardProps) {
  const queryClient = useQueryClient()
  const [enabled, setEnabled] = useState(current.enabled)
  const [url, setUrl] = useState(current.url)

  // Re-seed local edits whenever the server state changes (e.g. after
  // a successful save invalidates the query and the parent re-passes
  // the freshly-fetched settings).
  useEffect(() => {
    setEnabled(current.enabled)
    setUrl(current.url)
  }, [current.enabled, current.url])

  const updateMut = useMutation({
    mutationFn: (next: Partial<ApiSupportRequestSettings>) =>
      API().console.updateSettings({ feedback: { supportRequest: next } }),
    onSuccess: () => {
      message('Support request settings saved', 'success')
      void queryClient.invalidateQueries({ queryKey: ['console', 'settings'] })
    },
    onError: (e) => message(e),
  })

  const trimmedUrl = url.trim()
  const urlValidationError = (() => {
    if (!enabled) return null
    if (trimmedUrl === '') return 'URL is required when the button is enabled.'
    if (!isValidAbsoluteUrl(trimmedUrl))
      return 'Must be a valid absolute URL (http:// or https://).'
    return null
  })()

  const dirty = enabled !== current.enabled || trimmedUrl !== current.url

  return (
    <form
      className="space-y-5 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6"
      onSubmit={(e) => {
        e.preventDefault()
        if (!isAdmin || urlValidationError || !dirty) return
        updateMut.mutate({ enabled, url: trimmedUrl })
      }}
    >
      <div>
        <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
          New Support Request Button
        </h2>
        <p className="text-xs text-zinc-500 mt-1">
          When enabled, the Portal shows a "New Support Request" button
          that opens the URL below in a new tab. When disabled, the
          button is hidden from the Portal entirely.
        </p>
      </div>
      <fieldset disabled={!isAdmin} className="space-y-4 disabled:opacity-70">
        <label className="flex items-start gap-3 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/40 p-3 cursor-pointer">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
            className="mt-0.5 size-4 rounded border-zinc-300 dark:border-zinc-700 text-sky-600 focus:ring-sky-500"
          />
          <span className="text-sm">
            <span className="block font-medium text-zinc-900 dark:text-zinc-100">
              Show "New Support Request" button on the Portal
            </span>
            <span className="block text-xs text-zinc-500">
              Clicking the button opens the URL below in a new tab —
              the in-portal support form is not used.
            </span>
          </span>
        </label>
        <div>
          <label className="block text-sm font-medium mb-1">URL</label>
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://support.example.com/new"
            className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed"
          />
          {urlValidationError ? (
            <p className="text-xs text-rose-600 mt-1">{urlValidationError}</p>
          ) : (
            <p className="text-xs text-zinc-500 mt-1">
              Absolute URL the button opens (e.g. your Zendesk or
              Intercom form). Required when the button is enabled.
            </p>
          )}
        </div>
      </fieldset>
      {isAdmin && (
        <div className="flex justify-end">
          <Button
            type="submit"
            isLoading={updateMut.isPending}
            disabled={!!urlValidationError || !dirty}
          >
            Save
          </Button>
        </div>
      )}
    </form>
  )
}

interface UploadsCardProps {
  isAdmin: boolean
  current: ApiUploadSettings
}

function UploadsCard({ isAdmin, current }: UploadsCardProps) {
  const queryClient = useQueryClient()
  const [enabled, setEnabled] = useState(current.enabled)
  const [backend, setBackend] = useState<'local' | 'gcs'>(
    current.backend === 'gcs' ? 'gcs' : 'local',
  )
  const [gcsBucket, setGcsBucket] = useState(current.gcsBucket ?? '')

  // Re-seed local edits whenever the server state changes (e.g. after
  // a successful save invalidates the query and the parent re-passes
  // the freshly-fetched settings).
  useEffect(() => {
    setEnabled(current.enabled)
    setBackend(current.backend === 'gcs' ? 'gcs' : 'local')
    setGcsBucket(current.gcsBucket ?? '')
  }, [current.enabled, current.backend, current.gcsBucket])

  const updateMut = useMutation({
    mutationFn: (next: Partial<Omit<ApiUploadSettings, 'lastError'>>) =>
      API().console.updateSettings({ uploads: next }),
    onSuccess: () => {
      message('Upload settings saved', 'success')
      void queryClient.invalidateQueries({ queryKey: ['console', 'settings'] })
    },
    onError: (e) => message(e),
  })

  const trimmedBucket = gcsBucket.trim()
  const bucketValidationError = (() => {
    if (!enabled) return null
    if (backend !== 'gcs') return null
    if (trimmedBucket === '') return 'GCS bucket is required when backend is GCS.'
    return null
  })()

  const dirty =
    enabled !== current.enabled ||
    (enabled && backend !== (current.backend === 'gcs' ? 'gcs' : 'local')) ||
    (enabled && backend === 'gcs' && trimmedBucket !== (current.gcsBucket ?? ''))

  return (
    <form
      className="space-y-5 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6"
      onSubmit={(e) => {
        e.preventDefault()
        if (!isAdmin || bucketValidationError || !dirty) return
        const patch: Partial<Omit<ApiUploadSettings, 'lastError'>> = {
          enabled,
        }
        if (enabled) {
          patch.backend = backend
          patch.gcsBucket = backend === 'gcs' ? trimmedBucket : ''
        }
        updateMut.mutate(patch)
      }}
    >
      <div>
        <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
          File Uploads
        </h2>
        <p className="text-xs text-zinc-500 mt-1">
          Lets end users and admins attach images, videos, and documents
          to feedback. When disabled, the Portal attach button is hidden
          and every new upload (Portal, Console, profile picture, logo)
          is rejected with a 403. Previously-uploaded files remain
          accessible via their existing links.
        </p>
      </div>
      {current.lastError && (
        <div className="rounded-md border border-rose-200 bg-rose-50 dark:border-rose-900 dark:bg-rose-950/40 px-3 py-2 text-xs text-rose-700 dark:text-rose-300">
          <span className="font-semibold">Last backend error:</span>{' '}
          {current.lastError}
        </div>
      )}
      <fieldset disabled={!isAdmin} className="space-y-4 disabled:opacity-70">
        <label className="flex items-start gap-3 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/40 p-3 cursor-pointer">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
            className="mt-0.5 size-4 rounded border-zinc-300 dark:border-zinc-700 text-sky-600 focus:ring-sky-500"
          />
          <span className="text-sm">
            <span className="block font-medium text-zinc-900 dark:text-zinc-100">
              Allow file uploads
            </span>
            <span className="block text-xs text-zinc-500">
              When off, the Portal and Console reject new uploads with a
              403. Reads of existing files still work.
            </span>
          </span>
        </label>
        {enabled && (
          <div className="space-y-3 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/40 p-3">
            <span className="block text-sm font-medium">Storage backend</span>
            <label className="flex items-start gap-3 cursor-pointer">
              <input
                type="radio"
                name="upload-backend"
                value="local"
                checked={backend === 'local'}
                onChange={() => setBackend('local')}
                className="mt-0.5 size-4 border-zinc-300 dark:border-zinc-700 text-sky-600 focus:ring-sky-500"
              />
              <span className="text-sm">
                <span className="block font-medium">Local volume</span>
                <span className="block text-xs text-zinc-500">
                  Files write to <code>./data/files</code> on the server.
                  Mount a persistent volume there in production so files
                  survive container restarts.
                </span>
              </span>
            </label>
            <label className="flex items-start gap-3 cursor-pointer">
              <input
                type="radio"
                name="upload-backend"
                value="gcs"
                checked={backend === 'gcs'}
                onChange={() => setBackend('gcs')}
                className="mt-0.5 size-4 border-zinc-300 dark:border-zinc-700 text-sky-600 focus:ring-sky-500"
              />
              <span className="text-sm">
                <span className="block font-medium">Google Cloud Storage</span>
                <span className="block text-xs text-zinc-500">
                  Files write to a GCS bucket. Authentication uses
                  Application Default Credentials — the runtime service
                  account on Cloud Run / GCE / GKE.
                </span>
              </span>
            </label>
            {backend === 'gcs' && (
              <div>
                <label className="block text-sm font-medium mb-1">
                  GCS bucket
                </label>
                <input
                  type="text"
                  value={gcsBucket}
                  onChange={(e) => setGcsBucket(e.target.value)}
                  placeholder="my-mixdive-uploads"
                  className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed"
                />
                {bucketValidationError ? (
                  <p className="text-xs text-rose-600 mt-1">
                    {bucketValidationError}
                  </p>
                ) : (
                  <p className="text-xs text-zinc-500 mt-1">
                    Bucket name only. The runtime service account needs
                    object read/write and (for previews) the
                    iam.serviceAccountTokenCreator role on itself.
                  </p>
                )}
              </div>
            )}
          </div>
        )}
      </fieldset>
      {isAdmin && (
        <div className="flex justify-end">
          <Button
            type="submit"
            isLoading={updateMut.isPending}
            disabled={!!bucketValidationError || !dirty}
          >
            Save
          </Button>
        </div>
      )}
    </form>
  )
}

interface EditTemplateDialogProps {
  entryType: ApiEntryTypeValue | null
  settings: ApiSettings
  onClose: () => void
}

function EditTemplateDialog({ entryType, settings, onClose }: EditTemplateDialogProps) {
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState('')

  const stored = entryType
    ? settings.feedback.entryTypeTemplates?.[entryType] ?? ''
    : ''
  const defaultTemplate = entryType
    ? settings.feedback.defaultEntryTypeTemplates?.[entryType] ?? ''
    : ''
  const info = entryType ? entryTypeInfo(entryType) : undefined

  // Re-seed the draft whenever the dialog is opened for a new entry
  // type. Closing the dialog with the X / Cancel discards the draft.
  useEffect(() => {
    if (entryType) setDraft(stored)
  }, [entryType, stored])

  const saveMut = useMutation({
    mutationFn: (next: Record<string, string>) =>
      API().console.updateSettings({ feedback: { entryTypeTemplates: next } }),
    onSuccess: () => {
      message(`${info?.title ?? entryType} template saved`, 'success')
      void queryClient.invalidateQueries({ queryKey: ['console', 'settings'] })
      onClose()
    },
    onError: (e) => message(e),
  })

  const dirty = draft !== stored
  const canReset = defaultTemplate !== '' && draft !== defaultTemplate
  const open = entryType !== null

  return (
    <Dialog
      open={open}
      onClose={() => (saveMut.isPending ? undefined : onClose())}
      className="relative z-40"
    >
      <div className="fixed inset-0 bg-black/40" aria-hidden />
      <div className="fixed inset-0 grid place-items-center p-4">
        <DialogPanel className="w-full max-w-3xl rounded-xl bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-6 space-y-4">
          <div className="flex items-start justify-between gap-3">
            <div>
              <DialogTitle className="text-lg font-semibold">
                Edit {info?.title ?? entryType} template
              </DialogTitle>
              <p className="text-xs text-zinc-500 mt-1">
                Pre-fills the description field on the Portal new-feedback
                form for this entry type.
              </p>
            </div>
            <button
              type="button"
              onClick={onClose}
              disabled={saveMut.isPending}
              className="inline-flex size-7 items-center justify-center rounded-md text-zinc-500 hover:bg-zinc-100 disabled:opacity-50 dark:hover:bg-zinc-800"
            >
              <X className="size-4" />
            </button>
          </div>

          <MarkdownEditor
            value={draft}
            onChange={setDraft}
            placeholder="Markdown supported. Use headings, bullets, and prompts to guide what users should fill in."
            rows={14}
          />

          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              {canReset && (
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => setDraft(defaultTemplate)}
                  disabled={saveMut.isPending}
                >
                  Reset to default
                </Button>
              )}
              {draft !== '' && (
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => setDraft('')}
                  disabled={saveMut.isPending}
                >
                  Clear
                </Button>
              )}
            </div>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="ghost"
                onClick={onClose}
                disabled={saveMut.isPending}
              >
                Cancel
              </Button>
              <Button
                type="button"
                isLoading={saveMut.isPending}
                disabled={!dirty || !entryType}
                onClick={() => {
                  if (!entryType) return
                  // Send the full map so any concurrent admin edits to
                  // other types are not clobbered.
                  const next: Record<string, string> = {
                    ...(settings.feedback.entryTypeTemplates ?? {}),
                  }
                  next[entryType] = draft
                  saveMut.mutate(next)
                }}
              >
                Save
              </Button>
            </div>
          </div>
        </DialogPanel>
      </div>
    </Dialog>
  )
}
