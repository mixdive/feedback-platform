import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Sparkles, Trash2 } from 'lucide-react'

import Button from '@/components/button'
import EntryTypeCountChips from '@/components/entry-type-counts'
import TopicBadge from '@/components/topic-badge'
import { API, type ApiEntryTopicListItem } from '@/services/api'
import { useAppSelector } from '@/store/hooks'
import { message } from '@/utils/helpers'

const palette = [
  '#0EA5E9',
  '#10B981',
  '#6366F1',
  '#A855F7',
  '#EC4899',
  '#F59E0B',
  '#DC2626',
  '#94A3B8',
  '#7C2D12',
]

type FormState = {
  title: string
  description: string
  color: string
  sortOrder: number
}

const emptyForm: FormState = {
  title: '',
  description: '',
  color: palette[0],
  sortOrder: 0,
}

export default function TopicsPage() {
  const queryClient = useQueryClient()
  const isAdmin = useAppSelector((s) =>
    (s.auth.user?.roles ?? []).includes('admin'),
  )

  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'entry-topics'],
    queryFn: () => API().console.listEntryTopics(),
  })

  const [editing, setEditing] = useState<ApiEntryTopicListItem | null>(null)
  const [creating, setCreating] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState<ApiEntryTopicListItem | null>(null)

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['console', 'entry-topics'] })
    void queryClient.invalidateQueries({ queryKey: ['console', 'entries'] })
  }

  const createMut = useMutation({
    mutationFn: (body: FormState) =>
      API().console.createEntryTopic({
        title: body.title.trim(),
        description: body.description.trim() || undefined,
        color: body.color || undefined,
        sortOrder: body.sortOrder,
      }),
    onSuccess: () => {
      setCreating(false)
      message('Topic created', 'success')
      invalidate()
    },
    onError: (e) => message(e),
  })

  const updateMut = useMutation({
    mutationFn: ({ id, body }: { id: string; body: FormState }) =>
      API().console.updateEntryTopic(id, {
        title: body.title.trim(),
        description: body.description.trim(),
        color: body.color,
        sortOrder: body.sortOrder,
      }),
    onSuccess: () => {
      setEditing(null)
      message('Topic updated', 'success')
      invalidate()
    },
    onError: (e) => message(e),
  })

  const deleteMut = useMutation({
    mutationFn: (id: string) => API().console.deleteEntryTopic(id),
    onSuccess: () => {
      setConfirmDelete(null)
      message('Topic deleted', 'success')
      invalidate()
    },
    onError: (e) => message(e),
  })

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Topics</h1>
          <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
            Topics group entries by area of your product (sign up, feed,
            settings, billing…). They are admin-managed and visible only on
            the Console — never on the Portal. The AI assigns a topic when a
            new entry is created without one and can create a new topic when
            none of the existing ones fit.
          </p>
        </div>
        {isAdmin && (
          <Button size="sm" onClick={() => setCreating(true)}>
            <Plus className="size-4" />
            New topic
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
            No topics yet. Create one to start grouping entries — or let the AI
            pick one for you on the next submission.
          </p>
        )}
        {data && data.data.length > 0 && (
          <ul className="divide-y divide-zinc-200 dark:divide-zinc-800">
            {data.data.map((t) => (
              <li
                key={t.id}
                className="flex items-center gap-3 px-4 py-3 hover:bg-zinc-50 dark:hover:bg-zinc-950/40"
              >
                <TopicBadge topic={t} />
                <div className="min-w-0 flex-1">
                  {t.description ? (
                    <p className="text-sm text-zinc-600 dark:text-zinc-300">
                      {t.description}
                    </p>
                  ) : (
                    <p className="text-sm italic text-zinc-400">No description</p>
                  )}
                </div>
                {t.source === 'ai' && (
                  <span
                    title="Created by AI"
                    className="inline-flex shrink-0 items-center gap-1 rounded-full border border-violet-300 dark:border-violet-700 bg-violet-50 dark:bg-violet-500/10 px-2 py-0.5 text-[10px] uppercase tracking-wide text-violet-600 dark:text-violet-300"
                  >
                    <Sparkles className="size-3" />
                    AI
                  </span>
                )}
                <EntryTypeCountChips counts={t.entryTypeCounts} />
                {t.entryCount > 0 ? (
                  <Link
                    to={`/entry?topicId=${t.id}`}
                    title={`View entries in ${t.title}`}
                    className="text-xs tabular-nums text-sky-700 dark:text-sky-300 hover:underline shrink-0 min-w-[4rem] text-right"
                  >
                    {t.entryCount} {t.entryCount === 1 ? 'entry' : 'entries'}
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
                      onClick={() => setEditing(t)}
                      className="rounded p-1.5 text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800 hover:text-zinc-800 dark:hover:text-zinc-200"
                    >
                      <Pencil className="size-4" />
                    </button>
                    <button
                      type="button"
                      title="Delete"
                      onClick={() => setConfirmDelete(t)}
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
        <TopicFormDialog
          title="New topic"
          submitLabel="Create"
          isPending={createMut.isPending}
          initial={emptyForm}
          onCancel={() => setCreating(false)}
          onSubmit={(form) => createMut.mutate(form)}
        />
      )}

      {editing && (
        <TopicFormDialog
          title="Edit topic"
          submitLabel="Save"
          isPending={updateMut.isPending}
          initial={{
            title: editing.title,
            description: editing.description ?? '',
            color: editing.color || palette[0],
            sortOrder: editing.sortOrder,
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
              <DialogTitle className="text-lg font-semibold">Delete topic?</DialogTitle>
              <p className="text-sm text-zinc-600 dark:text-zinc-300">
                This removes <TopicBadge topic={confirmDelete} /> from every
                entry that references it. This action cannot be undone.
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

function TopicFormDialog({
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
        <DialogPanel className="w-full max-w-md rounded-xl bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-6 space-y-4">
          <DialogTitle className="text-lg font-semibold">{title}</DialogTitle>
          <form
            onSubmit={(ev) => {
              ev.preventDefault()
              if (!form.title.trim()) return
              onSubmit(form)
            }}
            className="space-y-4"
          >
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1">
                Title
              </label>
              <input
                value={form.title}
                onChange={(e) => setForm({ ...form, title: e.target.value })}
                placeholder="Sign up, Feed, Settings…"
                className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
                autoFocus
                maxLength={60}
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1">
                Description
              </label>
              <textarea
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
                placeholder="What does this topic cover?"
                rows={3}
                maxLength={240}
                className="block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm focus:border-sky-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1">
                Color
              </label>
              <div className="flex flex-wrap items-center gap-2">
                {palette.map((c) => (
                  <button
                    key={c}
                    type="button"
                    onClick={() => setForm({ ...form, color: c })}
                    aria-label={`Pick ${c}`}
                    className="size-7 rounded-full border-2 transition-transform hover:scale-110"
                    style={{
                      backgroundColor: c,
                      borderColor: form.color === c ? '#0f172a' : 'transparent',
                    }}
                  />
                ))}
                <input
                  type="text"
                  value={form.color}
                  onChange={(e) => setForm({ ...form, color: e.target.value })}
                  placeholder="#RRGGBB"
                  className="ml-2 w-28 h-8 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 text-xs font-mono focus:border-sky-500 focus:outline-none"
                  maxLength={32}
                />
              </div>
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1">
                Sort order
              </label>
              <input
                type="number"
                value={form.sortOrder}
                onChange={(e) =>
                  setForm({ ...form, sortOrder: Number(e.target.value) || 0 })
                }
                className="block w-32 h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
              />
              <p className="mt-1 text-xs text-zinc-500">
                Lower values come first in pickers and lists.
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
