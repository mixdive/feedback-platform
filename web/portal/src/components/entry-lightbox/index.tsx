import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronUp, ExternalLink, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import dayjs from 'dayjs'
import clsx from 'clsx'

import EntryTypeBadge from '@/components/entry-type-badge'
import Comments from '@/components/comments'
import Markdown from '@/components/markdown'
import StatusBadge from '@/components/status-badge'
import { API, type ApiEntryCreator } from '@/services/api'
import { message } from '@/utils/helpers'

function creatorLabel(c: ApiEntryCreator | undefined, fallback: string): string {
  if (!c) return fallback
  return c.name || c.username || fallback
}

interface Props {
  entryId: string | null
  onClose: () => void
}

export default function EntryLightbox({ entryId, onClose }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const isOpen = !!entryId

  const { data: entry, isLoading, error } = useQuery({
    queryKey: ['portal', 'entries', entryId ?? ''],
    queryFn: () => API().portal.getEntry(entryId ?? ''),
    enabled: !!entryId,
  })

  const voteMut = useMutation({
    mutationFn: () => API().portal.addVote(entryId ?? ''),
    onSuccess: (updated) => {
      queryClient.setQueryData(['portal', 'entries', entryId ?? ''], updated)
      void queryClient.invalidateQueries({ queryKey: ['portal', 'entries'] })
    },
    onError: (e) => message(e),
  })

  return (
    <Dialog open={isOpen} onClose={onClose} className="relative z-50">
      <div className="fixed inset-0 bg-zinc-950/60 backdrop-blur-sm" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className="flex max-h-[90vh] w-full max-w-2xl flex-col overflow-hidden rounded-xl bg-white dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 shadow-2xl">
          <div className="flex items-start gap-3 border-b border-zinc-200 dark:border-zinc-800 p-5">
            <DialogTitle className="flex-1 text-lg font-semibold leading-snug">
              {entry?.title ?? (isLoading ? t('common.loading') : t('entryLightbox.fallbackTitle'))}
            </DialogTitle>
            {entry && (
              <a
                href={`/entry/${entry.id}`}
                title={t('entryLightbox.openFullView')}
                className="rounded-md p-1.5 text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800"
              >
                <ExternalLink className="size-4" />
              </a>
            )}
            <button
              type="button"
              onClick={onClose}
              aria-label={t('entryLightbox.close')}
              className="rounded-md p-1.5 text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800"
            >
              <X className="size-4" />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-5">
            {isLoading && <p className="text-zinc-500">{t('common.loading')}</p>}
            {error && (
              <p className="text-rose-600">
                {(error as Error).message || t('entryLightbox.loadFailed')}
              </p>
            )}
            {entry && (
              <div className="space-y-5">
                <div className="flex items-start gap-4">
                  <button
                    type="button"
                    aria-pressed={entry.isVoted}
                    disabled={voteMut.isPending}
                    onClick={() => voteMut.mutate()}
                    className={clsx(
                      'flex shrink-0 flex-col items-center justify-center rounded-md border w-12 h-12 transition-colors',
                      entry.isVoted
                        ? 'border-sky-500 bg-sky-50 text-sky-700 dark:border-sky-400 dark:bg-sky-500/10 dark:text-sky-300 hover:bg-sky-100 dark:hover:bg-sky-500/20'
                        : 'border-zinc-300 dark:border-zinc-700 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800',
                      voteMut.isPending && 'opacity-50 cursor-not-allowed',
                    )}
                  >
                    <span className="text-base font-bold leading-none">{entry.voteCount}</span>
                    <span className="mt-1 flex items-center gap-0.5 text-[9px] font-medium uppercase tracking-wide leading-none">
                      <ChevronUp className="size-2.5" />
                      {t('entries.voteCta')}
                    </span>
                  </button>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      {entry.status && <StatusBadge status={entry.status} />}
                      {entry.entryType && <EntryTypeBadge entryType={entry.entryType} />}
                      <span className="text-xs text-zinc-500">
                        <span className="text-zinc-700 dark:text-zinc-300">
                          {t('common.byUser', { name: creatorLabel(entry.creator, t('common.anonymous')) })}
                        </span>{' '}
                        · {dayjs(entry.createdAt).format('LL')}
                      </span>
                    </div>
                    {entry.description ? (
                      <div className="mt-3">
                        <Markdown>{entry.description}</Markdown>
                      </div>
                    ) : (
                      <p className="mt-3 text-sm italic text-zinc-500">
                        {t('entryLightbox.noDescription')}
                      </p>
                    )}
                  </div>
                </div>

                <Comments entryId={entry.id} count={entry.commentCount} />
              </div>
            )}
          </div>
        </DialogPanel>
      </div>
    </Dialog>
  )
}
