import { Link, useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, ChevronUp, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import dayjs from 'dayjs'
import clsx from 'clsx'

import Comments from '@/components/comments'
import Markdown from '@/components/markdown'
import ReleaseBadge from '@/components/release-badge'
import StatusBadge from '@/components/status-badge'
import { API, type ApiEntry, type ApiEntryCreator, type ApiEntryStatusValue } from '@/services/api'
import { authResolved } from '@/store'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { entryStatusInfo } from '@/utils/entry-status'
import { message } from '@/utils/helpers'

function creatorLabel(c: ApiEntryCreator | undefined, fallback: string): string {
  if (!c) return fallback
  return c.name || c.username || fallback
}

export default function EntryDetailPage() {
  const { t } = useTranslation()
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const dispatch = useAppDispatch()
  const me = useAppSelector((s) => s.auth.user)

  const { data: entry, isLoading, error } = useQuery({
    queryKey: ['portal', 'entries', id],
    queryFn: () => API().portal.getEntry(id),
    enabled: !!id,
  })

  const voteMut = useMutation({
    mutationFn: () => API().portal.addVote(id),
    onSuccess: (updated) => {
      queryClient.setQueryData(['portal', 'entries', id], updated)
      void queryClient.invalidateQueries({ queryKey: ['portal', 'entries'] })
      API()
        .auth.me()
        .then((res) => dispatch(authResolved(res.user)))
        .catch(() => {})
    },
    onError: (e) => message(e),
  })

  if (isLoading) {
    return <p className="text-zinc-500">{t('common.loading')}</p>
  }

  if (error || !entry) {
    return (
      <div className="space-y-4">
        <Link
          to="/"
          className="inline-flex items-center gap-2 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
        >
          <ArrowLeft className="size-4" />
          {t('entryDetail.backToEntries')}
        </Link>
        <p className="text-rose-600">
          {(error as Error | undefined)?.message ?? t('entryDetail.notFound')}
        </p>
      </div>
    )
  }

  const isMine = !!me && entry.creator?.id === me.id
  const isClosed = entryStatusInfo(entry.status?.value as ApiEntryStatusValue | undefined).closed

  return (
    <div className="space-y-6">
      <div className="border-b border-zinc-200 dark:border-zinc-800 pb-4">
        <button
          type="button"
          onClick={() => navigate('/')}
          className="inline-flex items-center gap-2 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
        >
          <ArrowLeft className="size-4" />
          {t('entryDetail.backToEntries')}
        </button>
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <h1
            className={clsx(
              'text-2xl font-semibold break-words',
              isClosed && 'text-zinc-500 dark:text-zinc-400',
            )}
          >
            {entry.title}
          </h1>
          {entry.status && <StatusBadge status={entry.status} />}
          {isMine && (
            <span className="shrink-0 text-[10px] uppercase tracking-wide rounded px-1.5 py-0.5 border border-sky-300 dark:border-sky-500/40 bg-sky-50 dark:bg-sky-500/10 text-sky-700 dark:text-sky-300">
              {t('common.yours')}
            </span>
          )}
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_280px]">
        <div className="min-w-0 space-y-6">
          <div
            className={clsx(
              'rounded-lg border p-6',
              isClosed
                ? 'border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/40'
                : 'border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900',
            )}
          >
            <div className="flex items-start gap-5">
              <div className="min-w-0 flex-1">
                {entry.description ? (
                  <Markdown>{entry.description}</Markdown>
                ) : (
                  <p className="text-sm italic text-zinc-500">{t('entryDetail.noDescription')}</p>
                )}
              </div>
              <button
                type="button"
                aria-pressed={entry.isVoted}
                aria-busy={voteMut.isPending}
                disabled={voteMut.isPending}
                onClick={() => voteMut.mutate()}
                className={clsx(
                  'flex shrink-0 self-start flex-col items-center justify-center rounded-md border w-14 h-14 transition-colors',
                  entry.isVoted
                    ? 'border-sky-500 bg-sky-50 text-sky-700 dark:border-sky-400 dark:bg-sky-500/10 dark:text-sky-300 hover:bg-sky-100 dark:hover:bg-sky-500/20'
                    : 'border-zinc-300 dark:border-zinc-700 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800',
                  voteMut.isPending && 'cursor-wait',
                )}
              >
                <span className="text-lg font-bold leading-none">{entry.voteCount}</span>
                <span className="mt-1 flex items-center gap-0.5 text-[9px] font-medium uppercase tracking-wide leading-none">
                  {voteMut.isPending ? (
                    <Loader2 className="size-3 animate-spin" />
                  ) : (
                    <ChevronUp className="size-3" />
                  )}
                  {t('entries.voteCta')}
                </span>
              </button>
            </div>
          </div>

          <Comments entryId={entry.id} count={entry.commentCount} />
        </div>

        <Sidebar entry={entry} isMine={isMine} />
      </div>
    </div>
  )
}

