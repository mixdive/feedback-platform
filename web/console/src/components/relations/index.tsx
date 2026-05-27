import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link2, Plus, Sparkles, Trash2, X } from 'lucide-react'
import clsx from 'clsx'

import Button from '@/components/button'
import InternalBadge from '@/components/internal-badge'
import {
  API,
  type ApiEntry,
  type ApiEntryRelation,
  type ApiEntryRelationType,
} from '@/services/api'
import { message } from '@/utils/helpers'

// Relation types are a hardcoded enum (see models.EntryRelationType). The
// Console UI lists them in fixed order; admins cannot define new types.
const RELATION_TYPES: { value: ApiEntryRelationType; label: string }[] = [
  { value: 'related', label: 'Related' },
  { value: 'duplicate', label: 'Duplicate' },
]

interface Props {
  entry: ApiEntry
}

export default function Relations({ entry }: Props) {
  const queryClient = useQueryClient()
  const [isAdding, setIsAdding] = useState(false)

  const refresh = (updated: ApiEntry) => {
    queryClient.setQueryData(['console', 'entries', entry.id], updated)
    void queryClient.invalidateQueries({ queryKey: ['console', 'entries'] })
    // Peer entry's cached detail also stale — invalidate so a sibling
    // tab refetches its relations on next view.
    void queryClient.invalidateQueries({ queryKey: ['console', 'entries'], exact: false })
    // Both peers receive a mirrored activity row server-side; the
    // current entry's timeline picks it up here and the peer's
    // timeline rehydrates next time it opens.
    void queryClient.invalidateQueries({ queryKey: ['console', 'activities', entry.id] })
  }

  const removeMut = useMutation({
    mutationFn: (peerId: string) => API().console.deleteEntryRelation(entry.id, peerId),
    onSuccess: refresh,
    onError: (e) => message(e),
  })

  const addMut = useMutation({
    mutationFn: (body: { peerEntryId: string; type: ApiEntryRelationType }) =>
      API().console.addEntryRelation(entry.id, body),
    onSuccess: (updated) => {
      refresh(updated)
      setIsAdding(false)
      message('Relation added', 'success')
    },
    onError: (e) => message(e),
  })

  const total = entry.relations.length

  return (
    <section className="space-y-4">
      <div className="flex items-center justify-between gap-2 border-b border-zinc-200 dark:border-zinc-800 pb-2">
        <div className="flex items-center gap-2">
          <Link2 className="size-4 text-zinc-500" />
          <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-600 dark:text-zinc-400">
            Relations ({total})
          </h2>
        </div>
        {!isAdding && (
          <Button variant="ghost" size="sm" onClick={() => setIsAdding(true)}>
            <Plus className="size-4" />
            Add relation
          </Button>
        )}
      </div>

      {isAdding && (
        <AddRelationForm
          entryId={entry.id}
          existingPeerIds={useMemo(
            () => new Set(entry.relations.map((r) => r.peer.id)),
            [entry.relations],
          )}
          onCancel={() => setIsAdding(false)}
          onSubmit={(peerEntryId, type) => addMut.mutate({ peerEntryId, type })}
          isPending={addMut.isPending}
        />
      )}

      {total === 0 ? (
        <p className="text-sm italic text-zinc-500">
          No relations. Use “Add relation” to link this entry with another.
        </p>
      ) : (
        <ul className="space-y-2">
          {entry.relations.map((rel) => (
            <RelationItem
              key={`${rel.peer.id}|${rel.type}`}
              relation={rel}
              onRemove={() => removeMut.mutate(rel.peer.id)}
              isRemoving={
                removeMut.isPending && removeMut.variables === rel.peer.id
              }
            />
          ))}
        </ul>
      )}
    </section>
  )
}

function RelationItem({
  relation,
  onRemove,
  isRemoving,
}: {
  relation: ApiEntryRelation
  onRemove: () => void
  isRemoving: boolean
}) {
  return (
    <li className="flex items-start justify-between gap-3 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-3">
      <div className="min-w-0 flex-1">
        <div className="mb-1 flex flex-wrap items-center gap-2">
          <RelationTypeChip type={relation.type} />
          {relation.isAI && (
            <span
              className="inline-flex items-center gap-1 rounded bg-violet-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-violet-800 dark:bg-violet-900/40 dark:text-violet-200"
              title="Applied by the AI relation analyzer"
            >
              <Sparkles className="size-3" />
              AI
            </span>
          )}
          {relation.peer.isInternal && <InternalBadge />}
        </div>
        <Link
          to={`/entry/${relation.peer.id}`}
          className="block truncate text-sm font-medium text-zinc-800 hover:text-sky-600 dark:text-zinc-200 dark:hover:text-sky-400"
        >
          {relation.peer.title || '(untitled)'}
        </Link>
        <div className="mt-0.5 text-xs text-zinc-500">
          {relation.peer.voteCount}{' '}
          {relation.peer.voteCount === 1 ? 'vote' : 'votes'}
        </div>
      </div>
      <button
        type="button"
        onClick={onRemove}
        disabled={isRemoving}
        title="Remove relation"
        className="inline-flex size-8 shrink-0 items-center justify-center rounded-md text-zinc-500 hover:bg-zinc-100 hover:text-rose-600 disabled:opacity-50 dark:hover:bg-zinc-800"
      >
        <Trash2 className="size-4" />
      </button>
    </li>
  )
}

