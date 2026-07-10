import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  CalendarDays,
  CheckCircle2,
  Circle,
  FileText,
  Paperclip,
  Pencil,
  Plus,
  Trash2,
  X,
} from 'lucide-react'
import clsx from 'clsx'

import Button from '@/components/button'
import EntryTypeCountChips from '@/components/entry-type-counts'
import MarkdownEditor from '@/components/markdown-editor'
import {
  API,
  type ApiReleaseListItem,
  type ApiReleaseState,
} from '@/services/api'
import { useAppSelector } from '@/store/hooks'
import { message } from '@/utils/helpers'

type FormPdf = { url: string; name: string; size: number }

type FormState = {
  versionName: string
  title: string
  description: string
  releaseDate: string
  state: ApiReleaseState
  pdf: FormPdf | null
}

const emptyForm: FormState = {
  versionName: '',
  title: '',
  description: '',
  releaseDate: '',
  state: 'planned',
  pdf: null,
}

export default function ReleasesPage() {
  const queryClient = useQueryClient()
  const isAdmin = useAppSelector((s) =>
    (s.auth.user?.roles ?? []).includes('admin'),
  )

  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'releases'],
    queryFn: () => API().console.listReleases(),
  })

  const [editing, setEditing] = useState<ApiReleaseListItem | null>(null)
  const [creating, setCreating] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState<ApiReleaseListItem | null>(null)

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['console', 'releases'] })
    void queryClient.invalidateQueries({ queryKey: ['console', 'entries'] })
  }

  const createMut = useMutation({
    mutationFn: (body: FormState) =>
      API().console.createRelease({
        versionName: body.versionName.trim(),
        title: body.title.trim() || undefined,
        description: body.description || undefined,
        releaseDate: body.releaseDate || undefined,
        state: body.state,
        pdfFileUrl: body.pdf?.url || undefined,
        pdfFileName: body.pdf?.name || undefined,
        pdfFileSize: body.pdf?.size || undefined,
      }),
    onSuccess: () => {
      setCreating(false)
      message('Release created', 'success')
      invalidate()
    },
    onError: (e) => message(e),
  })

  const updateMut = useMutation({
    mutationFn: ({ id, body }: { id: string; body: FormState }) =>
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
      setEditing(null)
      message('Release updated', 'success')
      invalidate()
    },
    onError: (e) => message(e),
  })

  const deleteMut = useMutation({
    mutationFn: (id: string) => API().console.deleteRelease(id),
    onSuccess: () => {
      setConfirmDelete(null)
      message('Release deleted', 'success')
      invalidate()
    },
    onError: (e) => message(e),
  })

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Releases</h1>
          <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
            Plan and manage releases. Admins create version records here;
            admins and editors attach individual entries to a release from
            the entry edit form. Completed releases automatically appear on
            the Portal Changelog.
          </p>
        </div>
        {isAdmin && (
          <Button size="sm" onClick={() => setCreating(true)}>
            <Plus className="size-4" />
            New release
          </Button>
        )}
      </div>

      <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 overflow-hidden">
        {isLoading && <p className="p-4 text-sm text-zinc-500">Loading…</p>}
        {error && (
          <p className="p-4 text-sm text-rose-600">{(error as Error).message}</p>
        )}
        {data && data.data.length === 0 && (
          <p className="p-4 text-sm text-zinc-500">
            No releases yet. Click <span className="font-medium">New release</span> to
            plan your first version.
          </p>
        )}
        {data && data.data.length > 0 && (
          <ul className="divide-y divide-zinc-200 dark:divide-zinc-800">
            {data.data.map((r) => (
              <li
                key={r.id}
                className="flex items-center gap-3 px-4 py-3 hover:bg-zinc-50 dark:hover:bg-zinc-950/40"
              >
                <ReleaseStateChip state={r.state} />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="rounded-md bg-zinc-100 dark:bg-zinc-800 px-2 py-0.5 font-mono text-xs text-zinc-700 dark:text-zinc-200">
                      {r.versionName}
                    </span>
                    {r.title && (
                      <span className="truncate text-sm text-zinc-700 dark:text-zinc-200">
                        {r.title}
                      </span>
                    )}
                  </div>
                  <div className="mt-1 flex items-center gap-3 text-xs text-zinc-500">
                    {r.releaseDate ? (
                      <span className="inline-flex items-center gap-1">
                        <CalendarDays className="size-3.5" />
                        {formatDate(r.releaseDate)}
                      </span>
                    ) : (
                      <span className="italic">No date</span>
                    )}
                  </div>
                </div>
                <EntryTypeCountChips counts={r.entryTypeCounts} />
                {r.entryCount > 0 ? (
                  <Link
                    to={`/entry?releaseId=${r.id}`}
                    title={`View entries in ${r.versionName}`}
                    className="text-xs tabular-nums text-sky-700 dark:text-sky-300 hover:underline shrink-0 min-w-[4rem] text-right"
                  >
                    {r.entryCount} {r.entryCount === 1 ? 'entry' : 'entries'}
                  </Link>
                ) : (
                  <span className="text-xs tabular-nums text-zinc-500 shrink-0 min-w-[4rem] text-right">
                    0 entries
                  </span>
                )}
                {isAdmin && (
                  <div className="flex shrink-0 gap-1">
                    <button
                      type="button"
                      title="Edit"
                      onClick={() => setEditing(r)}
                      className="rounded p-1.5 text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800 hover:text-zinc-800 dark:hover:text-zinc-200"
                    >
                      <Pencil className="size-4" />
                    </button>
                    <button
                      type="button"
                      title="Delete"
                      onClick={() => setConfirmDelete(r)}
                      className="rounded p-1.5 text-zinc-500 hover:bg-rose-50 dark:hover:bg-rose-500/10 hover:text-rose-600"
                    >
                      <Trash2 className="size-4" />
                    </button>
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>

      {creating && (
        <ReleaseFormDialog
          title="New release"
          submitLabel="Create"
          isPending={createMut.isPending}
          initial={emptyForm}
          onCancel={() => setCreating(false)}
          onSubmit={(form) => createMut.mutate(form)}
        />
      )}

      {editing && (
        <ReleaseFormDialog
          title="Edit release"
          submitLabel="Save"
          isPending={updateMut.isPending}
          initial={{
            versionName: editing.versionName,
            title: editing.title ?? '',
            description: editing.description ?? '',
            releaseDate: editing.releaseDate ?? '',
            state: editing.state,
            pdf: editing.pdfFileUrl
              ? {
                  url: editing.pdfFileUrl,
                  name: editing.pdfFileName ?? 'release.pdf',
                  size: editing.pdfFileSize ?? 0,
                }
              : null,
          }}
          onCancel={() => setEditing(null)}
          onSubmit={(form) => updateMut.mutate({ id: editing.id, body: form })}
        />
      )}

      {confirmDelete && (
        <Dialog
          open
          onClose={() => (deleteMut.isPending ? undefined : setConfirmDelete(null))}
          className="relative z-40"
        >
          <div className="fixed inset-0 bg-black/40" aria-hidden />
          <div className="fixed inset-0 grid place-items-center p-4">
            <DialogPanel className="w-full max-w-md rounded-xl bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-6 space-y-4">
              <DialogTitle className="text-lg font-semibold">
                Delete release?
              </DialogTitle>
              <p className="text-sm text-zinc-600 dark:text-zinc-300">
                This removes <span className="font-mono">{confirmDelete.versionName}</span>{' '}
                and unlinks it from{' '}
                <span className="font-medium">
                  {confirmDelete.entryCount}{' '}
                  {confirmDelete.entryCount === 1 ? 'entry' : 'entries'}
                </span>
                . The entries are kept; only the release reference is cleared. This
                action cannot be undone.
              </p>
              <div className="flex justify-end gap-2">
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => setConfirmDelete(null)}
                  disabled={deleteMut.isPending}
                >
                  Cancel
                </Button>
                <Button
                  type="button"
                  variant="danger"
                  onClick={() => deleteMut.mutate(confirmDelete.id)}
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

function ReleaseStateChip({ state }: { state: ApiReleaseState }) {
  if (state === 'completed') {
    return (
      <span
        title="Completed"
        className="inline-flex size-7 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-500/15"
      >
        <CheckCircle2 className="size-4" />
      </span>
    )
  }
  return (
    <span
      title="Planned"
      className="inline-flex size-7 shrink-0 items-center justify-center rounded-full bg-sky-100 text-sky-600 dark:bg-sky-500/15"
    >
      <Circle className="size-4" />
    </span>
  )
}

function formatDate(isoDate: string): string {
  const [y, m, d] = isoDate.split('-').map(Number)
  if (!y || !m || !d) return isoDate
  const date = new Date(Date.UTC(y, m - 1, d))
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function ReleaseFormDialog({
  title,
  submitLabel,
  initial,
  isPending,
  onCancel,
  onSubmit,
}: {
  title: string
  submitLabel: string
  initial: FormState
  isPending: boolean
  onCancel: () => void
  onSubmit: (form: FormState) => void
}) {
  const [form, setForm] = useState<FormState>(initial)

  useEffect(() => {
    setForm(initial)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(initial)])

  return (
    <Dialog
      open
      onClose={() => (isPending ? undefined : onCancel())}
      className="relative z-40"
    >
      <div className="fixed inset-0 bg-black/40" aria-hidden />
      <div className="fixed inset-0 grid place-items-center p-4">
        <DialogPanel className="w-full max-w-lg rounded-xl bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-6 space-y-4">
          <DialogTitle className="text-lg font-semibold">{title}</DialogTitle>
          <form
            onSubmit={(ev) => {
              ev.preventDefault()
              if (!form.versionName.trim()) return
              onSubmit(form)
            }}
            className="space-y-4"
          >
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-zinc-500 mb-1">
                  Version name
                </label>
                <input
                  value={form.versionName}
                  onChange={(e) => setForm({ ...form, versionName: e.target.value })}
                  placeholder="v1.2.0"
                  className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
                  autoFocus
                  maxLength={60}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-zinc-500 mb-1">
                  Release date
                </label>
                <input
                  type="date"
                  value={form.releaseDate}
                  onChange={(e) => setForm({ ...form, releaseDate: e.target.value })}
                  className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
                />
              </div>
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1">
                Title
              </label>
              <input
                value={form.title}
                onChange={(e) => setForm({ ...form, title: e.target.value })}
                placeholder="The April release"
                className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
                maxLength={200}
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1">
                Description
              </label>
              <MarkdownEditor
                value={form.description}
                onChange={(next) =>
                  setForm({ ...form, description: next.slice(0, 10000) })
                }
                placeholder="Markdown supported. Paste, drop, or attach images."
                rows={6}
                upload={(file) => API().console.uploadFile(file)}
              />
              <p className="mt-1 text-xs text-zinc-500">
                Rendered above the linked entries on the Portal changelog.
              </p>
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1">
                PDF attachment
              </label>
              <PdfField
                value={form.pdf}
                onChange={(next) => setForm({ ...form, pdf: next })}
                disabled={isPending}
              />
              <p className="mt-1 text-xs text-zinc-500">
                Optional. Surfaced as a download link on the Portal
                changelog page.
              </p>
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1">
                State
              </label>
              <div className="flex gap-2">
                {(['planned', 'completed'] as const).map((s) => (
                  <button
                    type="button"
                    key={s}
                    onClick={() => setForm({ ...form, state: s })}
                    className={clsx(
                      'rounded-md border px-3 py-1.5 text-sm capitalize transition-colors',
                      form.state === s
                        ? 'border-sky-500 bg-sky-50 text-sky-700 dark:bg-sky-500/10 dark:text-sky-300'
                        : 'border-zinc-300 dark:border-zinc-700 text-zinc-600 dark:text-zinc-300 hover:bg-zinc-50 dark:hover:bg-zinc-800',
                    )}
                  >
                    {s}
                  </button>
                ))}
              </div>
              <p className="mt-1 text-xs text-zinc-500">
                Only completed releases appear on the Portal changelog.
              </p>
            </div>
            <div className="flex justify-end gap-2">
              <Button
                type="button"
                variant="ghost"
                onClick={onCancel}
                disabled={isPending}
              >
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending}>
                {submitLabel}
              </Button>
            </div>
          </form>
        </DialogPanel>
      </div>
    </Dialog>
  )
}

function PdfField({
  value,
  onChange,
  disabled,
}: {
  value: FormPdf | null
  onChange: (next: FormPdf | null) => void
  disabled: boolean
}) {
  const inputRef = useRef<HTMLInputElement | null>(null)
  const [uploading, setUploading] = useState(false)

  const handlePick = () => {
    if (disabled || uploading) return
    inputRef.current?.click()
  }

  const handleFile = async (file: File) => {
    if (file.type !== 'application/pdf') {
      message('Only PDF files are supported.')
      return
    }
    setUploading(true)
    try {
      const uploaded = await API().console.uploadFile(file)
      onChange({
        url: uploaded.url,
        name: uploaded.originalName || file.name,
        size: uploaded.size,
      })
    } catch (err) {
      message(err)
    } finally {
      setUploading(false)
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  return (
    <div>
      <input
        ref={inputRef}
        type="file"
        accept="application/pdf,.pdf"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0]
          if (file) void handleFile(file)
        }}
      />
      {value ? (
        <div className="flex items-center gap-3 rounded-md border border-zinc-300 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950/40 px-3 py-2">
          <FileText className="size-4 shrink-0 text-zinc-500" />
          <a
            href={value.url}
            target="_blank"
            rel="noopener noreferrer"
            className="min-w-0 flex-1 truncate text-sm text-sky-700 hover:underline dark:text-sky-300"
          >
            {value.name}
          </a>
          {value.size > 0 && (
            <span className="shrink-0 text-xs text-zinc-500 tabular-nums">
              {formatBytes(value.size)}
            </span>
          )}
          <button
            type="button"
            onClick={handlePick}
            disabled={disabled || uploading}
            className="shrink-0 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 px-2 py-1 text-xs text-zinc-600 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800 disabled:opacity-50"
          >
            {uploading ? 'Uploading…' : 'Replace'}
          </button>
          <button
            type="button"
            onClick={() => onChange(null)}
            disabled={disabled || uploading}
            title="Remove"
            className="shrink-0 rounded p-1 text-zinc-500 hover:bg-rose-50 dark:hover:bg-rose-500/10 hover:text-rose-600 disabled:opacity-50"
          >
            <X className="size-4" />
          </button>
        </div>
      ) : (
        <button
          type="button"
          onClick={handlePick}
          disabled={disabled || uploading}
          className="inline-flex items-center gap-2 rounded-md border border-dashed border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm text-zinc-600 dark:text-zinc-300 hover:bg-zinc-50 dark:hover:bg-zinc-800 disabled:opacity-50"
        >
          <Paperclip className="size-4" />
          {uploading ? 'Uploading…' : 'Attach PDF'}
        </button>
      )}
    </div>
  )
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