function Sidebar({ entry, isMine }: { entry: ApiEntry; isMine: boolean }) {
  const { t } = useTranslation()
  return (
    <aside className="space-y-4 lg:sticky lg:top-4 lg:self-start">
      <Section title={t('entryDetail.sectionStatus')}>
        <StatusBadge status={entry.status} />
      </Section>

      {entry.release && (
        <Section title={t('entryDetail.sectionRelease')}>
          <ReleaseBadge release={entry.release} />
          {entry.release.title && (
            <p className="mt-2 text-xs text-zinc-500">{entry.release.title}</p>
          )}
        </Section>
      )}

      <Section title={t('entryDetail.sectionSubmitter')}>
        <CreatorBlock creator={entry.creator} isMine={isMine} />
      </Section>

      <Section title={t('entryDetail.sectionVotes')}>
        <div className="text-sm">
          <span className="font-semibold">{entry.voteCount}</span>{' '}
          <span className="text-zinc-500">
            {t('entryDetail.voteCountLabel', { count: entry.voteCount })}
          </span>
        </div>
      </Section>

      <Section title={t('entryDetail.sectionComments')}>
        <div className="text-sm">
          <span className="font-semibold">{entry.commentCount}</span>{' '}
          <span className="text-zinc-500">
            {t('entryDetail.commentCountLabel', { count: entry.commentCount })}
          </span>
        </div>
      </Section>

      <Section title={t('entryDetail.sectionSubmitted')}>
        <div className="text-sm text-zinc-700 dark:text-zinc-300" title={entry.createdAt}>
          {dayjs(entry.createdAt).format('LL')}
        </div>
        <div className="text-xs text-zinc-500">
          {dayjs(entry.createdAt).format('LT')}
        </div>
      </Section>

      {entry.updatedAt && entry.updatedAt !== entry.createdAt && (
        <Section title={t('entryDetail.sectionUpdated')}>
          <div className="text-sm text-zinc-700 dark:text-zinc-300" title={entry.updatedAt}>
            {dayjs(entry.updatedAt).format('LL')}
          </div>
          <div className="text-xs text-zinc-500">
            {dayjs(entry.updatedAt).format('LT')}
          </div>
        </Section>
      )}
    </aside>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-3">
      <div className="text-xs font-medium uppercase tracking-wide text-zinc-500">
        {title}
      </div>
      <div className="mt-2">{children}</div>
    </div>
  )
}

function CreatorBlock({
  creator,
  isMine,
}: {
  creator?: ApiEntryCreator
  isMine: boolean
}) {
  const { t } = useTranslation()
  if (!creator) {
    return <span className="text-sm text-zinc-500">{t('common.anonymous')}</span>
  }
  const initial = (creator.name || creator.username || '?')[0]?.toUpperCase() ?? '?'
  return (
    <div className="flex items-center gap-2">
      {creator.imageUrl ? (
        <img src={creator.imageUrl} alt="" className="size-7 rounded-full object-cover" />
      ) : (
        <div className="grid size-7 shrink-0 place-items-center rounded-full bg-zinc-200 dark:bg-zinc-800 text-xs font-semibold text-zinc-600 dark:text-zinc-300">
          {initial}
        </div>
      )}
      <div className="min-w-0 text-sm">
        <div className="truncate text-zinc-800 dark:text-zinc-200">
          {creatorLabel(creator, t('common.anonymous'))}
          {isMine && <span className="ml-1 text-xs text-sky-600 dark:text-sky-400">{t('common.you')}</span>}
        </div>
        {creator.username && creator.name && (
          <div className="truncate text-xs text-zinc-500">@{creator.username}</div>
        )}
      </div>
    </div>
  )
}