function RelationTypeChip({ type }: { type: ApiEntryRelationType }) {
  const tone =
    type === 'duplicate'
      ? 'bg-rose-100 text-rose-800 dark:bg-rose-900/40 dark:text-rose-200'
      : 'bg-sky-100 text-sky-800 dark:bg-sky-900/40 dark:text-sky-200'
  const label =
    RELATION_TYPES.find((t) => t.value === type)?.label ?? String(type)
  return (
    <span
      className={clsx(
        'inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide',
        tone,
      )}
    >
      {label}
    </span>
  )
}

function AddRelationForm({
  entryId,
  existingPeerIds,
  onCancel,
  onSubmit,
  isPending,
}: {
  entryId: string
  existingPeerIds: Set<string>
  onCancel: () => void
  onSubmit: (peerEntryId: string, type: ApiEntryRelationType) => void
  isPending: boolean
}) {
  const [search, setSearch] = useState('')
  const [debounced, setDebounced] = useState('')
  const [selected, setSelected] = useState<{ id: string; title: string } | null>(null)
  const [type, setType] = useState<ApiEntryRelationType>('related')

  useEffect(() => {
    const t = window.setTimeout(() => setDebounced(search.trim()), 200)
    return () => window.clearTimeout(t)
  }, [search])

  const { data, isFetching } = useQuery({
    queryKey: ['console', 'entries', 'relation-search', debounced],
    queryFn: () =>
      API().console.listEntries({
        search: debounced,
        limit: 10,
      }),
    enabled: debounced.length > 0,
  })

  const candidates = useMemo(() => {
    if (!data) return []
    return data.data.filter(
      (e) => e.id !== entryId && !existingPeerIds.has(e.id),
    )
  }, [data, entryId, existingPeerIds])

  return (
    <div className="space-y-3 rounded-lg border border-sky-200 dark:border-sky-900 bg-sky-50/40 dark:bg-sky-950/20 p-3">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium uppercase tracking-wide text-zinc-600 dark:text-zinc-400">
          Add relation
        </span>
        <button
          type="button"
          onClick={onCancel}
          className="inline-flex size-7 items-center justify-center rounded-md text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800"
        >
          <X className="size-4" />
        </button>
      </div>

      {selected ? (
        <div className="flex items-center justify-between gap-3 rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm dark:border-zinc-800 dark:bg-zinc-900">
          <span className="truncate">{selected.title || '(untitled)'}</span>
          <button
            type="button"
            onClick={() => {
              setSelected(null)
              setSearch('')
            }}
            className="text-xs text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
          >
            Change
          </button>
        </div>
      ) : (
        <div>
          <input
            value={search}
            onChange={(ev) => setSearch(ev.target.value)}
            placeholder="Search entries to link…"
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

      <div>
        <label className="mb-1 block text-xs font-medium text-zinc-500">
          Relation type
        </label>
        <div className="flex gap-2">
          {RELATION_TYPES.map((t) => (
            <button
              key={t.value}
              type="button"
              onClick={() => setType(t.value)}
              className={clsx(
                'inline-flex items-center rounded-md border px-2.5 py-1 text-xs',
                type === t.value
                  ? 'border-sky-500 bg-sky-100 text-sky-800 dark:bg-sky-900/40 dark:text-sky-200'
                  : 'border-zinc-300 text-zinc-700 hover:bg-zinc-50 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800',
              )}
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>

      <div className="flex justify-end gap-2">
        <Button type="button" variant="ghost" onClick={onCancel}>
          Cancel
        </Button>
        <Button
          type="button"
          isLoading={isPending}
          disabled={!selected}
          onClick={() => {
            if (!selected) return
            onSubmit(selected.id, type)
          }}
        >
          Add
        </Button>
      </div>
    </div>
  )
}
