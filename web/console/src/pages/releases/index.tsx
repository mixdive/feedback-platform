import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CalendarDays, Pencil, Plus, Trash2 } from 'lucide-react'

import Button from '@/components/button'
import EntryTypeCountChips from '@/components/entry-type-counts'
import ReleaseFormDialog, {
  emptyForm,
  formatDate,
  ReleaseStateChip,
  type FormState,
} from '@/components/release-form-dialog'
import { API, type ApiReleaseListItem } from '@/services/api'
import { useAppSelector } from '@/store/hooks'
import { message } from '@/utils/helpers'

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
            Plan and manage releases. Open a release to attach entries directly;
            you can also pick a release from an entry's edit form. Completed
            releases automatically appear on the Portal Changelog.
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
                <Link to={`/releases/${r.id}`} className="min-w-0 flex-1 group">
                  <div className="flex items-center gap-2">
                    <span className="rounded-md bg-zinc-100 dark:bg-zinc-800 px-2 py-0.5 font-mono text-xs text-zinc-700 dark:text-zinc-200 group-hover:text-sky-700 dark:group-hover:text-sky-300">
                      {r.versionName}
                    </span>
                    {r.title && (
                      <span className="truncate text-sm text-zinc-700 dark:text-zinc-200 group-hover:underline">
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
                </Link>
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
