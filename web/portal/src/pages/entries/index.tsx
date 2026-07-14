import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ChevronUp,
  LifeBuoy,
  Loader2,
  MessageSquare,
  MessageSquarePlus,
  Search,
  X,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import dayjs from 'dayjs'
import clsx from 'clsx'

import Button from '@/components/button'
import EntryTypeBadge from '@/components/entry-type-badge'
import { markdownToPlainText } from '@/components/markdown'
import ReleaseBadge from '@/components/release-badge'
import StatusBadge from '@/components/status-badge'
import {
  API,
  type ApiEntryStatusValue,
  type ApiEntryTypeValue,
} from '@/services/api'
import { useAppSelector } from '@/store/hooks'
import { useSiteConfig } from '@/store/site/hooks'
import { entryStatusInfo } from '@/utils/entry-status'
import { message } from '@/utils/helpers'
import { userDisplayName } from '@/utils/user-display'

type Tab = 'all' | 'feature-request' | 'bug' | 'mine'
type Sort = 'top' | 'new'

export default function EntriesPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const me = useAppSelector((s) => s.auth.user)
  const siteConfig = useSiteConfig()
  const showSupportButton =
    !!siteConfig?.supportRequestEnabled && !!siteConfig?.supportRequestUrl
  const [tab, setTab] = useState<Tab>('all')
  const [sort, setSort] = useState<Sort>('new')
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  useEffect(() => {
    const tm = setTimeout(() => setSearch(searchInput.trim()), 250)
    return () => clearTimeout(tm)
  }, [searchInput])
  useEffect(() => {
    if (!me && tab === 'mine') setTab('all')
  }, [me, tab])

  const entryTypeFilter: ApiEntryTypeValue | undefined =
    tab === 'feature-request' ? 'feature-request' : tab === 'bug' ? 'bug' : undefined
  // Mine shows everything the user has submitted, closed entries
  // included. The All / Feature Requests / Bugs tabs hide closed
  // entries so the list reads as "what's still in flight".
  const openOnly = tab !== 'mine'

  const { data, isLoading, error } = useQuery({
    queryKey: ['portal', 'entries', { tab, sort, search }],
    queryFn: () =>
      API().portal.listEntries({
        sort,
        search: search || undefined,
        limit: 25,
        mine: tab === 'mine' ? true : undefined,
        entryType: entryTypeFilter,
        openOnly: openOnly ? true : undefined,
      }),
  })

  const voteMut = useMutation({
    mutationFn: (id: string) => API().portal.addVote(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['portal', 'entries'] })
    },
    onError: (e) => message(e),
  })
  const pendingVoteId = voteMut.isPending ? (voteMut.variables as string | undefined) : undefined

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-3">
          <h1 className="text-xl sm:text-2xl font-semibold">{t('entries.pageTitle')}</h1>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Link to="/new">
            <Button>
              <MessageSquarePlus className="size-4" />
              {t('entries.newFeedback')}
            </Button>
          </Link>
          {showSupportButton && (
            <a
              href={siteConfig!.supportRequestUrl}
              target="_blank"
              rel="noopener noreferrer"
            >
              <Button>
                <LifeBuoy className="size-4" />
                {t('entries.newSupportRequest')}
              </Button>
            </a>
          )}
        </div>
      </div>

      <div className="relative w-full sm:w-80">
        <Search className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-zinc-400" />
        <input
          type="search"
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          placeholder={t('entries.searchPlaceholder')}
          className="block w-full h-9 rounded-lg border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 pl-8 pr-8 text-sm transition-colors focus:border-brand-500 focus:ring-2 focus:ring-brand-200 dark:focus:ring-brand-500/30 focus:outline-none"
        />
        {searchInput && (
          <button
            type="button"
            onClick={() => setSearchInput('')}
            aria-label={t('entries.clearSearch')}
            className="absolute right-2 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200"
          >
            <X className="size-4" />
          </button>
        )}
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <TabSwitcher tab={tab} onChange={setTab} showMine={!!me} />
        <SortSwitcher sort={sort} onChange={setSort} />
      </div>

      {isLoading && <p className="text-zinc-500">{t('common.loading')}</p>}
      {error && <p className="text-rose-600">{(error as Error).message}</p>}

      {data && data.data.length === 0 && (
        <div className="rounded-xl border border-dashed border-zinc-300 dark:border-zinc-700 bg-white/40 dark:bg-zinc-900/40 p-12 text-center text-zinc-500">
          {search ? t('entries.noMatch', { query: search }) : emptyStateCopy(tab, t)}
        </div>
      )}

      {data && data.data.length > 0 && (
        <ul className="space-y-3">
          {data.data.map((f) => {
            const isMine = !!me && f.creator?.id === me.id
            const isClosed = entryStatusInfo(f.status?.value as ApiEntryStatusValue | undefined).closed
            const isPending = pendingVoteId === f.id
            return (
              <li
                key={f.id}
                className={clsx(
                  'flex items-start gap-3 sm:gap-4 rounded-xl border p-3 sm:p-4 shadow-soft transition-all duration-150',
                  isClosed
                    ? 'border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/40 opacity-75 hover:opacity-100 hover:border-zinc-300 dark:hover:border-zinc-700'
                    : 'border-zinc-200/70 dark:border-zinc-800 bg-white dark:bg-zinc-900 hover:border-zinc-300 hover:shadow-pop dark:hover:border-zinc-700',
                )}
              >
                <button
                  type="button"
                  aria-pressed={f.isVoted}
                  aria-busy={isPending}
                  disabled={isPending}
                  onClick={() => voteMut.mutate(f.id)}
                  className={clsx(
                    'hidden sm:flex shrink-0 self-start flex-col items-center justify-center rounded-lg border w-12 h-12 transition-all duration-150',
                    f.isVoted
                      ? 'border-brand-500 bg-brand-50 text-brand-700 dark:border-brand-400 dark:bg-brand-500/10 dark:text-brand-300 hover:bg-brand-100 dark:hover:bg-brand-500/20'
                      : 'border-zinc-300 dark:border-zinc-700 text-zinc-700 dark:text-zinc-300 hover:border-brand-400 hover:text-brand-700 hover:bg-brand-50 dark:hover:bg-brand-500/10',
                    isPending && 'cursor-wait',
                  )}
                >
                  <span className="text-base font-bold leading-none">{f.voteCount}</span>
                  <span className="mt-1 flex items-center gap-0.5 text-[9px] font-medium uppercase tracking-wide leading-none">
                    {isPending ? (
                      <Loader2 className="size-2.5 animate-spin" />
                    ) : (
                      <ChevronUp className="size-2.5" />
                    )}
                    {t('entries.voteCta')}
                  </span>
                </button>
                <div className="min-w-0 flex-1">
                  <div className="flex items-start gap-2">
                    <Link
                      to={`/entry/${f.id}`}
                      className={clsx(
                        'min-w-0 flex-1 font-medium hover:underline focus:outline-none focus:underline',
                        isClosed && 'text-zinc-500 dark:text-zinc-400',
                      )}
                    >
                      {f.title}
                    </Link>
                    {isMine && (
                      <span className="shrink-0 text-[10px] uppercase tracking-wide rounded-md px-1.5 py-0.5 border border-brand-300 dark:border-brand-500/40 bg-brand-50 dark:bg-brand-500/10 text-brand-700 dark:text-brand-300 font-medium">
                        {t('common.yours')}
                      </span>
                    )}
                  </div>
                  {(f.entryType?.value && f.entryType.value !== 'other') ||
                  f.status ||
                  f.release ? (
                    <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
                      {f.entryType && f.entryType.value !== 'other' && (
                        <EntryTypeBadge entryType={f.entryType} />
                      )}
                      {f.status && <StatusBadge status={f.status} />}
                      {f.release && <ReleaseBadge release={f.release} size="sm" />}
                    </div>
                  ) : null}
                  {f.description && (
                    <p className="mt-2 line-clamp-1 text-sm text-zinc-600 dark:text-zinc-400">
                      {markdownToPlainText(f.description)}
                    </p>
                  )}
                  <div className="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-zinc-500">
                    <span className="text-zinc-700 dark:text-zinc-300">
                      {t('common.byUser', { name: userDisplayName(f.creator, t('common.anonymous')) })}
                    </span>
                    <span>·</span>
                    <span>{dayjs(f.createdAt).format('LL')}</span>
                    <span>·</span>
                    <span className="inline-flex items-center gap-1">
                      <MessageSquare className="size-3.5" />
                      {f.commentCount}
                    </span>
                  </div>
                  <button
                    type="button"
                    aria-pressed={f.isVoted}
                    aria-busy={isPending}
                    disabled={isPending}
                    onClick={() => voteMut.mutate(f.id)}
                    className={clsx(
                      'mt-3 sm:hidden inline-flex items-center gap-1.5 self-start rounded-full border px-3 py-1 text-xs font-semibold uppercase tracking-wide transition-all duration-150',
                      f.isVoted
                        ? 'border-brand-500 bg-brand-50 text-brand-700 dark:border-brand-400 dark:bg-brand-500/10 dark:text-brand-300'
                        : 'border-zinc-300 dark:border-zinc-700 text-zinc-700 dark:text-zinc-300 active:bg-brand-50 dark:active:bg-brand-500/10',
                      isPending && 'cursor-wait',
                    )}
                  >
                    {isPending ? (
                      <Loader2 className="size-3.5 animate-spin" />
                    ) : (
                      <ChevronUp className="size-3.5" />
                    )}
                    <span>{t('entries.voteCta')}</span>
                    <span className="text-zinc-500 dark:text-zinc-400">·</span>
                    <span>{f.voteCount}</span>
                  </button>
                </div>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}

function TabSwitcher({
  tab,
  onChange,
  showMine,
}: {
  tab: Tab
  onChange: (next: Tab) => void
  showMine: boolean
}) {
  const { t } = useTranslation()
  const tabs: { value: Tab; label: string }[] = [
    { value: 'all', label: t('entries.tabAll') },
    { value: 'feature-request', label: t('entries.tabFeatureRequests') },
    { value: 'bug', label: t('entries.tabBugs') },
  ]
  if (showMine) tabs.push({ value: 'mine', label: t('entries.tabMine') })
  return (
    <div className="inline-flex flex-wrap rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-0.5 text-sm shadow-soft">
      {tabs.map((tt) => (
        <button
          key={tt.value}
          type="button"
          onClick={() => onChange(tt.value)}
          className={clsx(
            'px-3 py-1.5 rounded-md font-medium transition-all duration-150',
            tab === tt.value
              ? 'bg-brand-50 text-brand-700 dark:bg-brand-500/15 dark:text-brand-300'
              : 'text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300',
          )}
        >
          {tt.label}
        </button>
      ))}
    </div>
  )
}

function SortSwitcher({
  sort,
  onChange,
}: {
  sort: Sort
  onChange: (next: Sort) => void
}) {
  const { t } = useTranslation()
  const options: { value: Sort; label: string }[] = [
    { value: 'new', label: t('entries.sortNewest') },
    { value: 'top', label: t('entries.sortTopVotes') },
  ]
  return (
    <div className="inline-flex flex-wrap rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-0.5 text-sm shadow-soft">
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          onClick={() => onChange(o.value)}
          className={clsx(
            'px-3 py-1.5 rounded-md font-medium transition-all duration-150',
            sort === o.value
              ? 'bg-brand-50 text-brand-700 dark:bg-brand-500/15 dark:text-brand-300'
              : 'text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300',
          )}
        >
          {o.label}
        </button>
      ))}
    </div>
  )
}

type TFunc = ReturnType<typeof useTranslation>['t']

function emptyStateCopy(tab: Tab, t: TFunc): string {
  switch (tab) {
    case 'mine':
      return t('entries.emptyMine')
    case 'feature-request':
      return t('entries.emptyFeatureRequests')
    case 'bug':
      return t('entries.emptyBugs')
    default:
      return t('entries.emptyDefault')
  }
}
