import { useEffect, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { CheckCircle2, ChevronUp, CircleDot, Globe, Search, X } from 'lucide-react'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import clsx from 'clsx'

import EntryTypeBadge from '@/components/entry-type-badge'
import FilterMenu, { type FilterOption } from '@/components/filter-menu'
import InternalBadge from '@/components/internal-badge'
import { markdownToPlainText } from '@/components/markdown'
import StatusBadge from '@/components/status-badge'
import ReleaseBadge from '@/components/release-badge'
import TopicBadge from '@/components/topic-badge'
import TopicSelect from '@/components/topic-select'
import {
  API,
  type ApiEntryTypeValue,
  type ApiEntryCreator,
  type ApiEntryStatusValue,
} from '@/services/api'
import { ENTRY_TYPES } from '@/utils/entry-type'
import { ENTRY_STATUSES, entryStatusInfo } from '@/utils/entry-status'

dayjs.extend(relativeTime)

type Sort = 'new' | 'top'

function creatorLabel(c?: ApiEntryCreator): string {
  if (!c) return 'Anonymous'
  return c.name || c.username || 'Anonymous'
}

// EntriesView is the shared entries-list surface used by four pages:
// All Feedback (no lock), Feature Requests
// (lockedEntryType="feature-request"), Bugs ("bug"), and Support
// ("support"). When the type is locked the entry-type FilterMenu is
// hidden — per the locked-scope-dedicated-page pattern, locked filter
// controls are not shown — and the redundant entry-type badge is
// dropped from every row.
export type EntriesViewProps = {
  pageTitle: string
  lockedEntryType?: ApiEntryTypeValue
  // Enables the Release filter dropdown. Off by default so the Support
  // screen (which shares this view) doesn't show it; All Feedback,
  // Feature Requests, and Bugs opt in.
  enableReleaseFilter?: boolean
}

export default function EntriesView({
  pageTitle,
  lockedEntryType,
  enableReleaseFilter,
}: EntriesViewProps) {
  const [searchParams, setSearchParams] = useSearchParams()

  // The detail page reads location.state.from to pick its back-link target,
  // so each locked-scope list announces itself here.
  const fromKey =
    lockedEntryType === 'feature-request'
      ? 'feature-requests'
      : lockedEntryType === 'bug'
      ? 'bugs'
      : lockedEntryType === 'support'
      ? 'support'
      : undefined

  // Every filter lives in the URL so a filtered view is shareable,
  // survives refresh, and can be deep-linked to from elsewhere in the
  // Console — the Releases and Topics pages link here with a preset
  // releaseId / topicId. `sort` defaults to 'new' when absent.
  const setParam = (key: string, value: string) => {
    setSearchParams(
      (prev) => {
        const sp = new URLSearchParams(prev)
        if (!value) sp.delete(key)
        else sp.set(key, value)
        return sp
      },
      { replace: true },
    )
  }

  const sort: Sort = searchParams.get('sort') === 'top' ? 'top' : 'new'
  const setSort = (next: Sort) => setParam('sort', next === 'new' ? '' : next)
  const search = searchParams.get('q') ?? ''
  const filterEntryType = (searchParams.get('type') ?? '') as ApiEntryTypeValue | ''
  const filterStatus = (searchParams.get('status') ?? '') as ApiEntryStatusValue | ''
  const filterAuthorId = searchParams.get('author') ?? ''
  const filterTopicId = searchParams.get('topicId') ?? ''
  const filterReleaseId = searchParams.get('releaseId') ?? ''

  // The search box keeps responsive local state and writes its trimmed
  // value to `?q=` after a 250ms debounce. Seeded from the URL on mount
  // so a shared link pre-fills the box.
  const [searchInput, setSearchInput] = useState(search)
  // Route the debounced write through a latest-ref so it reads the
  // freshest URL params when the timer fires — without this, a filter
  // changed within the debounce window would be clobbered by a
  // stale-closure `q` write (setSearchParams's functional base is bound
  // to the render it was created in).
  const setParamRef = useRef(setParam)
  useEffect(() => {
    setParamRef.current = setParam
  })
  useEffect(() => {
    const t = setTimeout(() => setParamRef.current('q', searchInput.trim()), 250)
    return () => clearTimeout(t)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchInput])

  const effectiveEntryType: ApiEntryTypeValue | '' = lockedEntryType ?? filterEntryType
  // Release filter only applies on screens that opted in; ignore any
  // stray ?releaseId on the Support screen.
  const effectiveReleaseId = enableReleaseFilter ? filterReleaseId : ''

  const { data, isLoading, error } = useQuery({
    queryKey: [
      'console',
      'entries',
      {
        sort,
        search,
        filterEntryType: effectiveEntryType,
        filterStatus,
        filterAuthorId,
        filterTopicId,
        filterReleaseId: effectiveReleaseId,
      },
    ],
    queryFn: () =>
      API().console.listEntries({
        sort,
        search: search || undefined,
        entryType: effectiveEntryType || undefined,
        status: filterStatus || undefined,
        authorId: filterAuthorId || undefined,
        topicId: filterTopicId || undefined,
        releaseId: effectiveReleaseId || undefined,
        limit: 25,
      }),
  })

  const { data: authorsData } = useQuery({
    queryKey: ['console', 'entry-authors'],
    queryFn: () => API().console.listEntryAuthors(),
  })
  const authors = authorsData?.data

  const { data: topicsData } = useQuery({
    queryKey: ['console', 'entry-topics'],
    queryFn: () => API().console.listEntryTopics(),
  })
  const topics = topicsData?.data

  const { data: releasesData } = useQuery({
    queryKey: ['console', 'releases'],
    queryFn: () => API().console.listReleases(),
    enabled: enableReleaseFilter,
  })
  const releases = releasesData?.data

  const entryTypeOptions: FilterOption[] = [
    { id: '', label: 'All entry types' },
    ...ENTRY_TYPES.map((t) => ({ id: t.value, label: t.title })),
  ]
  const statusOptions: FilterOption[] = [
    { id: '', label: 'All statuses' },
    ...ENTRY_STATUSES.map((s) => ({ id: s.value, label: s.title })),
  ]
  const authorOptions: FilterOption[] = [
    { id: '', label: 'All authors' },
    ...(authors?.map((u) => ({
      id: u.id,
      label: u.name || u.username || 'Anonymous',
    })) ?? []),
  ]
  const releaseOptions: FilterOption[] = [
    { id: '', label: 'All releases' },
    ...(releases?.map((r) => ({
      id: r.id,
      label: r.versionName,
      hint: r.title || undefined,
    })) ?? []),
  ]
  // Surface the selected version in the trigger (like Sort) so a
  // deep-link from the Releases page reads "Release: v1.2.0" at a glance.
  const activeReleaseLabel = releases?.find(
    (r) => r.id === effectiveReleaseId,
  )?.versionName
  const sortOptions: FilterOption[] = [
    { id: 'new', label: 'Newest' },
    { id: 'top', label: 'Most votes' },
  ]
  // FilterMenu's trigger only shows the static `label` prop, so without
  // this the user can't tell at a glance that "Newest" is the default.
  // Surface the active option in the trigger label.
  const activeSortLabel =
    sortOptions.find((o) => o.id === sort)?.label ?? 'Sort'

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">{pageTitle}</h1>

      <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 overflow-hidden">
        {/* Top bar: search left, filter dropdowns right. Mirrors GitHub. */}
        <div className="flex flex-wrap items-center gap-3 border-b border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/40 px-3 py-2">
          <div className="relative flex-1 min-w-[12rem] max-w-xl">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-zinc-400" />
            <input
              type="search"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              placeholder="Search by title or description"
              className="block w-full h-8 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 pl-8 pr-8 text-sm focus:border-sky-500 focus:outline-none"
            />
            {searchInput && (
              <button
                type="button"
                onClick={() => setSearchInput('')}
                aria-label="Clear search"
                className="absolute right-2 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200"
              >
                <X className="size-4" />
              </button>
            )}
          </div>
          <div className="flex flex-wrap items-center gap-4">
            <FilterMenu
              label="Author"
              options={authorOptions}
              value={filterAuthorId}
              onChange={(id) => setParam('author', id)}
            />
            {!lockedEntryType && (
              <FilterMenu
                label="Entry Types"
                options={entryTypeOptions}
                value={filterEntryType}
                onChange={(id) => setParam('type', id)}
              />
            )}
            <FilterMenu
              label="Statuses"
              options={statusOptions}
              value={filterStatus}
              onChange={(id) => setParam('status', id)}
            />
            <div className="flex items-center gap-2">
              <span className="text-sm text-zinc-600 dark:text-zinc-300">Topic</span>
              <TopicSelect
                topics={topics}
                value={filterTopicId}
                onChange={(id) => setParam('topicId', id)}
                placeholder="Filter"
                align="right"
                emptyLabel="All topics"
              />
            </div>
            {enableReleaseFilter && (
              <FilterMenu
                label={
                  activeReleaseLabel ? `Release: ${activeReleaseLabel}` : 'Release'
                }
                options={releaseOptions}
                value={effectiveReleaseId}
                onChange={(id) => setParam('releaseId', id)}
              />
            )}
            <FilterMenu
              label={`Sort: ${activeSortLabel}`}
              options={sortOptions}
              value={sort}
              onChange={(id) => setSort(id as Sort)}
            />
          </div>
        </div>

        {isLoading && (
          <p className="px-4 py-8 text-sm text-zinc-500">Loading…</p>
        )}
        {error && (
          <p className="px-4 py-8 text-sm text-rose-600">
            {(error as Error).message}
          </p>
        )}

        {data && data.data.length === 0 && (
          <div className="px-4 py-12 text-center text-sm text-zinc-500">
            {search ? `No entries match "${search}".` : 'No entries yet.'}
          </div>
        )}

        {data && data.data.length > 0 && (
          <ul className="divide-y divide-zinc-200 dark:divide-zinc-800">
            {data.data.map((f) => {
              const isClosed = entryStatusInfo(f.status?.value).closed
              const StateIcon = isClosed ? CheckCircle2 : CircleDot
              const stateColor = isClosed
                ? 'text-violet-600 dark:text-violet-400'
                : 'text-emerald-600 dark:text-emerald-400'
              // Public = visible on the Portal. Internal records are
              // Console-only; flagging the public ones makes the
              // Console reader's "who can see this?" answer obvious
              // without inspecting each row's status badge.
              const isPublic = !f.isInternal

              return (
                <li
                  key={f.id}
                  className={clsx(
                    'flex items-start gap-3 px-4 py-3 transition-colors',
                    f.isInternal
                      ? 'bg-amber-50/60 hover:bg-amber-50 dark:bg-amber-500/[0.06] dark:hover:bg-amber-500/[0.1]'
                      : isClosed
                      ? 'bg-zinc-50/80 hover:bg-zinc-100/60 dark:bg-zinc-950/40 dark:hover:bg-zinc-950/60'
                      : 'hover:bg-zinc-50 dark:hover:bg-zinc-950/40',
                    isClosed && 'opacity-70',
                  )}
                >
                  <StateIcon
                    className={clsx('mt-0.5 size-4 shrink-0', stateColor)}
                    aria-label={isClosed ? 'Closed' : 'Open'}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                      <Link
                        to={`/entry/${f.id}`}
                        state={fromKey ? { from: fromKey } : undefined}
                        className={clsx(
                          'font-medium hover:underline focus:outline-none focus:underline',
                          isClosed
                            ? 'text-zinc-500 dark:text-zinc-400 line-through decoration-zinc-400/60'
                            : 'text-zinc-900 dark:text-zinc-50',
                        )}
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
                      {!lockedEntryType && f.entryType && f.entryType.value !== 'other' && (
                        <EntryTypeBadge entryType={f.entryType} appliedByAI={f.entryTypeAppliedByAI} />
                      )}
                      {f.status && <StatusBadge status={f.status} />}
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
                        {isClosed ? 'closed' : 'opened'}{' '}
                        {dayjs(f.createdAt).fromNow()} by{' '}
                        <span className="text-zinc-700 dark:text-zinc-300">
                          {creatorLabel(f.creator)}
                        </span>
                      </span>
                    </div>
                  </div>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}
