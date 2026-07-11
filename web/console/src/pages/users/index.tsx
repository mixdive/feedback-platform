import { useEffect, useState } from 'react'
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { ArrowDown, ArrowUp, Search, X } from 'lucide-react'
import clsx from 'clsx'

import {
  API,
  type ApiSortDirection,
  type ApiUserRow,
  type ApiUserSort,
} from '@/services/api'

type SortKey =
  | 'alpha'
  | 'created'
  | 'entries'
  | 'feature-requests'
  | 'bugs'
  | 'others'

type ColumnConfig = {
  key: SortKey
  label: string
  align: 'left' | 'right'
  defaultDirection: ApiSortDirection
}

const columns: ColumnConfig[] = [
  { key: 'alpha', label: 'User', align: 'left', defaultDirection: 'asc' },
  { key: 'created', label: 'Joined', align: 'left', defaultDirection: 'desc' },
  { key: 'entries', label: 'Entries', align: 'right', defaultDirection: 'desc' },
  {
    key: 'feature-requests',
    label: 'Feature requests',
    align: 'right',
    defaultDirection: 'desc',
  },
  { key: 'bugs', label: 'Bugs', align: 'right', defaultDirection: 'desc' },
  { key: 'others', label: 'Others', align: 'right', defaultDirection: 'desc' },
]

const PAGE_LIMIT = 25

