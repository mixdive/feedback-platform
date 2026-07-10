import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft,
  CalendarDays,
  FileText,
  Lock,
  Pencil,
  Plus,
  Trash2,
  X,
} from 'lucide-react'

import Button from '@/components/button'
import Markdown from '@/components/markdown'
import ReleaseFormDialog, {
  formatDate,
  ReleaseStateChip,
  type FormState,
} from '@/components/release-form-dialog'
import { API, type ApiReleaseDetail } from '@/services/api'
import { useAppSelector } from '@/store/hooks'
import { message } from '@/utils/helpers'

export default function ReleaseDetailPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const isAdmin = useAppSelector((s) =>
    (s.auth.user?.roles ?? []).includes('admin'),
  )

  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'release', id],
    queryFn: () => API().console.getRelease(id),
    enabled: !!id,
  })

  const [editing, setEditing] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['console', 'release', id] })
    void queryClient.invalidateQueries({ queryKey: ['console', 'releases'] })
    void queryClient.invalidateQueries({ queryKey: ['console', 'entries'] })
  }

  const updateMut = useMutation({
    mutationFn: (body: FormState) =>
      API().console.updateRelease(id, {
        versionName: body.versionName.trim(),
        title: body.title.trim(),
        description: body.description,
        releaseDate: body.releaseDate,
        state: body.state,
        pdfFile: body.pdf
          ? { url: body.pdf.url, name: body.pdf.name, size: body.pdf.size }
          : { url: '', name: '', size: 0 },
      }),
    onSuccess: () => {
      setEditing(false)
      message('Release updated', 'success')
      invalidate()
    },
    onError: (e) => message(e),
  })

  const deleteMut = useMutation({
    mutationFn: () => API().console.deleteRelease(id),
    onSuccess: () => {
      message('Release deleted', 'success')
      void queryClient.invalidateQueries({ queryKey: ['console', 'releases'] })
      void queryClient.invalidateQueries({ queryKey: ['console', 'entries'] })
      navigate('/releases')
    },
    onError: (e) => message(e),
  })

  // assignMut drives both add (releaseId = this release) and remove
  // (releaseId = '') — both are a single-field patch on the entry.
  const assignMut = useMutation({
    mutationFn: ({ entryId, releaseId }: { entryId: string; releaseId: string }) =>
      API().console.updateEntry(entryId, { releaseId }),
    onSuccess: () => invalidate(),
    onError: (e) => message(e),
  })
  const pendingEntryId = assignMut.isPending
    ? assignMut.variables?.entryId
    : undefined

  if (isLoading) {
    return <p className="p-4 text-sm text-zinc-500">Loading…</p>
  }
  if (error) {
    return (
      <p className="p-4 text-sm text-rose-600">{(error as Error).message}</p>
    )
  }
  if (!data) {
    return (
      <div className="space-y-4">
        <BackLink />
        <p className="p-4 text-sm text-zinc-500">Release not found.</p>
      </div>
    )
  }

  const release: ApiReleaseDetail = data

  return (
    <div className="space-y-4">
      <BackLink />

      {/* Header card */}
      <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-5 space-y-3">
        <div className="flex items-start gap-3">
          <ReleaseStateChip state={release.state} />
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <span className="rounded-md bg-zinc-100 dark:bg-zinc-800 px-2 py-0.5 font-mono text-xs text-zinc-700 dark:text-zinc-200">
                {release.versionName}
              </span>
              {release.title && (
                <h1 className="text-xl font-semibold">{release.title}</h1>
              )}
            </div>
            <div className="mt-1.5 flex items-center gap-3 text-xs text-zinc-500">
              <span className="capitalize">{release.state}</span>
              {release.releaseDate ? (
                <span className="inline-flex items-center gap-1">
                  <CalendarDays className="size-3.5" />
                  {formatDate(release.releaseDate)}
                </span>
              ) : (
                <span className="italic">No date</span>
              )}
            </div>
          </div>
          {isAdmin && (
            <div className="flex shrink-0 gap-1">
              <button
                type="button"
                title="Edit"
                onClick={() => setEditing(true)}
                className="rounded p-1.5 text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800 hover:text-zinc-800 dark:hover:text-zinc-200"
              >
                <Pencil className="size-4" />
              </button>
              <button
                type="button"
                title="Delete"
                onClick={() => setConfirmDelete(true)}
                className="rounded p-1.5 text-zinc-500 hover:bg-rose-50 dark:hover:bg-rose-500/10 hover:text-rose-600"
              >
                <Trash2 className="size-4" />
              </button>
            </div>
          )}
        </div>

        {release.description && <Markdown>{release.description}</Markdown>}

        {release.pdfFileUrl && (
          <a
            href={release.pdfFileUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 rounded-md border border-zinc-300 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950/40 px-3 py-2 text-sm text-sky-700 hover:bg-zinc-100 dark:text-sky-300 dark:hover:bg-zinc-800"
          >
            <FileText className="size-4 shrink-0 text-zinc-500" />
            {release.pdfFileName || 'release.pdf'}
          </a>
        )}
      </div>

      {/* Entries in this release */}
      <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 overflow-hidden">
        <div className="border-b border-zinc-200 dark:border-zinc-800 px-4 py-3">
          <h2 className="text-sm font-semibold">
            Entries in this release{' '}
            <span className="text-zinc-400">({release.entries.length})</span>
          </h2>
        </div>
        {release.entries.length === 0 ? (
          <p className="px-4 py-3 text-sm text-zinc-500">
            No entries yet. Search below to add some.
          </p>
        ) : (
          <ul className="divide-y divide-zinc-100 dark:divide-zinc-800">
            {release.entries.map((e) => (
              <li key={e.id} className="flex items-center gap-3 px-4 py-2.5">
                <Link
                  to={`/entry/${e.id}`}
                  className="min-w-0 flex-1 truncate text-sm text-zinc-700 hover:text-sky-700 hover:underline dark:text-zinc-200 dark:hover:text-sky-300"
                >
                  {e.title || '(untitled)'}
                </Link>
                {e.isInternal && (
                  <span className="inline-flex shrink-0 items-center gap-1 rounded bg-amber-50 px-1.5 py-0.5 text-xs text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">
                    <Lock className="size-3" />
                    Internal
                  </span>
                )}
                <button
                  type="button"
                  title="Remove from release"
                  disabled={assignMut.isPending}
                  onClick={() =>
                    assignMut.mutate({ entryId: e.id, releaseId: '' })
                  }
                  className="shrink-0 rounded p-1.5 text-zinc-500 hover:bg-rose-50 hover:text-rose-600 disabled:opacity-50 dark:hover:bg-rose-500/10"
                >
                  {pendingEntryId === e.id ? (
                    <span className="inline-block size-4 animate-spin rounded-full border-2 border-current border-r-transparent" />
                  ) : (
                    <X className="size-4" />
                  )}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Add entries */}
      <AddEntriesPanel
        releaseId={release.id}
        pendingEntryId={pendingEntryId}
        disabled={assignMut.isPending}
        onAdd={(entryId) => assignMut.mutate({ entryId, releaseId: release.id })}
      />

      {editing && (
        <ReleaseFormDialog
          title="Edit release"
          submitLabel="Save"
          isPending={updateMut.isPending}
          initial={{
            versionName: release.versionName,
            title: release.title ?? '',
            description: release.description ?? '',
            releaseDate: release.releaseDate ?? '',
            state: release.state,
            pdf: release.pdfFileUrl
              ? {
                  url: release.pdfFileUrl,
                  name: release.pdfFileName ?? 'release.pdf',
                  size: release.pdfFileSize ?? 0,
                }
              : null,
          }}
          onCancel={() => setEditing(false)}
          onSubmit={(form) => updateMut.mutate(form)}
        />
      )}

      {confirmDelete && (
        <Dialog
          open
          onClose={() => (deleteMut.isPending ? undefined : setConfirmDelete(false))}
          className="relative z-40"
        >
          <div className="fixed inset-0 bg-black/40" aria-hidden />
          <div className="fixed inset-0 grid place-items-center p-4">
            <DialogPanel className="w-full max-w-md rounded-xl bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-6 space-y-4">
              <DialogTitle className="text-lg font-semibold">
                Delete release?
              </DialogTitle>
              <p className="text-sm text-zinc-600 dark:text-zinc-300">
                This removes{' '}
                <span className="font-mono">{release.versionName}</span> and
                unlinks it from{' '}
                <span className="font-medium">
                  {release.entries.length}{' '}
                  {release.entries.length === 1 ? 'entry' : 'entries'}
                </span>
                . The entries are kept; only the release reference is cleared.
                This action cannot be undone.
              </p>
              <div className="flex justify-end gap-2">
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => setConfirmDelete(false)}
                  disabled={deleteMut.isPending}
                >
                  Cancel
                </Button>
                <Button
                  type="button"
                  variant="danger"
                  onClick={() => deleteMut.mutate()}
                  isLoading={deleteMut.isPending}
                >
                  Delete
                </Button>
              </div>
            </DialogPanel>
          </div>
        </Dialog>
      )}
    </div>
  )
}

function BackLink() {
  return (
    <Link
      to="/releases"
      className="inline-flex items-center gap-1.5 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
    >
      <ArrowLeft className="size-4" />
      Releases
    </Link>
  )
}

// AddEntriesPanel is the debounced entry-search picker. Mirrors the
// merge-dialog search pattern: a 200ms-debounced query against the
// entry list endpoint, results excluding entries already in THIS
// release. Empty query surfaces the most recent entries so admins have
// a starting point without typing.
function AddEntriesPanel({
  releaseId,
  pendingEntryId,
  disabled,
  onAdd,
}: {
  releaseId: string
  pendingEntryId: string | undefined
  disabled: boolean
  onAdd: (entryId: string) => void
}) {
  const [search, setSearch] = useState('')
  const [debounced, setDebounced] = useState('')

  useEffect(() => {
    const t = window.setTimeout(() => setDebounced(search.trim()), 200)
    return () => window.clearTimeout(t)
  }, [search])

  const { data, isFetching } = useQuery({
    queryKey: ['console', 'entries', 'release-picker', releaseId, debounced],
    queryFn: () => API().console.listEntries({ search: debounced, limit: 10 }),
  })

  const candidates = useMemo(
    () => (data?.data ?? []).filter((e) => e.release?.id !== releaseId),
    [data, releaseId],
  )

  return (
    <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 overflow-hidden">
      <div className="border-b border-zinc-200 dark:border-zinc-800 px-4 py-3">
        <h2 className="text-sm font-semibold">Add entries</h2>
        <p className="mt-0.5 text-xs text-zinc-500">
          Search feedback entries by title and attach them to this release.
        </p>
      </div>
      <div className="p-4 space-y-3">
        <input
          value={search}
          onChange={(ev) => setSearch(ev.target.value)}
          placeholder="Search entries by title…"
          className="block w-full h-9 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
        />
        <div className="rounded-md border border-zinc-200 dark:border-zinc-800 overflow-hidden">
          {isFetching && candidates.length === 0 && (
            <p className="px-3 py-2 text-xs text-zinc-500">Searching…</p>
          )}
          {!isFetching && candidates.length === 0 && (
            <p className="px-3 py-2 text-xs text-zinc-500">
              {debounced ? 'No matches.' : 'No entries to add.'}
            </p>
          )}
          {candidates.length > 0 && (
            <ul className="divide-y divide-zinc-100 dark:divide-zinc-800">
              {candidates.map((e) => (
                <li key={e.id} className="flex items-center gap-3 px-3 py-2">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate text-sm text-zinc-700 dark:text-zinc-200">
                        {e.title || '(untitled)'}
                      </span>
                      {e.isInternal && (
                        <span className="inline-flex shrink-0 items-center gap-1 rounded bg-amber-50 px-1.5 py-0.5 text-xs text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">
                          <Lock className="size-3" />
                          Internal
                        </span>
                      )}
                    </div>
                    {e.release && (
                      <span className="mt-0.5 inline-block text-xs text-zinc-400">
                        Currently in{' '}
                        <span className="font-mono">{e.release.versionName}</span>
                      </span>
                    )}
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    variant={e.release ? 'ghost' : 'primary'}
                    disabled={disabled}
                    isLoading={pendingEntryId === e.id}
                    onClick={() => onAdd(e.id)}
                  >
                    {!e.release && <Plus className="size-4" />}
                    {e.release ? 'Move here' : 'Add'}
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
