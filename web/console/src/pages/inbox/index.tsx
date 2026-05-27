import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronUp, CircleDot, Globe } from 'lucide-react'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'

import EntryTypeBadge from '@/components/entry-type-badge'
import InternalBadge from '@/components/internal-badge'
import { markdownToPlainText } from '@/components/markdown'
import ReleaseBadge from '@/components/release-badge'
import TopicBadge from '@/components/topic-badge'
import {
  API,
  type ApiEntryCreator,
  type ApiEntryStatusValue,
} from '@/services/api'
import { ENTRY_STATUSES } from '@/utils/entry-status'
import { message } from '@/utils/helpers'

dayjs.extend(relativeTime)

function creatorLabel(c?: ApiEntryCreator): string {
  if (!c) return 'Anonymous'
  return c.name || c.username || 'Anonymous'
}

// Inbox scope is hardcoded: every entry whose status is "new",
// regardless of category. The list is triage-ordered (newest first).
export default function InboxPage() {
  const queryClient = useQueryClient()

  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'inbox'],
    queryFn: () =>
      API().console.listEntries({
        status: 'new',
        sort: 'new',
        limit: 50,
      }),
  })

  const statusMut = useMutation({
    mutationFn: ({ id, status }: { id: string; status: ApiEntryStatusValue }) =>
      API().console.updateEntry(id, { status }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['console', 'inbox'] })
      void queryClient.invalidateQueries({ queryKey: ['console', 'entries'] })
    },
    onError: (e) => message(e),
  })

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-semibold">Inbox</h1>
        <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
          New entries waiting for triage.
        </p>
      </div>

      {isLoading && (
        <div className="rounded-xl border border-zinc-200/70 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-soft px-4 py-8 text-sm text-zinc-500">
          Loading…
        </div>
      )}
      {error && (
        <div className="rounded-xl border border-zinc-200/70 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-soft px-4 py-8 text-sm text-rose-600">
          {(error as Error).message}
        </div>
      )}

      {data && data.data.length === 0 && (
        <div className="rounded-xl border border-zinc-200/70 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-soft px-4 py-12 text-center text-sm text-zinc-500">
          No new entries right now.
        </div>
      )}

      {data && data.data.length > 0 && (
        <ul className="space-y-3">
          {data.data.map((f) => {
            const isPublic = !f.isInternal
            return (
            <li
              key={f.id}
              className={
                f.isInternal
                  ? 'flex items-start gap-3 rounded-xl border border-amber-300/70 dark:border-amber-500/40 bg-amber-50/60 dark:bg-amber-500/[0.06] px-4 py-3 shadow-soft transition-all duration-150 hover:border-amber-400 hover:shadow-pop dark:hover:border-amber-400/60'
                  : 'flex items-start gap-3 rounded-xl border border-zinc-200/70 dark:border-zinc-800 bg-white dark:bg-zinc-900 px-4 py-3 shadow-soft transition-all duration-150 hover:border-zinc-300 hover:shadow-pop dark:hover:border-zinc-700'
              }
            >
              <CircleDot
                className="mt-0.5 size-4 shrink-0 text-emerald-600 dark:text-emerald-400"
                aria-label="Open"
              />
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                  <Link
                    to={`/entry/${f.id}`}
                    state={{ from: 'inbox' }}
                    className="font-medium text-zinc-900 dark:text-zinc-50 hover:underline focus:outline-none focus:underline"
                  >
                    {f.title}
                  </Link>
                  {isPublic && (
                    <span
                      title="Visible on the Portal"
                      className="inline-flex items-center gap-1 rounded-full border border-emerald-300 dark:border-emerald-500/40 bg-emerald-50 dark:bg-emerald-500/10 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-emerald-700 dark:text-emerald-300"
                    >
                      <Globe className="size-3" />
                      Public
                    </span>
                  )}
                  {f.entryType && f.entryType.value !== 'other' && (
                    <EntryTypeBadge entryType={f.entryType} appliedByAI={f.entryTypeAppliedByAI} />
                  )}
                  {f.topics.map((t) => (
                    <TopicBadge
                      key={t.id}
                      topic={t}
                      size="sm"
                      appliedByAI={f.aiTopicIds?.includes(t.id)}
                    />
                  ))}
                  {f.release && <ReleaseBadge release={f.release} size="sm" />}
                  {f.isInternal && <InternalBadge />}
                </div>
                {f.description && (
                  <p className="mt-0.5 line-clamp-1 text-sm text-zinc-600 dark:text-zinc-400">
                    {markdownToPlainText(f.description)}
                  </p>
                )}
                <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-zinc-500">
                  <span className="inline-flex items-center gap-1">
                    <ChevronUp className="size-3.5" />
                    <span className="tabular-nums">{f.voteCount}</span>
                    <span>{f.voteCount === 1 ? 'vote' : 'votes'}</span>
                  </span>
                  <span className="opacity-50">·</span>
                  <span>
                    opened {dayjs(f.createdAt).fromNow()} by{' '}
                    <span className="text-zinc-700 dark:text-zinc-300">
                      {creatorLabel(f.creator)}
                    </span>
                  </span>
                </div>
              </div>
              <div className="shrink-0 self-center">
                <InboxStatusChanger
                  currentStatus={f.status?.value ?? 'new'}
                  pending={
                    statusMut.isPending && statusMut.variables?.id === f.id
                  }
                  onChange={(status) =>
                    statusMut.mutate({ id: f.id, status })
                  }
                />
              </div>
            </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}

// Inbox-only quick status changer. The Console "list cells read-only"
// convention does not apply to the inbox — its purpose is triage, so an
// inline status mutation is the primary action on every row.
function InboxStatusChanger({
  currentStatus,
  pending,
  onChange,
}: {
  currentStatus: ApiEntryStatusValue
  pending: boolean
  onChange: (next: ApiEntryStatusValue) => void
}) {
  return (
    <select
      value={currentStatus}
      disabled={pending}
      onChange={(e) => onChange(e.target.value as ApiEntryStatusValue)}
      onClick={(e) => e.stopPropagation()}
      aria-label="Change status"
      className="h-8 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 text-sm transition-colors focus:border-brand-500 focus:outline-none disabled:opacity-50"
    >
      {ENTRY_STATUSES.map((s) => (
        <option key={s.value} value={s.value}>
          {s.title}
        </option>
      ))}
    </select>
  )
}