export default function UsersPage() {
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [sort, setSort] = useState<SortKey>('alpha')
  const [direction, setDirection] = useState<ApiSortDirection>('asc')
  const [page, setPage] = useState(1)

  // Debounce free-text search and reset to page 1 when the active query
  // (or sort) changes so the user never lands on an empty trailing page.
  useEffect(() => {
    const t = setTimeout(() => setSearch(searchInput.trim()), 250)
    return () => clearTimeout(t)
  }, [searchInput])
  useEffect(() => {
    setPage(1)
  }, [search, sort, direction])

  const { data, isLoading, error, isFetching } = useQuery({
    queryKey: ['console', 'users', { search, sort, direction, page }],
    queryFn: () =>
      API().console.listUsers({
        search: search || undefined,
        sort: sort as ApiUserSort,
        direction,
        page,
        limit: PAGE_LIMIT,
      }),
    placeholderData: keepPreviousData,
  })

  const onHeaderClick = (col: ColumnConfig) => {
    if (col.key === sort) {
      setDirection((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSort(col.key)
      setDirection(col.defaultDirection)
    }
  }

  const rows = data?.data ?? []
  const total = data?.meta.total ?? 0
  const hasMore = data?.meta.hasMore ?? false
  const showingFrom = rows.length === 0 ? 0 : (page - 1) * PAGE_LIMIT + 1
  const showingTo = (page - 1) * PAGE_LIMIT + rows.length

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-semibold">Users</h1>
        <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
          Every account in the system — administrators, editors, and portal
          visitors — with how many entries each has submitted broken down by
          category.
        </p>
      </div>

      <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 overflow-hidden">
        <div className="flex flex-wrap items-center gap-3 border-b border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/40 px-3 py-2">
          <div className="relative flex-1 min-w-[12rem] max-w-xl">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-zinc-400" />
            <input
              type="search"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              placeholder="Search by name, username, or email"
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
          <div className="text-xs text-zinc-500 tabular-nums">
            {isLoading
              ? 'Loading…'
              : total === 0
                ? 'No users'
                : `${total.toLocaleString()} user${total === 1 ? '' : 's'}`}
          </div>
        </div>

        {error && (
          <p className="p-4 text-sm text-rose-600">{(error as Error).message}</p>
        )}

        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-zinc-50 dark:bg-zinc-950/40 text-xs uppercase tracking-wide text-zinc-500">
              <tr>
                {columns.map((col) => {
                  const active = sort === col.key
                  return (
                    <th
                      key={col.key}
                      scope="col"
                      className={clsx(
                        'px-4 py-2 font-medium select-none',
                        col.align === 'right' ? 'text-right' : 'text-left',
                      )}
                    >
                      <button
                        type="button"
                        onClick={() => onHeaderClick(col)}
                        className={clsx(
                          'inline-flex items-center gap-1 hover:text-zinc-800 dark:hover:text-zinc-200',
                          active && 'text-zinc-900 dark:text-zinc-50',
                          col.align === 'right' && 'flex-row-reverse',
                        )}
                      >
                        <span>{col.label}</span>
                        {active && (
                          direction === 'asc' ? (
                            <ArrowUp className="size-3" />
                          ) : (
                            <ArrowDown className="size-3" />
                          )
                        )}
                      </button>
                    </th>
                  )
                })}
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800">
              {rows.map((u) => (
                <UserRow key={u.id} user={u} />
              ))}
              {!isLoading && rows.length === 0 && (
                <tr>
                  <td
                    colSpan={columns.length}
                    className="px-4 py-8 text-center text-sm text-zinc-500"
                  >
                    {search
                      ? `No users match "${search}".`
                      : 'No users yet.'}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {total > PAGE_LIMIT && (
          <div className="flex items-center justify-between gap-3 border-t border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/40 px-3 py-2 text-xs text-zinc-500">
            <div className="tabular-nums">
              {showingFrom}–{showingTo} of {total.toLocaleString()}
              {isFetching && ' · refreshing…'}
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-1 disabled:opacity-40"
              >
                Previous
              </button>
              <button
                type="button"
                onClick={() => setPage((p) => p + 1)}
                disabled={!hasMore}
                className="rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-1 disabled:opacity-40"
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function UserRow({ user }: { user: ApiUserRow }) {
  const display = user.username?.trim() || user.name?.trim() || user.email || '—'
  const sub =
    user.name && user.email && user.name !== user.email ? user.email : ''
  return (
    <tr className="cursor-default">
      <td className="px-4 py-3">
        <div className="flex items-center gap-3">
          {user.imageUrl ? (
            <img
              src={user.imageUrl}
              alt=""
              className="size-7 rounded-full object-cover"
            />
          ) : (
            <div className="grid size-7 shrink-0 place-items-center rounded-full bg-zinc-200 dark:bg-zinc-800 text-[11px] font-semibold text-zinc-600 dark:text-zinc-300">
              {(display[0] || '?').toUpperCase()}
            </div>
          )}
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className="truncate text-sm font-medium" title={display}>
                {display}
              </span>
              {user.roles.length > 0 &&
                user.roles.map((r) => (
                  <span
                    key={r}
                    className="inline-flex shrink-0 items-center rounded-full border border-zinc-300 dark:border-zinc-700 bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-zinc-600 dark:text-zinc-300"
                  >
                    {r}
                  </span>
                ))}
              {user.isBlocked && (
                <span className="inline-flex shrink-0 items-center rounded-full border border-rose-300 dark:border-rose-800 bg-rose-50 dark:bg-rose-500/10 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-rose-600 dark:text-rose-300">
                  Blocked
                </span>
              )}
            </div>
            {sub && (
              <div className="truncate text-xs text-zinc-500" title={sub}>
                {sub}
              </div>
            )}
          </div>
        </div>
      </td>
      <td className="px-4 py-3 text-sm text-zinc-500 whitespace-nowrap">
        {formatDate(user.createdAt)}
      </td>
      <td className="px-4 py-3 text-right text-sm tabular-nums">
        {user.entryCount.toLocaleString()}
      </td>
      <td className="px-4 py-3 text-right text-sm tabular-nums">
        {user.featureRequestCount.toLocaleString()}
      </td>
      <td className="px-4 py-3 text-right text-sm tabular-nums">
        {user.bugCount.toLocaleString()}
      </td>
      <td className="px-4 py-3 text-right text-sm tabular-nums">
        {user.otherCount.toLocaleString()}
      </td>
    </tr>
  )
}

function formatDate(iso: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}
