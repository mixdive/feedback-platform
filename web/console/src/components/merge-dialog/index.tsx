import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { GitMerge, X } from 'lucide-react'

import Button from '@/components/button'
import { API, type ApiEntry } from '@/services/api'
import { message } from '@/utils/helpers'

interface Props {
  entry: ApiEntry
  open: boolean
  onClose: () => void
}

export default function MergeDialog({ entry, open, onClose }: Props) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [debounced, setDebounced] = useState('')
  const [selected, setSelected] = useState<{ id: string; title: string } | null>(null)

  useEffect(() => {
    if (!open) {
      setSearch('')
      setDebounced('')
      setSelected(null)
    }
  }, [open])

  useEffect(() => {
    const t = window.setTimeout(() => setDebounced(search.trim()), 200)
    return () => window.clearTimeout(t)
  }, [search])

  const { data, isFetching } = useQuery({
    queryKey: ['console', 'entries', 'merge-search', debounced],
    queryFn: () =>
      API().console.listEntries({
        search: debounced,
        limit: 10,
      }),
    enabled: open && debounced.length > 0,
  })

  const candidates = useMemo(() => {
    if (!data) return []
    return data.data.filter((e) => e.id !== entry.id)
  }, [data, entry.id])

  const mergeMut = useMutation({
    mutationFn: (targetEntryId: string) =>
      API().console.mergeEntry(entry.id, { targetEntryId }),
    onSuccess: (updated, targetEntryId) => {
      queryClient.setQueryData(['console', 'entries', entry.id], updated)
      void queryClient.invalidateQueries({ queryKey: ['console', 'entries'] })
      void queryClient.invalidateQueries({ queryKey: ['console', 'activities', entry.id] })
      message('Merged. Redirecting…', 'success')
      onClose()
      navigate(`/entry/${targetEntryId}`)
    },
    onError: (e) => message(e),
  })

  return (
    <Dialog
      open={open}
      onClose={() => (mergeMut.isPending ? undefined : onClose())}
      className="relative z-40"
    >
      <div className="fixed inset-0 bg-black/40" aria-hidden />
      <div className="fixed inset-0 grid place-items-center p-4">
        <DialogPanel className="w-full max-w-lg rounded-xl bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-6 space-y-4">
          <div className="flex items-start justify-between gap-3">
            <div className="flex items-center gap-2">
              <GitMerge className="size-5 text-zinc-500" />
              <DialogTitle className="text-lg font-semibold">
                Merge into another entry
              </DialogTitle>
            </div>
            <button
              type="button"
              onClick={onClose}
              disabled={mergeMut.isPending}
              className="inline-flex size-7 items-center justify-center rounded-md text-zinc-500 hover:bg-zinc-100 disabled:opacity-50 dark:hover:bg-zinc-800"
            >
              <X className="size-4" />
            </button>
          </div>

          <p className="text-sm text-zinc-600 dark:text-zinc-300">
            This will mark{' '}
            <span className="font-medium text-zinc-900 dark:text-zinc-100">
              {entry.title || '(untitled)'}
            </span>{' '}
            as a duplicate, transfer its unique votes to the target, cancel
            it, and add a system comment recording the merge.
          </p>

          {selected ? (
            <div className="flex items-center justify-between gap-3 rounded-md border border-zinc-200 bg-zinc-50 px-3 py-2 text-sm dark:border-zinc-800 dark:bg-zinc-950">
              <div className="min-w-0">
                <div className="text-xs uppercase tracking-wide text-zinc-500">
                  Merge target
                </div>
                <div className="truncate font-medium">
                  {selected.title || '(untitled)'}
                </div>
              </div>
              <button
                type="button"
                onClick={() => {
                  setSelected(null)
                  setSearch('')
                }}
                disabled={mergeMut.isPending}
                className="text-xs text-zinc-500 hover:text-zinc-800 disabled:opacity-50 dark:hover:text-zinc-200"
              >
                Change
              </button>
            </div>
          ) : (
            <div>
              <input
                value={search}
                autoFocus
                onChange={(ev) => setSearch(ev.target.value)}
                placeholder="Search entries to merge into…"
                className="block w-full h-9 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
              />
              {debounced.length > 0 && (
                <div className="mt-2 max-h-60 overflow-y-auto rounded-md border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900">
                  {isFetching && (
                    <div className="p-2 text-xs text-zinc-500">Searching…</div>
                  )}
                  {!isFetching && candidates.length === 0 && (
                    <div className="p-2 text-xs text-zinc-500">No matches.</div>
                  )}
                  {candidates.map((e) => (
                    <button
                      type="button"
                      key={e.id}
                      onClick={() => setSelected({ id: e.id, title: e.title })}
                      className="block w-full truncate border-b border-zinc-100 px-3 py-2 text-left text-sm last:border-b-0 hover:bg-zinc-50 dark:border-zinc-800 dark:hover:bg-zinc-800"
                    >
                      {e.title || '(untitled)'}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}

          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="ghost"
              onClick={onClose}
              disabled={mergeMut.isPending}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="danger"
              isLoading={mergeMut.isPending}
              disabled={!selected}
              onClick={() => {
                if (!selected) return
                mergeMut.mutate(selected.id)
              }}
            >
              Merge
            </Button>
          </div>
        </DialogPanel>
      </div>
    </Dialog>
  )
}
